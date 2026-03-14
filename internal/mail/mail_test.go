package mail

import (
	"strings"
	"testing"

	"github.com/derangedhermits/website/internal/config"
)

func TestBuildMessage(t *testing.T) {
	msg := buildMessage("from@example.com", "to@example.com", "Test Subject", "<h1>Hello</h1>")
	s := string(msg)

	if !strings.Contains(s, "From: from@example.com\r\n") {
		t.Error("expected From header")
	}
	if !strings.Contains(s, "To: to@example.com\r\n") {
		t.Error("expected To header")
	}
	if !strings.Contains(s, "Subject: Test Subject\r\n") {
		t.Error("expected Subject header")
	}
	if !strings.Contains(s, "MIME-Version: 1.0\r\n") {
		t.Error("expected MIME-Version header")
	}
	if !strings.Contains(s, "Content-Type: text/html") {
		t.Error("expected Content-Type header")
	}
	if !strings.Contains(s, "<h1>Hello</h1>") {
		t.Error("expected HTML body")
	}
	// Header/body separator
	if !strings.Contains(s, "\r\n\r\n") {
		t.Error("expected blank line between headers and body")
	}
}

func TestMailerEnabled(t *testing.T) {
	cfg := &config.Config{SMTPHost: ""}
	m := New(cfg)
	if m.Enabled() {
		t.Error("expected Enabled() to be false when SMTPHost is empty")
	}

	cfg.SMTPHost = "smtp.example.com"
	m = New(cfg)
	if !m.Enabled() {
		t.Error("expected Enabled() to be true when SMTPHost is set")
	}
}

func TestSendDisabled(t *testing.T) {
	cfg := &config.Config{SMTPHost: ""}
	m := New(cfg)
	if err := m.Send("to@example.com", "Subject", "body"); err == nil {
		t.Error("expected error when SMTP not configured")
	}
}

func TestSendToMany_AllFail(t *testing.T) {
	cfg := &config.Config{SMTPHost: ""}
	m := New(cfg)
	errs := m.SendToMany([]string{"a@example.com", "b@example.com"}, "Subject", "body")
	if len(errs) != 2 {
		t.Errorf("expected 2 errors, got %d", len(errs))
	}
}

func TestSendToMany_EmptyList(t *testing.T) {
	cfg := &config.Config{SMTPHost: "smtp.example.com"}
	m := New(cfg)
	errs := m.SendToMany(nil, "Subject", "body")
	if len(errs) != 0 {
		t.Errorf("expected 0 errors for empty list, got %d", len(errs))
	}
}
