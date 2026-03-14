package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestEventTimeDisplayedInAthens(t *testing.T) {
	cleanAll(t)
	tmpl := parsePage("events.html")

	// Create an event at 16:00 Athens time
	athens, _ := time.LoadLocation("Europe/Athens")
	eventDate := time.Date(2027, 6, 15, 16, 0, 0, 0, athens)

	e := &db.Event{
		Title:    "Time Check",
		Format:   "Legacy",
		Date:     eventDate,
		Location: "Dragonphoenix Inn",
	}
	db.CreateEvent(testDB, e)

	h := &EventsHandler{DB: testDB, Templates: tmpl}
	handler := middleware.CSRF("test-secret")(http.HandlerFunc(h.List))

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	// The rendered page must show 16:00, not the UTC equivalent (13:00)
	if !strings.Contains(body, "16:00") {
		t.Errorf("expected page to display 16:00 (Athens time), body:\n%s", body)
	}
	if strings.Contains(body, "13:00") {
		t.Error("page is displaying UTC time (13:00) instead of Athens time (16:00)")
	}
}

func TestHomeEventTimeDisplayedInAthens(t *testing.T) {
	cleanAll(t)
	tmpl := parsePage("home.html", "subscribe_result.html")

	athens, _ := time.LoadLocation("Europe/Athens")
	eventDate := time.Date(2027, 6, 15, 16, 0, 0, 0, athens)

	db.CreateEvent(testDB, &db.Event{
		Title:  "Home Time Check",
		Format: "Legacy",
		Date:   eventDate,
	})

	h := &HomeHandler{DB: testDB, Templates: tmpl}
	handler := middleware.CSRF("test-secret")(http.HandlerFunc(h.ServeHTTP))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "16:00") {
		t.Errorf("expected homepage to display 16:00 (Athens time), body:\n%s", body)
	}
	if strings.Contains(body, "13:00") {
		t.Error("homepage is displaying UTC time (13:00) instead of Athens time (16:00)")
	}
}
