package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/derangedhermits/website/internal/db"
	"github.com/derangedhermits/website/internal/mail"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type SubscribeHandler struct {
	DB        *sql.DB
	Templates *template.Template
	BaseURL   string
	Mailer    *mail.Mailer
}

func (h *SubscribeHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	name := strings.TrimSpace(r.FormValue("name"))

	if email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}
	if len(email) > 254 {
		http.Error(w, "Email too long", http.StatusBadRequest)
		return
	}
	if !emailRegex.MatchString(email) {
		http.Error(w, "Invalid email address", http.StatusBadRequest)
		return
	}
	if len(name) > 100 {
		http.Error(w, "Name too long", http.StatusBadRequest)
		return
	}

	sub, err := db.CreateSubscriber(h.DB, email, name)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Send confirmation email (non-blocking — don't leak whether the email already existed)
	if sub != nil {
		go func() {
			if err := h.sendConfirmationEmail(sub); err != nil {
				slog.Error("Failed to send confirmation email", "email", sub.Email, "error", err)
			}
		}()
	}

	data := map[string]interface{}{
		"Success":       true,
		"AlreadyExists": sub == nil,
	}

	// If this is an HTMX or fetch request, return just the result fragment
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		h.Templates.ExecuteTemplate(w, "subscribe_result.html", data)
		return
	}

	// Otherwise redirect back with a query param
	http.Redirect(w, r, "/?subscribed=1", http.StatusSeeOther)
}

func (h *SubscribeHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Missing token", http.StatusBadRequest)
		return
	}

	err := db.ConfirmSubscriber(h.DB, token)

	data := map[string]interface{}{}
	if err != nil {
		data["ConfirmError"] = true
	} else {
		data["Confirmed"] = true
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.Templates.ExecuteTemplate(w, "layout", data)
}

func (h *SubscribeHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Missing token", http.StatusBadRequest)
		return
	}

	err := db.UnsubscribeByToken(h.DB, token)
	if err != nil {
		http.Error(w, "Invalid or expired token", http.StatusNotFound)
		return
	}

	data := map[string]interface{}{
		"Unsubscribed": true,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.Templates.ExecuteTemplate(w, "layout", data)
}

func (h *SubscribeHandler) sendConfirmationEmail(sub *db.Subscriber) error {
	confirmURL := fmt.Sprintf("%s/confirm?token=%s", h.BaseURL, sub.Token)
	body := fmt.Sprintf(`
		<div style="font-family:sans-serif;max-width:600px;margin:0 auto;background:#1a1a2e;color:#e0e0e0;padding:24px;border-radius:8px;">
			<h1 style="color:#4ecca3;">Confirm Your Subscription</h1>
			<p>Hey%s! Thanks for signing up for The Deranged Hermits event notifications.</p>
			<p>Please confirm your email address by clicking the button below:</p>
			<p style="text-align:center;margin:32px 0;">
				<a href="%s" style="background:#4ecca3;color:#1a1a2e;padding:12px 24px;border-radius:6px;text-decoration:none;font-weight:bold;">Confirm Subscription</a>
			</p>
			<p style="font-size:12px;color:#888;">If you didn't subscribe, you can safely ignore this email.</p>
		</div>
	`, nameGreeting(sub.Name), confirmURL)

	return h.Mailer.Send(sub.Email, "Confirm your subscription — The Deranged Hermits", body)
}

func nameGreeting(name string) string {
	if name == "" {
		return ""
	}
	return " " + name
}
