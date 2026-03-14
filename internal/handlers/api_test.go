package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/derangedhermits/website/internal/config"
	"github.com/derangedhermits/website/internal/db"
	"github.com/derangedhermits/website/internal/mail"
	"github.com/derangedhermits/website/internal/middleware"
)

func newAPIRouter() (*chi.Mux, *APIHandler) {
	cfg := &config.Config{}
	mailer := mail.New(cfg)
	apiH := &APIHandler{DB: testDB, Mailer: mailer, BaseURL: "http://localhost"}

	r := chi.NewRouter()
	r.Use(middleware.RequireAPIKey("test-api-key"))
	r.Get("/api/events", apiH.ListEvents)
	r.Get("/api/events/{id}", apiH.GetEvent)
	r.Post("/api/events", apiH.CreateEvent)
	r.Put("/api/events/{id}", apiH.UpdateEvent)
	r.Delete("/api/events/{id}", apiH.DeleteEvent)
	r.Get("/api/subscribers", apiH.ListSubscribers)

	return r, apiH
}

func apiRequest(method, path, body string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	r.Header.Set("Authorization", "Bearer test-api-key")
	return r
}

func TestAPIListEvents(t *testing.T) {
	cleanAll(t)
	router, _ := newAPIRouter()

	db.CreateEvent(testDB, &db.Event{Title: "API Event", Format: "Legacy", Date: futureDate()})

	req := apiRequest(http.MethodGet, "/api/events", "")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var events []apiEvent
	json.NewDecoder(rec.Body).Decode(&events)
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
	if events[0].Title != "API Event" {
		t.Errorf("expected 'API Event', got %q", events[0].Title)
	}
}

func TestAPIListEvents_UpcomingFilter(t *testing.T) {
	cleanAll(t)
	router, _ := newAPIRouter()

	db.CreateEvent(testDB, &db.Event{Title: "Future", Format: "Legacy", Date: futureDate()})

	req := apiRequest(http.MethodGet, "/api/events?upcoming=true", "")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var events []apiEvent
	json.NewDecoder(rec.Body).Decode(&events)
	if len(events) != 1 {
		t.Errorf("expected 1 upcoming event, got %d", len(events))
	}
}

func TestAPICreateEvent(t *testing.T) {
	cleanAll(t)
	router, _ := newAPIRouter()

	body := `{"title":"Created via API","format":"Premodern","date":"2027-06-01T16:00:00Z","location":"Athens","entry_fee":"5€"}`
	req := apiRequest(http.MethodPost, "/api/events", body)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var event apiEvent
	json.NewDecoder(rec.Body).Decode(&event)
	if event.Title != "Created via API" {
		t.Errorf("expected 'Created via API', got %q", event.Title)
	}
	if event.ID == "" {
		t.Error("expected event ID in response")
	}
}

func TestAPICreateEvent_MissingTitle(t *testing.T) {
	cleanAll(t)
	router, _ := newAPIRouter()

	body := `{"format":"Legacy","date":"2027-06-01T16:00:00Z"}`
	req := apiRequest(http.MethodPost, "/api/events", body)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAPICreateEvent_BadJSON(t *testing.T) {
	cleanAll(t)
	router, _ := newAPIRouter()

	req := apiRequest(http.MethodPost, "/api/events", "not json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAPIGetEvent(t *testing.T) {
	cleanAll(t)
	router, _ := newAPIRouter()

	e := &db.Event{Title: "Get Me", Format: "Legacy", Date: futureDate()}
	db.CreateEvent(testDB, e)

	req := apiRequest(http.MethodGet, "/api/events/"+e.ID, "")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var event apiEvent
	json.NewDecoder(rec.Body).Decode(&event)
	if event.Title != "Get Me" {
		t.Errorf("expected 'Get Me', got %q", event.Title)
	}
}

func TestAPIGetEvent_NotFound(t *testing.T) {
	cleanAll(t)
	router, _ := newAPIRouter()

	req := apiRequest(http.MethodGet, "/api/events/00000000-0000-0000-0000-000000000000", "")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestAPIUpdateEvent(t *testing.T) {
	cleanAll(t)
	router, _ := newAPIRouter()

	e := &db.Event{Title: "Original", Format: "Legacy", Date: futureDate()}
	db.CreateEvent(testDB, e)

	body := `{"title":"Updated via API","format":"Premodern","date":"2027-06-01T16:00:00Z"}`
	req := apiRequest(http.MethodPut, "/api/events/"+e.ID, body)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var event apiEvent
	json.NewDecoder(rec.Body).Decode(&event)
	if event.Title != "Updated via API" {
		t.Errorf("expected 'Updated via API', got %q", event.Title)
	}
}

func TestAPIDeleteEvent(t *testing.T) {
	cleanAll(t)
	router, _ := newAPIRouter()

	e := &db.Event{Title: "Delete Me", Format: "Legacy", Date: futureDate()}
	db.CreateEvent(testDB, e)

	req := apiRequest(http.MethodDelete, "/api/events/"+e.ID, "")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// Verify deleted
	got, _ := db.GetEventByID(testDB, e.ID)
	if got != nil {
		t.Error("expected event to be deleted")
	}
}

func TestAPIListSubscribers(t *testing.T) {
	cleanAll(t)
	router, _ := newAPIRouter()

	db.CreateSubscriber(testDB, "api-sub@example.com", "API Sub")

	req := apiRequest(http.MethodGet, "/api/subscribers", "")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var subs []apiSubscriber
	json.NewDecoder(rec.Body).Decode(&subs)
	if len(subs) != 1 {
		t.Errorf("expected 1 subscriber, got %d", len(subs))
	}
}

func TestAPIUnauthorized(t *testing.T) {
	router, _ := newAPIRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	// No auth header
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
