package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any environment variables that could interfere
	envVars := []string{"PORT", "DATABASE_URL", "SESSION_SECRET", "API_KEY", "BASE_URL", "SMTP_HOST", "SMTP_PORT", "SMTP_USER", "SMTP_PASS", "SMTP_FROM"}
	saved := make(map[string]string)
	for _, k := range envVars {
		saved[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	defer func() {
		for k, v := range saved {
			if v != "" {
				os.Setenv(k, v)
			}
		}
	}()

	cfg := Load()

	if cfg.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Port)
	}
	if cfg.SMTPPort != 587 {
		t.Errorf("expected default SMTP port 587, got %d", cfg.SMTPPort)
	}
	if cfg.SMTPFrom != "noreply@derangedhermits.com" {
		t.Errorf("expected default SMTP from, got %q", cfg.SMTPFrom)
	}
	if cfg.SMTPHost != "" {
		t.Errorf("expected empty SMTP host by default, got %q", cfg.SMTPHost)
	}
	if cfg.BaseURL != "http://localhost:8080" {
		t.Errorf("expected default base URL, got %q", cfg.BaseURL)
	}
}

func TestLoad_FromEnv(t *testing.T) {
	os.Setenv("PORT", "3000")
	os.Setenv("DATABASE_URL", "postgres://test:test@localhost/test")
	os.Setenv("SESSION_SECRET", "supersecret")
	os.Setenv("API_KEY", "my-key")
	os.Setenv("BASE_URL", "https://example.com")
	os.Setenv("SMTP_HOST", "smtp.example.com")
	os.Setenv("SMTP_PORT", "465")
	os.Setenv("SMTP_USER", "user@example.com")
	os.Setenv("SMTP_PASS", "smtppass")
	os.Setenv("SMTP_FROM", "events@example.com")
	defer func() {
		for _, k := range []string{"PORT", "DATABASE_URL", "SESSION_SECRET", "API_KEY", "BASE_URL", "SMTP_HOST", "SMTP_PORT", "SMTP_USER", "SMTP_PASS", "SMTP_FROM"} {
			os.Unsetenv(k)
		}
	}()

	cfg := Load()

	if cfg.Port != 3000 {
		t.Errorf("expected port 3000, got %d", cfg.Port)
	}
	if cfg.DatabaseURL != "postgres://test:test@localhost/test" {
		t.Errorf("expected custom DB URL, got %q", cfg.DatabaseURL)
	}
	if cfg.SessionSecret != "supersecret" {
		t.Errorf("expected custom session secret, got %q", cfg.SessionSecret)
	}
	if cfg.APIKey != "my-key" {
		t.Errorf("expected custom API key, got %q", cfg.APIKey)
	}
	if cfg.BaseURL != "https://example.com" {
		t.Errorf("expected custom base URL, got %q", cfg.BaseURL)
	}
	if cfg.SMTPHost != "smtp.example.com" {
		t.Errorf("expected custom SMTP host, got %q", cfg.SMTPHost)
	}
	if cfg.SMTPPort != 465 {
		t.Errorf("expected SMTP port 465, got %d", cfg.SMTPPort)
	}
	if cfg.SMTPUser != "user@example.com" {
		t.Errorf("expected custom SMTP user, got %q", cfg.SMTPUser)
	}
	if cfg.SMTPPass != "smtppass" {
		t.Errorf("expected custom SMTP pass, got %q", cfg.SMTPPass)
	}
	if cfg.SMTPFrom != "events@example.com" {
		t.Errorf("expected custom SMTP from, got %q", cfg.SMTPFrom)
	}
}

func TestGetEnv(t *testing.T) {
	os.Setenv("TEST_HERMITS_VAR", "hello")
	defer os.Unsetenv("TEST_HERMITS_VAR")

	if v := getEnv("TEST_HERMITS_VAR", "default"); v != "hello" {
		t.Errorf("expected 'hello', got %q", v)
	}
	if v := getEnv("TEST_HERMITS_MISSING", "fallback"); v != "fallback" {
		t.Errorf("expected 'fallback', got %q", v)
	}
}
