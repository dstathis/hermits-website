package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

const (
	csrfCookieName = "csrf_token"
	csrfFormField  = "csrf_token"
	csrfHeaderName = "X-CSRF-Token"
	csrfMaxAge     = 12 * time.Hour
)

type csrfContextKey struct{}

// CSRF middleware protects against cross-site request forgery.
// It sets a CSRF cookie on GET requests and validates the token on POST/PUT/DELETE.
func CSRF(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				// Re-use existing valid CSRF cookie; only issue a new one if missing/invalid
				var token string
				if c, err := r.Cookie(csrfCookieName); err == nil && validSignature(c.Value, secret) {
					token = c.Value
				} else {
					token = generateCSRFToken(secret)
					http.SetCookie(w, &http.Cookie{
						Name:     csrfCookieName,
						Value:    token,
						Path:     "/",
						HttpOnly: false, // JS needs to read it for AJAX
						SameSite: http.SameSiteLaxMode,
						MaxAge:   int(csrfMaxAge.Seconds()),
					})
				}
				// Store token in context so CSRFTemplateField always sees the current token
				ctx := context.WithValue(r.Context(), csrfContextKey{}, token)
				next.ServeHTTP(w, r.WithContext(ctx))

			case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
				cookie, err := r.Cookie(csrfCookieName)
				if err != nil {
					http.Error(w, "CSRF token missing", http.StatusForbidden)
					return
				}

				// Accept token from form field or header
				formToken := r.FormValue(csrfFormField)
				headerToken := r.Header.Get(csrfHeaderName)
				submittedToken := formToken
				if submittedToken == "" {
					submittedToken = headerToken
				}

				if !validateCSRFToken(cookie.Value, submittedToken, secret) {
					http.Error(w, "CSRF token invalid", http.StatusForbidden)
					return
				}

				next.ServeHTTP(w, r)

			default:
				next.ServeHTTP(w, r)
			}
		})
	}
}

func generateCSRFToken(secret string) string {
	b := make([]byte, 32)
	rand.Read(b)
	raw := hex.EncodeToString(b)
	sig := signCSRF(raw, secret)
	return raw + "." + sig
}

func validateCSRFToken(cookieToken, submittedToken, secret string) bool {
	if cookieToken == "" || submittedToken == "" {
		return false
	}
	// The submitted token must match the cookie token
	if cookieToken != submittedToken {
		return false
	}
	// Verify the signature
	parts := splitToken(cookieToken)
	if parts == nil {
		return false
	}
	expectedSig := signCSRF(parts[0], secret)
	return hmac.Equal([]byte(parts[1]), []byte(expectedSig))
}

func signCSRF(data, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

// validSignature checks if a token has a correct HMAC signature.
func validSignature(token, secret string) bool {
	parts := splitToken(token)
	if parts == nil {
		return false
	}
	expected := signCSRF(parts[0], secret)
	return hmac.Equal([]byte(parts[1]), []byte(expected))
}

func splitToken(token string) []string {
	for i := len(token) - 1; i >= 0; i-- {
		if token[i] == '.' {
			if i == 0 || i == len(token)-1 {
				return nil
			}
			return []string{token[:i], token[i+1:]}
		}
	}
	return nil
}

// CSRFToken extracts the current CSRF token from request context (preferred)
// or falls back to the request cookie.
func CSRFToken(r *http.Request) string {
	if token, ok := r.Context().Value(csrfContextKey{}).(string); ok {
		return token
	}
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// CSRFTemplateField returns an HTML hidden input for forms.
func CSRFTemplateField(r *http.Request) template.HTML {
	token := CSRFToken(r)
	return template.HTML(fmt.Sprintf(`<input type="hidden" name="%s" value="%s">`, csrfFormField, token))
}
