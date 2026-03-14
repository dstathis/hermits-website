package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/derangedhermits/website/internal/db"
	"github.com/derangedhermits/website/internal/mail"
)

type APIHandler struct {
	DB      *sql.DB
	Mailer  *mail.Mailer
	BaseURL string
}

type apiEvent struct {
	ID          string `json:"id,omitempty"`
	Title       string `json:"title"`
	Format      string `json:"format"`
	Description string `json:"description"`
	Date        string `json:"date"`
	Location    string `json:"location"`
	LocationURL string `json:"location_url"`
	EntryFee    string `json:"entry_fee"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

type apiSubscriber struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Confirmed bool   `json:"confirmed"`
	CreatedAt string `json:"created_at"`
}

func (h *APIHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	var events []db.Event
	var err error

	if r.URL.Query().Get("upcoming") == "true" {
		events, err = db.GetUpcomingEvents(h.DB)
	} else {
		events, err = db.GetAllEvents(h.DB)
	}
	if err != nil {
		jsonError(w, "Failed to fetch events", http.StatusInternalServerError)
		return
	}

	result := make([]apiEvent, len(events))
	for i, e := range events {
		result[i] = toAPIEvent(&e)
	}
	jsonResponse(w, result, http.StatusOK)
}

func (h *APIHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	event, err := db.GetEventByID(h.DB, id)
	if err != nil {
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if event == nil {
		jsonError(w, "Event not found", http.StatusNotFound)
		return
	}
	jsonResponse(w, toAPIEvent(event), http.StatusOK)
}

func (h *APIHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var input apiEvent
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB max
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	event, err := fromAPIEvent(&input)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := db.CreateEvent(h.DB, event); err != nil {
		jsonError(w, "Failed to create event", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, toAPIEvent(event), http.StatusCreated)
}

func (h *APIHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	existing, err := db.GetEventByID(h.DB, id)
	if err != nil || existing == nil {
		jsonError(w, "Event not found", http.StatusNotFound)
		return
	}

	var input apiEvent
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB max
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	event, err := fromAPIEvent(&input)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	event.ID = id

	if err := db.UpdateEvent(h.DB, event); err != nil {
		jsonError(w, "Failed to update event", http.StatusInternalServerError)
		return
	}

	updated, _ := db.GetEventByID(h.DB, id)
	jsonResponse(w, toAPIEvent(updated), http.StatusOK)
}

func (h *APIHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := db.DeleteEvent(h.DB, id); err != nil {
		jsonError(w, "Failed to delete event", http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"status": "deleted"}, http.StatusOK)
}

func (h *APIHandler) NotifySubscribers(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	event, err := db.GetEventByID(h.DB, id)
	if err != nil || event == nil {
		jsonError(w, "Event not found", http.StatusNotFound)
		return
	}

	subs, err := db.GetConfirmedSubscriberEmails(h.DB)
	if err != nil {
		jsonError(w, "Failed to get subscribers", http.StatusInternalServerError)
		return
	}

	if len(subs) == 0 {
		jsonResponse(w, map[string]interface{}{"notified": 0}, http.StatusOK)
		return
	}

	subject := "New Event: " + event.Title
	adminH := &AdminHandler{BaseURL: h.BaseURL}

	var errCount int
	for _, sub := range subs {
		body := adminH.renderNotificationEmail(event, sub.Token)
		if err := h.Mailer.Send(sub.Email, subject, body); err != nil {
			errCount++
		}
	}

	jsonResponse(w, map[string]interface{}{
		"notified": len(subs) - errCount,
		"errors":   errCount,
	}, http.StatusOK)
}

func (h *APIHandler) ListSubscribers(w http.ResponseWriter, r *http.Request) {
	subs, err := db.GetAllSubscribers(h.DB)
	if err != nil {
		jsonError(w, "Failed to fetch subscribers", http.StatusInternalServerError)
		return
	}

	result := make([]apiSubscriber, len(subs))
	for i, s := range subs {
		result[i] = apiSubscriber{
			ID:        s.ID,
			Email:     s.Email,
			Name:      s.Name,
			Confirmed: s.Confirmed,
			CreatedAt: s.CreatedAt.Format(time.RFC3339),
		}
	}
	jsonResponse(w, result, http.StatusOK)
}

func toAPIEvent(e *db.Event) apiEvent {
	return apiEvent{
		ID:          e.ID,
		Title:       e.Title,
		Format:      e.Format,
		Description: e.Description,
		Date:        e.Date.Format(time.RFC3339),
		Location:    e.Location,
		LocationURL: e.LocationURL,
		EntryFee:    e.EntryFee,
		CreatedAt:   e.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   e.UpdatedAt.Format(time.RFC3339),
	}
}

func fromAPIEvent(a *apiEvent) (*db.Event, error) {
	title := strings.TrimSpace(a.Title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	date, err := time.Parse(time.RFC3339, a.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format, expected RFC3339")
	}
	return &db.Event{
		Title:       title,
		Format:      strings.TrimSpace(a.Format),
		Description: strings.TrimSpace(a.Description),
		Date:        date,
		Location:    strings.TrimSpace(a.Location),
		LocationURL: strings.TrimSpace(a.LocationURL),
		EntryFee:    strings.TrimSpace(a.EntryFee),
	}, nil
}

func jsonResponse(w http.ResponseWriter, data interface{}, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
