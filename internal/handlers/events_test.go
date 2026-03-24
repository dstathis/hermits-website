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

func TestEventDetail(t *testing.T) {
	cleanAll(t)
	tmpl := parsePage("event_detail.html")

	e := &db.Event{
		Title:       "Detail Test Event",
		Format:      "Premodern",
		Date:        futureDate(),
		Location:    "Test Venue",
		LocationURL: "https://example.com/map",
		EntryFee:    "5€",
		Description: "A test event for detail page",
	}
	db.CreateEvent(testDB, e)

	h := &EventsHandler{DB: testDB, DetailTemplate: tmpl}

	r := chi.NewRouter()
	r.Get("/events/{id}", h.Detail)

	req := httptest.NewRequest(http.MethodGet, "/events/"+e.ID, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Detail Test Event") {
		t.Error("expected body to contain event title")
	}
	if !strings.Contains(body, "Premodern") {
		t.Error("expected body to contain event format")
	}
	if !strings.Contains(body, "A test event for detail page") {
		t.Error("expected body to contain event description")
	}
	if !strings.Contains(body, "https://example.com/map") {
		t.Error("expected body to contain location URL")
	}
	if !strings.Contains(body, "5€") {
		t.Error("expected body to contain entry fee")
	}
}

func TestEventDetail_NotFound(t *testing.T) {
	tmpl := parsePage("event_detail.html")
	h := &EventsHandler{DB: testDB, DetailTemplate: tmpl}

	r := chi.NewRouter()
	r.Get("/events/{id}", h.Detail)

	req := httptest.NewRequest(http.MethodGet, "/events/00000000-0000-0000-0000-000000000000", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestEventDetailSetsPageTitle(t *testing.T) {
	cleanAll(t)
	tmpl := parsePage("event_detail.html")

	e := &db.Event{
		Title:  "Title Test",
		Format: "Legacy",
		Date:   futureDate(),
	}
	db.CreateEvent(testDB, e)

	h := &EventsHandler{DB: testDB, DetailTemplate: tmpl}

	r := chi.NewRouter()
	r.Get("/events/{id}", h.Detail)

	req := httptest.NewRequest(http.MethodGet, "/events/"+e.ID, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	// The layout template uses .Title for the <title> tag
	if !strings.Contains(body, "<title>Title Test") {
		t.Errorf("expected page <title> to contain event title, body:\n%s", body)
	}
}

func TestEventsListLinksToDetail(t *testing.T) {
	cleanAll(t)
	tmpl := parsePage("events.html")

	e := &db.Event{
		Title:  "Linked Event",
		Format: "Legacy",
		Date:   futureDate(),
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
	expectedLink := "/events/" + e.ID
	if !strings.Contains(body, expectedLink) {
		t.Errorf("expected events list to contain link %q", expectedLink)
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
