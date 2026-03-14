package handlers

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/derangedhermits/website/internal/config"
	"github.com/derangedhermits/website/internal/db"
	"github.com/derangedhermits/website/internal/mail"
	"github.com/derangedhermits/website/internal/middleware"
)

func newAdminHandler() *AdminHandler {
	dir := templateDir()
	parseAdmin := func(file string) *template.Template {
		return template.Must(template.New(file).Funcs(testFuncMap).ParseFiles(dir + "/" + file))
	}
	loginTmpl := parseAdmin("admin_login.html")
	dashTmpl := parseAdmin("admin.html")
	formTmpl := parseAdmin("admin_event_form.html")
	inviteTmpl := parseAdmin("admin_invite.html")
	cfg := &config.Config{}
	mailer := mail.New(cfg)

	return &AdminHandler{
		DB:                testDB,
		LoginTemplate:     loginTmpl,
		DashTemplate:      dashTmpl,
		EventFormTemplate: formTmpl,
		InviteTemplate:    inviteTmpl,
		Mailer:            mailer,
		BaseURL:           "http://localhost",
		SessionSecret:     "test-secret",
	}
}

func TestAdminLoginPage(t *testing.T) {
	h := newAdminHandler()
	handler := middleware.CSRF("test-secret")(http.HandlerFunc(h.LoginPage))

	req := httptest.NewRequest(http.MethodGet, "/admin/login", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestAdminLogin_Success(t *testing.T) {
	cleanAll(t)
	h := newAdminHandler()

	db.CreateAdminUser(testDB, "logintest", "password123")
	token := getCSRFToken(t)

	form := url.Values{}
	form.Set("username", "logintest")
	form.Set("password", "password123")
	form.Set("csrf_token", token)

	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})
	rec := httptest.NewRecorder()

	handler := middleware.CSRF("test-secret")(http.HandlerFunc(h.Login))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d: %s", rec.Code, rec.Body.String())
	}

	// Should have session cookie
	var hasSession bool
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session" {
			hasSession = true
		}
	}
	if !hasSession {
		t.Error("expected session cookie to be set on login")
	}
}

func TestAdminLogin_WrongPassword(t *testing.T) {
	cleanAll(t)
	h := newAdminHandler()

	db.CreateAdminUser(testDB, "logintest2", "correct")
	token := getCSRFToken(t)

	form := url.Values{}
	form.Set("username", "logintest2")
	form.Set("password", "wrong")
	form.Set("csrf_token", token)

	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})
	rec := httptest.NewRecorder()

	handler := middleware.CSRF("test-secret")(http.HandlerFunc(h.Login))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAdminDashboard(t *testing.T) {
	cleanAll(t)
	h := newAdminHandler()

	db.CreateAdminUser(testDB, "dashuser", "pass1234")
	user, _ := db.Authenticate(testDB, "dashuser", "pass1234")

	// Create context with user ID
	handler := middleware.CSRF("test-secret")(http.HandlerFunc(h.Dashboard))
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, user.ID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminCreateEvent(t *testing.T) {
	cleanAll(t)
	h := newAdminHandler()
	token := getCSRFToken(t)

	form := url.Values{}
	form.Set("title", "Admin Created Event")
	form.Set("format", "Legacy")
	form.Set("event_date", "2027-06-01")
	form.Set("event_time", "16:00")
	form.Set("location", "Test Venue")
	form.Set("entry_fee", "5€")
	form.Set("csrf_token", token)

	req := httptest.NewRequest(http.MethodPost, "/admin/events", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})
	rec := httptest.NewRecorder()

	handler := middleware.CSRF("test-secret")(http.HandlerFunc(h.CreateEvent))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d: %s", rec.Code, rec.Body.String())
	}

	events, _ := db.GetAllEvents(testDB)
	found := false
	for _, e := range events {
		if e.Title == "Admin Created Event" {
			found = true
		}
	}
	if !found {
		t.Error("expected event to be created")
	}
}

func TestAdminCreateEvent_MissingTitle(t *testing.T) {
	h := newAdminHandler()
	token := getCSRFToken(t)

	form := url.Values{}
	form.Set("format", "Legacy")
	form.Set("event_date", "2027-06-01")
	form.Set("csrf_token", token)

	req := httptest.NewRequest(http.MethodPost, "/admin/events", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})
	rec := httptest.NewRecorder()

	handler := middleware.CSRF("test-secret")(http.HandlerFunc(h.CreateEvent))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAdminDeleteEvent(t *testing.T) {
	cleanAll(t)
	h := newAdminHandler()
	token := getCSRFToken(t)

	e := &db.Event{Title: "To Delete", Format: "Legacy", Date: futureDate()}
	db.CreateEvent(testDB, e)

	r := chi.NewRouter()
	r.Use(middleware.CSRF("test-secret"))
	r.Post("/admin/events/{id}/delete", h.DeleteEvent)

	form := url.Values{}
	form.Set("csrf_token", token)

	req := httptest.NewRequest(http.MethodPost, "/admin/events/"+e.ID+"/delete", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", rec.Code)
	}

	got, _ := db.GetEventByID(testDB, e.ID)
	if got != nil {
		t.Error("expected event to be deleted")
	}
}

