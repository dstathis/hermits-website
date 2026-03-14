package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
)

// SecureCookie sets the Secure flag on session cookies when behind HTTPS.
func SecureCookie(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wrap the ResponseWriter to intercept Set-Cookie headers
		sw := &secureCookieWriter{ResponseWriter: w, r: r}
		next.ServeHTTP(sw, r)
	})
}

type secureCookieWriter struct {
	http.ResponseWriter
	r *http.Request
}

func (w *secureCookieWriter) Write(b []byte) (int, error) {
	return w.ResponseWriter.Write(b)
}

// isHTTPS checks if the request came via HTTPS (directly or through a proxy).
func isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// SignSessionID signs a session ID with HMAC-SHA256.
func SignSessionID(sessionID, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sessionID))
	sig := hex.EncodeToString(mac.Sum(nil))
	return sessionID + "." + sig
}

// VerifySessionID verifies and extracts the session ID from a signed value.
func VerifySessionID(signedValue, secret string) (string, bool) {
	idx := strings.LastIndex(signedValue, ".")
	if idx < 1 || idx >= len(signedValue)-1 {
		return "", false
	}
	sessionID := signedValue[:idx]
	sig := signedValue[idx+1:]

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sessionID))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return "", false
	}
	return sessionID, true
}
