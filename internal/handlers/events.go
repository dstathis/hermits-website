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
	"github.com/derangedhermits/website/internal/middleware"
)

type EventsHandler struct {
	DB        *sql.DB
	Templates *template.Template
}

func (h *EventsHandler) List(w http.ResponseWriter, r *http.Request) {
	upcoming, err := db.GetUpcomingEvents(h.DB)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	past, err := db.GetPastEvents(h.DB)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Upcoming":  upcoming,
		"Past":      past,
		"CSRFField": middleware.CSRFTemplateField(r),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.Templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *EventsHandler) ICal(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	event, err := db.GetEventByID(h.DB, id)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if event == nil {
		http.NotFound(w, r)
		return
	}

	end := event.Date.Add(4 * time.Hour)
	uid := event.ID + "@derangedhermits.com"

	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\n")
	b.WriteString("VERSION:2.0\r\n")
	b.WriteString("PRODID:-//Deranged Hermits//EN\r\n")
	b.WriteString("BEGIN:VEVENT\r\n")
	fmt.Fprintf(&b, "UID:%s\r\n", uid)
	fmt.Fprintf(&b, "DTSTART:%s\r\n", event.Date.UTC().Format("20060102T150405Z"))
	fmt.Fprintf(&b, "DTEND:%s\r\n", end.UTC().Format("20060102T150405Z"))
	fmt.Fprintf(&b, "SUMMARY:%s\r\n", escapeIcal(event.Title))
	fmt.Fprintf(&b, "LOCATION:%s\r\n", escapeIcal(event.Location))
	fmt.Fprintf(&b, "DESCRIPTION:%s\r\n", escapeIcal(event.Description))
	if event.LocationURL != "" {
		fmt.Fprintf(&b, "URL:%s\r\n", event.LocationURL)
	}
	b.WriteString("END:VEVENT\r\n")
	b.WriteString("END:VCALENDAR\r\n")

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.ics"`, event.ID))
	w.Write([]byte(b.String()))
}

func escapeIcal(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, ";", "\\;")
	return s
}
