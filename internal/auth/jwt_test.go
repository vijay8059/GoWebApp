package auth

import (
	"errors"
	"testing"
	"time"
)

func newVerifier(t *testing.T) *Verifier {
	t.Helper()
	v, err := New(Config{
		Secret:   "test-secret-do-not-use-in-prod",
		Issuer:   "gowebapp-test",
		Audience: "gowebapp",
		TTL:      time.Minute,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return v
}

func TestNew_RejectsEmptySecret(t *testing.T) {
	if _, err := New(Config{}); !errors.Is(err, ErrMissingSecret) {
		t.Fatalf("err = %v, want ErrMissingSecret", err)
	}
}

func TestSignAndParse_RoundTrip(t *testing.T) {
	v := newVerifier(t)
	tok, err := v.Sign("alice")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	sub, err := v.Parse(tok)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if sub != "alice" {
		t.Fatalf("subject = %q, want alice", sub)
	}
}

func TestParse_RejectsWrongSecret(t *testing.T) {
	signer := newVerifier(t)
	tok, _ := signer.Sign("alice")

	other, _ := New(Config{Secret: "different-secret", Issuer: "gowebapp-test", Audience: "gowebapp"})
	if _, err := other.Parse(tok); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestParse_RejectsExpiredToken(t *testing.T) {
	v, _ := New(Config{Secret: "s", TTL: -1 * time.Second})
	// TTL <= 0 falls back to 1h; use a custom one-off signer with a past exp.
	short, _ := New(Config{Secret: "s", TTL: time.Millisecond})
	tok, _ := short.Sign("alice")
	time.Sleep(5 * time.Millisecond)
	if _, err := v.Parse(tok); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken (expired)", err)
	}
}

func TestParse_RejectsWrongIssuer(t *testing.T) {
	signer, _ := New(Config{Secret: "s", Issuer: "evil", TTL: time.Minute})
	tok, _ := signer.Sign("alice")

	v, _ := New(Config{Secret: "s", Issuer: "gowebapp", TTL: time.Minute})
	if _, err := v.Parse(tok); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken (issuer)", err)
	}
}
