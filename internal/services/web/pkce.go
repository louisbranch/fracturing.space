package web

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// PKCE helpers bridge browser OAuth login to OAuth providers without storing
// secrets server-side, keeping the OAuth code exchange bound to this login flow.

// generateCodeVerifier returns a random PKCE code verifier (64 hex characters).
func generateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// computeS256Challenge derives the S256 challenge for OAuth authorization requests.
// This forces clients and server to use the same PKCE binding during token exchange.
func computeS256Challenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
