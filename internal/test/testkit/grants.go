package testkit

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

const (
	defaultAISessionGrantIssuer   = "fracturing-space-game"
	defaultAISessionGrantAudience = "fracturing-space-ai"
	defaultAISessionGrantTTL      = "10m"
	defaultAISessionGrantHMACKey  = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY"
)

var (
	joinGrantKeyOnce    sync.Once
	joinGrantPrivateKey ed25519.PrivateKey
	joinGrantPublicKey  ed25519.PublicKey
)

func ensureJoinGrantKey(t *testing.T) {
	t.Helper()

	joinGrantKeyOnce.Do(func() {
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate join grant key: %v", err)
		}
		joinGrantPublicKey = publicKey
		joinGrantPrivateKey = privateKey
	})
}

// SetJoinGrantEnv configures the shared join-grant test environment.
func SetJoinGrantEnv(t *testing.T, issuer, audience string) {
	t.Helper()

	ensureJoinGrantKey(t)

	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", issuer)
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", audience)
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY", base64.RawStdEncoding.EncodeToString(joinGrantPublicKey))
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY", base64.RawStdEncoding.EncodeToString(joinGrantPrivateKey))
}

// SignJoinGrantToken returns a signed join-grant JWT using the shared test key.
func SignJoinGrantToken(t *testing.T, issuer, audience, campaignID, inviteID, userID string, now time.Time) string {
	t.Helper()

	ensureJoinGrantKey(t)
	if now.IsZero() {
		now = time.Now().UTC()
	}
	headerJSON, err := json.Marshal(map[string]string{
		"alg": "EdDSA",
		"typ": "JWT",
	})
	if err != nil {
		t.Fatalf("encode join grant header: %v", err)
	}
	payloadJSON, err := json.Marshal(map[string]any{
		"iss":         issuer,
		"aud":         audience,
		"exp":         now.Add(5 * time.Minute).Unix(),
		"iat":         now.Unix(),
		"jti":         fmt.Sprintf("jti-%d", now.UnixNano()),
		"campaign_id": campaignID,
		"invite_id":   inviteID,
		"user_id":     userID,
	})
	if err != nil {
		t.Fatalf("encode join grant payload: %v", err)
	}
	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := encodedHeader + "." + encodedPayload
	signature := ed25519.Sign(joinGrantPrivateKey, []byte(signingInput))
	encodedSig := base64.RawURLEncoding.EncodeToString(signature)
	return signingInput + "." + encodedSig
}

// SetAISessionGrantEnv configures the shared AI session grant test environment.
func SetAISessionGrantEnv(t *testing.T) {
	t.Helper()
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_ISSUER", defaultAISessionGrantIssuer)
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_AUDIENCE", defaultAISessionGrantAudience)
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_HMAC_KEY", defaultAISessionGrantHMACKey)
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_TTL", defaultAISessionGrantTTL)
}
