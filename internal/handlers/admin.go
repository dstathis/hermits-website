package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/derangedhermits/website/internal/db"
	"github.com/derangedhermits/website/internal/mail"
	"github.com/derangedhermits/website/internal/middleware"
)

type AdminHandler struct {
	DB                *sql.DB
	LoginTemplate     *template.Template
	DashTemplate      *template.Template
	EventFormTemplate *template.Template
	InviteTemplate    *template.Template
	Mailer            *mail.Mailer
	BaseURL           string
	SessionSecret     string
}

// GET /admin/login
func (h *AdminHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"CSRFField": middleware.CSRFTemplateField(r),
		"Flash":     r.URL.Query().Get("flash"),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.LoginTemplate.Execute(w, data)
}

// POST /admin/login
func (h *AdminHandler) Login(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	user, err := db.Authenticate(h.DB, username, password)
	if err != nil || user == nil {
		data := map[string]interface{}{"Error": "Invalid username or password", "CSRFField": middleware.CSRFTemplateField(r)}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		h.LoginTemplate.Execute(w, data)
		return
	}

	session, err := db.CreateSession(h.DB, user.ID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	signedID := middleware.SignSessionID(session.ID, h.SessionSecret)
	secure := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    signedID,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// POST /admin/logout
func (h *AdminHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		db.DeleteSession(h.DB, cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

// GET /admin
func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	events, err := db.GetAllEvents(h.DB)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	subs, err := db.GetAllSubscribers(h.DB)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	users, err := db.GetAllAdminUsers(h.DB)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Events":        events,
		"Subscribers":   subs,
		"AdminUsers":    users,
		"CurrentUserID": r.Context().Value(middleware.UserIDKey),
		"Flash":         r.URL.Query().Get("flash"),
		"CSRFField":     middleware.CSRFTemplateField(r),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.DashTemplate.Execute(w, data)
}

// GET /admin/events/new
func (h *AdminHandler) NewEventForm(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Event":     &db.Event{},
		"IsNew":     true,
		"CSRFField": middleware.CSRFTemplateField(r),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.EventFormTemplate.Execute(w, data)
}

// GET /admin/events/{id}/edit
func (h *AdminHandler) EditEventForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	event, err := db.GetEventByID(h.DB, id)
	if err != nil || event == nil {
		http.NotFound(w, r)
		return
	}
	data := map[string]interface{}{
		"Event":     event,
		"IsNew":     false,
		"CSRFField": middleware.CSRFTemplateField(r),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.EventFormTemplate.Execute(w, data)
}

// POST /admin/events
func (h *AdminHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	event, err := parseEventForm(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := db.CreateEvent(h.DB, event); err != nil {
		http.Error(w, "Failed to create event", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin?flash=Event+created", http.StatusSeeOther)
}

// POST /admin/events/{id}
func (h *AdminHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	event, err := parseEventForm(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	event.ID = id
	if err := db.UpdateEvent(h.DB, event); err != nil {
		http.Error(w, "Failed to update event", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin?flash=Event+updated", http.StatusSeeOther)
}

// POST /admin/events/{id}/delete
func (h *AdminHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := db.DeleteEvent(h.DB, id); err != nil {
		http.Error(w, "Failed to delete event", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin?flash=Event+deleted", http.StatusSeeOther)
}

// POST /admin/events/{id}/notify
func (h *AdminHandler) NotifySubscribers(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	event, err := db.GetEventByID(h.DB, id)
	if err != nil || event == nil {
		http.NotFound(w, r)
		return
	}

	subs, err := db.GetConfirmedSubscriberEmails(h.DB)
	if err != nil {
		http.Error(w, "Failed to get subscribers", http.StatusInternalServerError)
		return
	}

	if len(subs) == 0 {
		http.Redirect(w, r, "/admin?flash=No+subscribers+to+notify", http.StatusSeeOther)
		return
	}

	subject := fmt.Sprintf("New Event: %s", event.Title)

	// Send each subscriber a personalized email with their unsubscribe token
	var errCount int
	for _, sub := range subs {
		body := h.renderNotificationEmail(event, sub.Token)
		if err := h.Mailer.Send(sub.Email, subject, body); err != nil {
			errCount++
		}
	}

	if errCount > 0 {
		http.Redirect(w, r, fmt.Sprintf("/admin?flash=Sent+with+%d+errors", errCount), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin?flash=Notified+%d+subscribers", len(subs)), http.StatusSeeOther)
}

// POST /admin/subscribers/{id}/delete
func (h *AdminHandler) DeleteSubscriber(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := db.DeleteSubscriber(h.DB, id); err != nil {
		http.Error(w, "Failed to delete subscriber", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin?flash=Subscriber+removed", http.StatusSeeOther)
}

// POST /admin/users/invite
func (h *AdminHandler) InviteUser(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.FormValue("username"))
	email := strings.TrimSpace(r.FormValue("email"))

	if username == "" || email == "" {
		http.Redirect(w, r, "/admin?flash=Username+and+email+are+required", http.StatusSeeOther)
		return
	}
	if len(username) > 100 || len(email) > 254 {
		http.Redirect(w, r, "/admin?flash=Input+too+long", http.StatusSeeOther)
		return
	}

	token, err := db.InviteAdminUser(h.DB, username, email)
	if err != nil {
		http.Redirect(w, r, "/admin?flash=Failed+to+invite+user+(username+may+already+exist)", http.StatusSeeOther)
		return
	}

	// Send invite email
	go func() {
		inviteURL := fmt.Sprintf("%s/admin/invite/accept?token=%s", h.BaseURL, token)
		body := fmt.Sprintf(`
			<div style="font-family:sans-serif;max-width:600px;margin:0 auto;background:#1a1a2e;color:#e0e0e0;padding:24px;border-radius:8px;">
				<h1 style="color:#4ecca3;">You've Been Invited!</h1>
				<p>You've been invited to be an admin of <strong>The Deranged Hermits</strong> website.</p>
				<p>Your username is: <strong>%s</strong></p>
				<p style="text-align:center;margin:32px 0;">
					<a href="%s" style="background:#4ecca3;color:#1a1a2e;padding:12px 24px;border-radius:6px;text-decoration:none;font-weight:bold;">Set Your Password</a>
				</p>
				<p style="font-size:12px;color:#888;">If you weren't expecting this invite, you can safely ignore this email.</p>
			</div>
		`, username, inviteURL)
		h.Mailer.Send(email, "You're invited — The Deranged Hermits Admin", body)
	}()

	http.Redirect(w, r, "/admin?flash=Invite+sent+to+"+email, http.StatusSeeOther)
}

// POST /admin/users/{id}/delete
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	currentUserID, _ := r.Context().Value(middleware.UserIDKey).(string)
	if id != currentUserID {
		http.Redirect(w, r, "/admin?flash=You+can+only+remove+your+own+account", http.StatusSeeOther)
		return
	}
	if err := db.DeleteAdminUser(h.DB, id); err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}
	// Log out after self-deletion
	cookie, err := r.Cookie("session")
	if err == nil {
		db.DeleteSession(h.DB, cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

// POST /admin/password
func (h *AdminHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	currentUserID, _ := r.Context().Value(middleware.UserIDKey).(string)
	oldPass := r.FormValue("current_password")
	newPass := r.FormValue("new_password")
	confirmPass := r.FormValue("new_password_confirm")

	if len(newPass) < 8 {
		http.Redirect(w, r, "/admin?flash=New+password+must+be+at+least+8+characters", http.StatusSeeOther)
		return
	}
	if newPass != confirmPass {
		http.Redirect(w, r, "/admin?flash=New+passwords+do+not+match", http.StatusSeeOther)
		return
	}

	if err := db.ChangePassword(h.DB, currentUserID, oldPass, newPass); err != nil {
		http.Redirect(w, r, "/admin?flash=Incorrect+current+password", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin?flash=Password+changed", http.StatusSeeOther)
}

// GET /admin/invite/accept
func (h *AdminHandler) AcceptInviteForm(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Missing token", http.StatusBadRequest)
		return
	}
	user, err := db.GetAdminUserByInviteToken(h.DB, token)
	if err != nil || user == nil {
		http.Error(w, "Invalid or expired invite link", http.StatusNotFound)
		return
	}
	data := map[string]interface{}{
		"Token":     token,
		"Username":  user.Username,
		"CSRFField": middleware.CSRFTemplateField(r),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.InviteTemplate.Execute(w, data)
}

// POST /admin/invite/accept
func (h *AdminHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	password := r.FormValue("password")
	confirm := r.FormValue("password_confirm")

	if token == "" {
		http.Error(w, "Missing token", http.StatusBadRequest)
		return
	}
	if len(password) < 8 {
		user, _ := db.GetAdminUserByInviteToken(h.DB, token)
		uname := ""
		if user != nil {
			uname = user.Username
		}
		data := map[string]interface{}{
			"Token":     token,
			"Username":  uname,
			"Error":     "Password must be at least 8 characters",
			"CSRFField": middleware.CSRFTemplateField(r),
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		h.InviteTemplate.Execute(w, data)
		return
	}
	if password != confirm {
		user, _ := db.GetAdminUserByInviteToken(h.DB, token)
		uname := ""
		if user != nil {
			uname = user.Username
		}
		data := map[string]interface{}{
			"Token":     token,
			"Username":  uname,
			"Error":     "Passwords do not match",
			"CSRFField": middleware.CSRFTemplateField(r),
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		h.InviteTemplate.Execute(w, data)
		return
	}

	if err := db.AcceptInvite(h.DB, token, password); err != nil {
		http.Error(w, "Invalid or expired invite link", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, "/admin/login?flash=Password+set!+You+can+now+log+in.", http.StatusSeeOther)
}

func (h *AdminHandler) renderNotificationEmail(event *db.Event, unsubscribeToken string) string {
	loc := event.Location
	if event.LocationURL != "" {
		loc = fmt.Sprintf(`<a href="%s">%s</a>`, event.LocationURL, event.Location)
	}
	return fmt.Sprintf(`
		<div style="font-family:sans-serif;max-width:600px;margin:0 auto;background:#1a1a2e;color:#e0e0e0;padding:24px;border-radius:8px;">
			<h1 style="color:#4ecca3;">%s</h1>
			<p><strong>Format:</strong> %s</p>
			<p><strong>Date:</strong> %s</p>
			<p><strong>Location:</strong> %s</p>
			<p><strong>Entry:</strong> %s</p>
			<p>%s</p>
			<hr style="border-color:#333;">
			<p style="font-size:12px;color:#888;">You're receiving this because you subscribed to The Deranged Hermits event notifications.
			<a href="%s/unsubscribe?token=%s" style="color:#4ecca3;">Unsubscribe</a></p>
		</div>
	`, event.Title, event.Format, event.Date.Format("Monday, 2 January 2006 · 15:04"), loc, event.EntryFee, event.Description, h.BaseURL, unsubscribeToken)
}

func parseEventForm(r *http.Request) (*db.Event, error) {
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if len(title) > 200 {
		return nil, fmt.Errorf("title must be under 200 characters")
	}

	format := strings.TrimSpace(r.FormValue("format"))
	if len(format) > 100 {
		return nil, fmt.Errorf("format must be under 100 characters")
	}

	location := strings.TrimSpace(r.FormValue("location"))
	if len(location) > 200 {
		return nil, fmt.Errorf("location must be under 200 characters")
	}

	locationURL := strings.TrimSpace(r.FormValue("location_url"))
	if len(locationURL) > 500 {
		return nil, fmt.Errorf("location URL must be under 500 characters")
	}

	entryFee := strings.TrimSpace(r.FormValue("entry_fee"))
	if len(entryFee) > 50 {
		return nil, fmt.Errorf("entry fee must be under 50 characters")
	}

	description := strings.TrimSpace(r.FormValue("description"))
	if len(description) > 5000 {
		return nil, fmt.Errorf("description must be under 5000 characters")
	}

	dateStr := r.FormValue("event_date")
	timeStr := r.FormValue("event_time")
	if timeStr == "" {
		timeStr = "16:00"
	}
	loc, _ := time.LoadLocation("Europe/Athens")
	date, err := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+timeStr, loc)
	if err != nil {
		return nil, fmt.Errorf("invalid date: %w", err)
	}

	return &db.Event{
		Title:       title,
		Format:      format,
		Description: description,
		Date:        date,
		Location:    location,
		LocationURL: locationURL,
		EntryFee:    entryFee,
	}, nil
}
