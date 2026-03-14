package middleware

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/derangedhermits/website/internal/db"
)

type contextKey string

const UserIDKey contextKey = "userID"

func RequireAuth(database *sql.DB, sessionSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session")
			if err != nil {
				http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
				return
			}
			sessionID, ok := VerifySessionID(cookie.Value, sessionSecret)
			if !ok {
				http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
				return
			}
			session, err := db.GetSession(database, sessionID)
			if err != nil || session == nil {
				http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, session.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireAPIKey(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if apiKey == "" {
				http.Error(w, `{"error":"API not configured"}`, http.StatusServiceUnavailable)
				return
			}
			auth := r.Header.Get("Authorization")
			if auth != "Bearer "+apiKey {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
