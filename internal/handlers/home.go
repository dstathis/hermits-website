package handlers

import (
	"database/sql"
	"html/template"
	"net/http"

	"github.com/derangedhermits/website/internal/db"
	"github.com/derangedhermits/website/internal/middleware"
)

type HomeHandler struct {
	DB        *sql.DB
	Templates *template.Template
}

func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	next, err := db.GetNextEvent(h.DB)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"NextEvent": next,
		"CSRFField": middleware.CSRFTemplateField(r),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.Templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}
