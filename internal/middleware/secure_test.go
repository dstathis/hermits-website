package middleware

import (
	"testing"
)

func TestSignAndVerifySessionID(t *testing.T) {
	secret := "my-session-secret"
	sessionID := "abc-123-def-456"

	signed := SignSessionID(sessionID, secret)
	if signed == sessionID {
		t.Fatal("signed value should differ from raw session ID")
	}

	// Verify with correct secret
	got, ok := VerifySessionID(signed, secret)
	if !ok {
		t.Fatal("expected verification to succeed")
	}
	if got != sessionID {
		t.Errorf("expected session ID %q, got %q", sessionID, got)
	}

	// Verify with wrong secret
	_, ok = VerifySessionID(signed, "wrong-secret")
	if ok {
		t.Error("expected verification to fail with wrong secret")
	}

	// Verify tampered value
	_, ok = VerifySessionID("tampered.value", secret)
	if ok {
		t.Error("expected verification to fail for tampered value")
	}

	// Verify empty
	_, ok = VerifySessionID("", secret)
	if ok {
		t.Error("expected verification to fail for empty value")
	}

	// Verify no separator
	_, ok = VerifySessionID("noseparator", secret)
	if ok {
		t.Error("expected verification to fail when no separator")
	}
}

func TestSignSessionID_DifferentSecrets(t *testing.T) {
	id := "session-id"
	sig1 := SignSessionID(id, "secret-a")
	sig2 := SignSessionID(id, "secret-b")
	if sig1 == sig2 {
		t.Error("different secrets should produce different signatures")
	}
}
