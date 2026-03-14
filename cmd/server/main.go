package main

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/derangedhermits/website/internal/config"
	"github.com/derangedhermits/website/internal/db"
	"github.com/derangedhermits/website/internal/handlers"
	"github.com/derangedhermits/website/internal/mail"
	"github.com/derangedhermits/website/internal/middleware"
)

func main() {
	// Structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()

	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	mailer := mail.New(cfg)

	// Parse each page template separately with the layout to avoid
	// conflicting "content" block definitions.
	layoutFile := filepath.Join("templates", "layout.html")
	parsePage := func(files ...string) *template.Template {
		paths := []string{layoutFile}
		for _, f := range files {
			paths = append(paths, filepath.Join("templates", f))
		}
		t, err := template.ParseFiles(paths...)
		if err != nil {
			slog.Error("Failed to parse template", "files", files, "error", err)
			os.Exit(1)
		}
		return t
	}

	homeTmpl := parsePage("home.html", "subscribe_result.html")
	eventsTmpl := parsePage("events.html")

	// Admin templates are standalone (no layout)—parse them individually.
	adminLoginTmpl := template.Must(template.ParseFiles(filepath.Join("templates", "admin_login.html")))
	adminTmpl := template.Must(template.ParseFiles(filepath.Join("templates", "admin.html")))
	adminEventFormTmpl := template.Must(template.ParseFiles(filepath.Join("templates", "admin_event_form.html")))
	adminInviteTmpl := template.Must(template.ParseFiles(filepath.Join("templates", "admin_invite.html")))

	homeH := &handlers.HomeHandler{DB: database, Templates: homeTmpl}
	eventsH := &handlers.EventsHandler{DB: database, Templates: eventsTmpl}
	subH := &handlers.SubscribeHandler{DB: database, Templates: homeTmpl, BaseURL: cfg.BaseURL, Mailer: mailer}
	adminH := &handlers.AdminHandler{
		DB:                database,
		LoginTemplate:     adminLoginTmpl,
		DashTemplate:      adminTmpl,
		EventFormTemplate: adminEventFormTmpl,
		InviteTemplate:    adminInviteTmpl,
		Mailer:            mailer,
		BaseURL:           cfg.BaseURL,
		SessionSecret:     cfg.SessionSecret,
	}
	apiH := &handlers.APIHandler{DB: database, Mailer: mailer, BaseURL: cfg.BaseURL}

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Compress(5))

	// Rate limiters
	loginRL := middleware.RateLimit(10, 1*time.Minute)
	subscribeRL := middleware.RateLimit(5, 1*time.Minute)
	apiRL := middleware.RateLimit(60, 1*time.Minute)

	// CSRF protection for browser routes
	csrfMW := middleware.CSRF(cfg.SessionSecret)

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Health check (no CSRF, no auth)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := database.Ping(); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Public browser routes (with CSRF)
	r.Group(func(r chi.Router) {
		r.Use(csrfMW)

		r.Get("/", homeH.ServeHTTP)
		r.Get("/events", eventsH.List)
		r.Get("/events/{id}/ical", eventsH.ICal)
		r.With(subscribeRL).Post("/subscribe", subH.Subscribe)
		r.Get("/confirm", subH.Confirm)
		r.Get("/unsubscribe", subH.Unsubscribe)

		r.With(loginRL).Get("/admin/login", adminH.LoginPage)
		r.With(loginRL).Post("/admin/login", adminH.Login)

		// Invite accept (public, no auth required)
		r.Get("/admin/invite/accept", adminH.AcceptInviteForm)
		r.Post("/admin/invite/accept", adminH.AcceptInvite)

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(database, cfg.SessionSecret))

			r.Get("/admin", adminH.Dashboard)
			r.Post("/admin/logout", adminH.Logout)

			r.Get("/admin/events/new", adminH.NewEventForm)
			r.Post("/admin/events", adminH.CreateEvent)
			r.Get("/admin/events/{id}/edit", adminH.EditEventForm)
			r.Post("/admin/events/{id}", adminH.UpdateEvent)
			r.Post("/admin/events/{id}/delete", adminH.DeleteEvent)
			r.Post("/admin/events/{id}/notify", adminH.NotifySubscribers)

			r.Post("/admin/subscribers/{id}/delete", adminH.DeleteSubscriber)

			r.Post("/admin/users/invite", adminH.InviteUser)
			r.Post("/admin/users/{id}/delete", adminH.DeleteUser)
			r.Post("/admin/password", adminH.ChangePassword)
		})
	})

	r.Route("/api", func(r chi.Router) {
		r.Use(apiRL)
		r.Use(middleware.RequireAPIKey(cfg.APIKey))

		r.Get("/events", apiH.ListEvents)
		r.Get("/events/{id}", apiH.GetEvent)
		r.Post("/events", apiH.CreateEvent)
		r.Put("/events/{id}", apiH.UpdateEvent)
		r.Delete("/events/{id}", apiH.DeleteEvent)
		r.Post("/events/{id}/notify", apiH.NotifySubscribers)
		r.Get("/subscribers", apiH.ListSubscribers)
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("Starting server", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-done
	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Forced shutdown", "error", err)
		os.Exit(1)
	}
	slog.Info("Server stopped gracefully")
}
