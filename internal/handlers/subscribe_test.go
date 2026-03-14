package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/derangedhermits/website/internal/config"
	"github.com/derangedhermits/website/internal/db"
	"github.com/derangedhermits/website/internal/mail"
	"github.com/derangedhermits/website/internal/middleware"
)

func TestSubscribe_Success(t *testing.T) {
	cleanAll(t)
	tmpl := parsePage("home.html", "subscribe_result.html")
	cfg := &config.Config{}
	mailer := mail.New(cfg)

	h := &SubscribeHandler{DB: testDB, Templates: tmpl, BaseURL: "http://localhost", Mailer: mailer}

	// Get a CSRF token first
	token := getCSRFToken(t)

	form := url.Values{}
	form.Set("email", "sub@example.com")
	form.Set("name", "Tester")
	form.Set("csrf_token", token)

	req := httptest.NewRequest(http.MethodPost, "/subscribe", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})
	rec := httptest.NewRecorder()

	handler := middleware.CSRF("test-secret")(http.HandlerFunc(h.Subscribe))
	handler.ServeHTTP(rec, req)

	// Should redirect
	if rec.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSubscribe_MissingEmail(t *testing.T) {
	cleanAll(t)
	tmpl := parsePage("home.html", "subscribe_result.html")
	cfg := &config.Config{}
	mailer := mail.New(cfg)

	h := &SubscribeHandler{DB: testDB, Templates: tmpl, BaseURL: "http://localhost", Mailer: mailer}

	token := getCSRFToken(t)

	form := url.Values{}
	form.Set("name", "Tester")
	form.Set("csrf_token", token)

	req := httptest.NewRequest(http.MethodPost, "/subscribe", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})
	rec := httptest.NewRecorder()

	handler := middleware.CSRF("test-secret")(http.HandlerFunc(h.Subscribe))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestSubscribe_InvalidEmail(t *testing.T) {
	cleanAll(t)
	tmpl := parsePage("home.html", "subscribe_result.html")
	cfg := &config.Config{}
	mailer := mail.New(cfg)

	h := &SubscribeHandler{DB: testDB, Templates: tmpl, BaseURL: "http://localhost", Mailer: mailer}

	token := getCSRFToken(t)

	form := url.Values{}
	form.Set("email", "not-an-email")
	form.Set("csrf_token", token)

	req := httptest.NewRequest(http.MethodPost, "/subscribe", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})
	rec := httptest.NewRecorder()

	handler := middleware.CSRF("test-secret")(http.HandlerFunc(h.Subscribe))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestConfirm(t *testing.T) {
	cleanAll(t)
	tmpl := parsePage("home.html", "subscribe_result.html")
	cfg := &config.Config{}
	mailer := mail.New(cfg)

	h := &SubscribeHandler{DB: testDB, Templates: tmpl, BaseURL: "http://localhost", Mailer: mailer}

	sub, _ := db.CreateSubscriber(testDB, "confirm-test@example.com", "")

	handler := middleware.CSRF("test-secret")(http.HandlerFunc(h.Confirm))
	req := httptest.NewRequest(http.MethodGet, "/confirm?token="+sub.Token, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestConfirm_BadToken(t *testing.T) {
	tmpl := parsePage("home.html", "subscribe_result.html")
	cfg := &config.Config{}
	mailer := mail.New(cfg)

	h := &SubscribeHandler{DB: testDB, Templates: tmpl, BaseURL: "http://localhost", Mailer: mailer}

	handler := middleware.CSRF("test-secret")(http.HandlerFunc(h.Confirm))
	req := httptest.NewRequest(http.MethodGet, "/confirm?token=bad-token", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 (with error flash), got %d", rec.Code)
	}
}

func TestConfirm_MissingToken(t *testing.T) {
	tmpl := parsePage("home.html", "subscribe_result.html")
	cfg := &config.Config{}
	mailer := mail.New(cfg)

	h := &SubscribeHandler{DB: testDB, Templates: tmpl, BaseURL: "http://localhost", Mailer: mailer}

	req := httptest.NewRequest(http.MethodGet, "/confirm", nil)
	rec := httptest.NewRecorder()
	h.Confirm(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestUnsubscribe(t *testing.T) {
	cleanAll(t)
	tmpl := parsePage("home.html", "subscribe_result.html")
	cfg := &config.Config{}
	mailer := mail.New(cfg)

	h := &SubscribeHandler{DB: testDB, Templates: tmpl, BaseURL: "http://localhost", Mailer: mailer}

	sub, _ := db.CreateSubscriber(testDB, "unsub-test@example.com", "")
	db.ConfirmSubscriber(testDB, sub.Token)

	handler := middleware.CSRF("test-secret")(http.HandlerFunc(h.Unsubscribe))
	req := httptest.NewRequest(http.MethodGet, "/unsubscribe?token="+sub.Token, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestUnsubscribe_BadToken(t *testing.T) {
	tmpl := parsePage("home.html", "subscribe_result.html")
	cfg := &config.Config{}
	mailer := mail.New(cfg)

	h := &SubscribeHandler{DB: testDB, Templates: tmpl, BaseURL: "http://localhost", Mailer: mailer}

	req := httptest.NewRequest(http.MethodGet, "/unsubscribe?token=invalid", nil)
	rec := httptest.NewRecorder()
	h.Unsubscribe(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

// --- Helpers ---

func futureDate() time.Time {
	return time.Now().Add(48 * time.Hour)
}

func containsString(body, substr string) bool {
	return strings.Contains(body, substr)
}

func getCSRFToken(t *testing.T) string {
	t.Helper()
	// Use CSRF middleware on a dummy GET to get a token
	var token string
	handler := middleware.CSRF("test-secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token = middleware.CSRFToken(r)
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if token == "" {
		t.Fatal("failed to get CSRF token")
	}
	return token
}