func TestAdminDeleteUser_SelfOnly(t *testing.T) {
	cleanAll(t)
	h := newAdminHandler()
	token := getCSRFToken(t)

	db.CreateAdminUser(testDB, "selfuser", "pass1234")
	db.CreateAdminUser(testDB, "otheruser", "pass1234")
	self, _ := db.Authenticate(testDB, "selfuser", "pass1234")
	other, _ := db.Authenticate(testDB, "otheruser", "pass1234")

	r := chi.NewRouter()
	r.Use(middleware.CSRF("test-secret"))
	r.Post("/admin/users/{id}/delete", h.DeleteUser)

	// Try to delete another user — should fail
	form := url.Values{}
	form.Set("csrf_token", token)

	req := httptest.NewRequest(http.MethodPost, "/admin/users/"+other.ID+"/delete", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, self.ID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("expected 303 redirect, got %d", rec.Code)
	}

	// Other user should still exist
	users, _ := db.GetAllAdminUsers(testDB)
	if len(users) != 2 {
		t.Error("other user should not have been deleted")
	}
}

func TestAdminAcceptInvite(t *testing.T) {
	cleanAll(t)
	h := newAdminHandler()

	inviteToken, _ := db.InviteAdminUser(testDB, "newinvitee", "invite@test.com")

	// GET the form
	handler := middleware.CSRF("test-secret")(http.HandlerFunc(h.AcceptInviteForm))
	req := httptest.NewRequest(http.MethodGet, "/admin/invite/accept?token="+inviteToken, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// POST to accept
	csrfToken := getCSRFToken(t)
	form := url.Values{}
	form.Set("token", inviteToken)
	form.Set("password", "newpassword123")
	form.Set("password_confirm", "newpassword123")
	form.Set("csrf_token", csrfToken)

	req = httptest.NewRequest(http.MethodPost, "/admin/invite/accept", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfToken})
	rec = httptest.NewRecorder()
	handler = middleware.CSRF("test-secret")(http.HandlerFunc(h.AcceptInvite))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d: %s", rec.Code, rec.Body.String())
	}

	// Should now be able to log in
	user, _ := db.Authenticate(testDB, "newinvitee", "newpassword123")
	if user == nil {
		t.Error("expected to authenticate after accepting invite")
	}
}

func TestAdminAcceptInvite_PasswordTooShort(t *testing.T) {
	cleanAll(t)
	h := newAdminHandler()

	inviteToken, _ := db.InviteAdminUser(testDB, "shortpw", "short@test.com")

	csrfToken := getCSRFToken(t)
	form := url.Values{}
	form.Set("token", inviteToken)
	form.Set("password", "short")
	form.Set("password_confirm", "short")
	form.Set("csrf_token", csrfToken)

	req := httptest.NewRequest(http.MethodPost, "/admin/invite/accept", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfToken})
	rec := httptest.NewRecorder()
	handler := middleware.CSRF("test-secret")(http.HandlerFunc(h.AcceptInvite))
	handler.ServeHTTP(rec, req)

	// Should re-render form (200) with error, not redirect
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for short password, got %d", rec.Code)
	}
}

func TestAdminAcceptInvite_PasswordMismatch(t *testing.T) {
	cleanAll(t)
	h := newAdminHandler()

	inviteToken, _ := db.InviteAdminUser(testDB, "mismatch", "mismatch@test.com")

	csrfToken := getCSRFToken(t)
	form := url.Values{}
	form.Set("token", inviteToken)
	form.Set("password", "password123")
	form.Set("password_confirm", "different123")
	form.Set("csrf_token", csrfToken)

	req := httptest.NewRequest(http.MethodPost, "/admin/invite/accept", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfToken})
	rec := httptest.NewRecorder()
	handler := middleware.CSRF("test-secret")(http.HandlerFunc(h.AcceptInvite))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for mismatch, got %d", rec.Code)
	}
}

func TestAdminChangePassword(t *testing.T) {
	cleanAll(t)
	h := newAdminHandler()

	db.CreateAdminUser(testDB, "changepwuser", "oldpass12")
	user, _ := db.Authenticate(testDB, "changepwuser", "oldpass12")

	csrfToken := getCSRFToken(t)
	form := url.Values{}
	form.Set("current_password", "oldpass12")
	form.Set("new_password", "newpass12")
	form.Set("new_password_confirm", "newpass12")
	form.Set("csrf_token", csrfToken)

	req := httptest.NewRequest(http.MethodPost, "/admin/password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfToken})
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, user.ID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler := middleware.CSRF("test-secret")(http.HandlerFunc(h.ChangePassword))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify new password works
	u, _ := db.Authenticate(testDB, "changepwuser", "newpass12")
	if u == nil {
		t.Error("expected new password to work")
	}
}
