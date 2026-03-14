package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const testSecret = "test-secret-key-for-csrf"

func TestGenerateCSRFToken(t *testing.T) {
	token := generateCSRFToken(testSecret)
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	parts := splitToken(token)
	if parts == nil || len(parts) != 2 {
		t.Fatalf("expected token with two parts, got %q", token)
	}
	if len(parts[0]) != 64 { // 32 bytes hex
		t.Errorf("expected 64 char raw part, got %d", len(parts[0]))
	}
}

func TestValidateCSRFToken(t *testing.T) {
	token := generateCSRFToken(testSecret)

	if !validateCSRFToken(token, token, testSecret) {
		t.Error("expected matching tokens to validate")
	}
	if validateCSRFToken(token, token, "wrong-secret") {
		t.Error("expected different secret to fail")
	}
	if validateCSRFToken(token, "completely-wrong", testSecret) {
		t.Error("expected mismatched tokens to fail")
	}
	if validateCSRFToken("", token, testSecret) {
		t.Error("expected empty cookie token to fail")
	}
	if validateCSRFToken(token, "", testSecret) {
		t.Error("expected empty submitted token to fail")
	}
}

func TestValidSignature(t *testing.T) {
	token := generateCSRFToken(testSecret)
	if !validSignature(token, testSecret) {
		t.Error("expected valid signature")
	}
	if validSignature("tampered.sig", testSecret) {
		t.Error("expected invalid signature for tampered token")
	}
	if validSignature("no-dot-here", testSecret) {
		t.Error("expected invalid for token without separator")
	}
	if validSignature("", testSecret) {
		t.Error("expected invalid for empty token")
	}
}

func TestSplitToken(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"abc.def", []string{"abc", "def"}},
		{"abc.def.ghi", []string{"abc.def", "ghi"}},
		{".abc", nil},
		{"abc.", nil},
		{"nodot", nil},
		{"", nil},
	}
	for _, tt := range tests {
		got := splitToken(tt.input)
		if tt.want == nil {
			if got != nil {
				t.Errorf("splitToken(%q) = %v, want nil", tt.input, got)
			}
			continue
		}
		if got == nil || got[0] != tt.want[0] || got[1] != tt.want[1] {
			t.Errorf("splitToken(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestCSRFMiddleware_GETSetsTokenCookie(t *testing.T) {
	handler := CSRF(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	cookies := resp.Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == csrfCookieName {
			csrfCookie = c
		}
	}
	if csrfCookie == nil {
		t.Fatal("expected CSRF cookie to be set on GET")
	}
	if !validSignature(csrfCookie.Value, testSecret) {
		t.Error("CSRF cookie has invalid signature")
	}
}

func TestCSRFMiddleware_GETReusesValidCookie(t *testing.T) {
	existingToken := generateCSRFToken(testSecret)

	handler := CSRF(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that the context token matches the existing cookie
		token := CSRFToken(r)
		if token != existingToken {
			t.Errorf("expected context token %q, got %q", existingToken, token)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: existingToken})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should NOT set a new cookie since the existing one is valid
	resp := rec.Result()
	for _, c := range resp.Cookies() {
		if c.Name == csrfCookieName {
			t.Error("should not set new CSRF cookie when existing one is valid")
		}
	}
}

func TestCSRFMiddleware_POSTValid(t *testing.T) {
	token := generateCSRFToken(testSecret)
	called := false

	handler := CSRF(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	form := url.Values{}
	form.Set(csrfFormField, token)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: token})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("expected handler to be called for valid CSRF")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestCSRFMiddleware_POSTMissingCookie(t *testing.T) {
	handler := CSRF(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	form := url.Values{}
	form.Set(csrfFormField, "some-token")

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestCSRFMiddleware_POSTMismatchedToken(t *testing.T) {
	cookieToken := generateCSRFToken(testSecret)
	formToken := generateCSRFToken(testSecret)

	handler := CSRF(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	form := url.Values{}
	form.Set(csrfFormField, formToken)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: cookieToken})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestCSRFMiddleware_POSTViaHeader(t *testing.T) {
	token := generateCSRFToken(testSecret)
	called := false

	handler := CSRF(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(csrfHeaderName, token)
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: token})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("expected handler to be called for valid header CSRF")
	}
}

func TestCSRFTemplateField(t *testing.T) {
	handler := CSRF(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		field := CSRFTemplateField(r)
		s := string(field)
		if !strings.Contains(s, `<input type="hidden"`) {
			t.Error("expected hidden input in template field")
		}
		if !strings.Contains(s, `name="csrf_token"`) {
			t.Error("expected csrf_token name in template field")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
}
