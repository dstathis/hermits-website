package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/derangedhermits/website/internal/db"
	"github.com/derangedhermits/website/internal/middleware"
)

func TestEventsList(t *testing.T) {
	cleanAll(t)
	tmpl := parsePage("events.html")

	db.CreateEvent(testDB, &db.Event{Title: "Future Event", Format: "Legacy", Date: futureDate()})

	h := &EventsHandler{DB: testDB, Templates: tmpl}
	handler := middleware.CSRF("test-secret")(http.HandlerFunc(h.List))

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Future Event") {
		t.Error("expected body to contain event title")
	}
}

func TestEventsICal(t *testing.T) {
	cleanAll(t)

	e := &db.Event{
		Title:    "iCal Test",
		Format:   "Legacy",
		Date:     futureDate(),
		Location: "Test Venue",
	}
	db.CreateEvent(testDB, e)

	h := &EventsHandler{DB: testDB}

	r := chi.NewRouter()
	r.Get("/events/{id}/ical", h.ICal)

	req := httptest.NewRequest(http.MethodGet, "/events/"+e.ID+"/ical", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/calendar") {
		t.Errorf("expected text/calendar content type, got %q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "BEGIN:VCALENDAR") {
		t.Error("expected iCal content")
	}
	if !strings.Contains(body, "iCal Test") {
		t.Error("expected event title in iCal")
	}
}

func TestEventsICal_NotFound(t *testing.T) {
	h := &EventsHandler{DB: testDB}

	r := chi.NewRouter()
	r.Get("/events/{id}/ical", h.ICal)

	req := httptest.NewRequest(http.MethodGet, "/events/00000000-0000-0000-0000-000000000000/ical", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}
