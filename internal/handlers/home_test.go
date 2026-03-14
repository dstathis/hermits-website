package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/derangedhermits/website/internal/db"
	"github.com/derangedhermits/website/internal/middleware"
)

func TestHomeHandler_OK(t *testing.T) {
	cleanAll(t)
	tmpl := parsePage("home.html", "subscribe_result.html")

	h := &HomeHandler{DB: testDB, Templates: tmpl}

	// Wrap with CSRF so CSRFTemplateField works
	handler := middleware.CSRF("test-secret")(http.HandlerFunc(h.ServeHTTP))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if len(body) == 0 {
		t.Error("expected non-empty body")
	}
}

func TestHomeHandler_WithNextEvent(t *testing.T) {
	cleanAll(t)
	tmpl := parsePage("home.html", "subscribe_result.html")

	e := &db.Event{
		Title:  "Upcoming Test",
		Format: "Legacy",
		Date:   futureDate(),
	}
	db.CreateEvent(testDB, e)

	h := &HomeHandler{DB: testDB, Templates: tmpl}
	handler := middleware.CSRF("test-secret")(http.HandlerFunc(h.ServeHTTP))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	// Should contain the event title
	if body := rec.Body.String(); !containsString(body, "Upcoming Test") {
		t.Error("expected body to contain event title")
	}
}

func TestHomeHandler_404ForOtherPaths(t *testing.T) {
	tmpl := parsePage("home.html", "subscribe_result.html")
	h := &HomeHandler{DB: testDB, Templates: tmpl}

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}
