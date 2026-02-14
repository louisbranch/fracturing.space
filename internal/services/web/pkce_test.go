package web

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestGenerateCodeVerifier(t *testing.T) {
	v1, err := generateCodeVerifier()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 32 random bytes → 64 hex characters.
	if len(v1) != 64 {
		t.Fatalf("verifier length = %d, want 64", len(v1))
	}

	// Should be unique across calls.
	v2, err := generateCodeVerifier()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v1 == v2 {
		t.Fatal("expected unique verifiers")
	}
}

func TestComputeS256Challenge(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	got := computeS256Challenge(verifier)

	// Independently compute the expected value.
	hash := sha256.Sum256([]byte(verifier))
	want := base64.RawURLEncoding.EncodeToString(hash[:])
	if got != want {
		t.Fatalf("computeS256Challenge(%q) = %q, want %q", verifier, got, want)
	}
}
