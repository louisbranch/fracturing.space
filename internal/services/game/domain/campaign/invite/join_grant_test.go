package invite

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestLoadJoinGrantConfigFromEnv(t *testing.T) {
	t.Setenv(EnvJoinGrantIssuer, "")
	t.Setenv(EnvJoinGrantAudience, "")
	t.Setenv(EnvJoinGrantPublicKey, "")

	if _, err := LoadJoinGrantConfigFromEnv(nil); err == nil {
		t.Fatal("expected error when env vars are missing")
	}

	pubKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	t.Setenv(EnvJoinGrantIssuer, "issuer")
	t.Setenv(EnvJoinGrantAudience, "audience")
	t.Setenv(EnvJoinGrantPublicKey, base64.RawStdEncoding.EncodeToString(pubKey))

	cfg, err := LoadJoinGrantConfigFromEnv(nil)
	if err != nil {
		t.Fatalf("load join grant config: %v", err)
	}
	if cfg.Issuer != "issuer" || cfg.Audience != "audience" {
		t.Fatal("expected issuer and audience to be loaded")
	}
	if len(cfg.Key) != ed25519.PublicKeySize {
		t.Fatalf("expected public key size %d", ed25519.PublicKeySize)
	}
}

func TestValidateJoinGrantSuccess(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	now := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
	grant := signJoinGrant(t, priv, map[string]any{
		"alg": "EdDSA",
		"typ": "JWT",
	}, map[string]any{
		"iss":         "issuer",
		"aud":         []string{"game-service", "secondary"},
		"exp":         now.Add(2 * time.Hour).Unix(),
		"iat":         now.Add(-time.Minute).Unix(),
		"jti":         "jti-1",
		"campaign_id": "campaign-1",
		"invite_id":   "invite-1",
		"user_id":     "user-1",
	})

	cfg := JoinGrantConfig{Issuer: "issuer", Audience: "game-service", Key: pub, Now: func() time.Time { return now }}
	claims, err := ValidateJoinGrant(grant, JoinGrantExpectation{CampaignID: "campaign-1", InviteID: "invite-1", UserID: "user-1"}, cfg)
	if err != nil {
		t.Fatalf("validate join grant: %v", err)
	}
	if claims.Issuer != "issuer" {
		t.Fatalf("expected issuer claim issuer, got %s", claims.Issuer)
	}
	if claims.CampaignID != "campaign-1" || claims.InviteID != "invite-1" || claims.UserID != "user-1" {
		t.Fatal("expected campaign, invite, and user claims to match")
	}
	if !claims.ExpiresAt.Equal(time.Unix(now.Add(2*time.Hour).Unix(), 0).UTC()) {
		t.Fatal("expected expires at to match exp")
	}
}

func TestValidateJoinGrantExpired(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	now := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
	grant := signJoinGrant(t, priv, map[string]any{"alg": "EdDSA"}, map[string]any{
		"iss":         "issuer",
		"aud":         "game-service",
		"exp":         now.Add(-time.Minute).Unix(),
		"jti":         "jti-1",
		"campaign_id": "campaign-1",
		"invite_id":   "invite-1",
		"user_id":     "user-1",
	})

	cfg := JoinGrantConfig{Issuer: "issuer", Audience: "game-service", Key: pub, Now: func() time.Time { return now }}
	_, err = ValidateJoinGrant(grant, JoinGrantExpectation{CampaignID: "campaign-1", InviteID: "invite-1", UserID: "user-1"}, cfg)
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expired error, got %v", err)
	}
}

func TestValidateJoinGrantMismatch(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	now := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
	grant := signJoinGrant(t, priv, map[string]any{"alg": "EdDSA"}, map[string]any{
		"iss":         "issuer",
		"aud":         "game-service",
		"exp":         now.Add(time.Hour).Unix(),
		"jti":         "jti-1",
		"campaign_id": "campaign-1",
		"invite_id":   "invite-1",
		"user_id":     "user-2",
	})

	cfg := JoinGrantConfig{Issuer: "issuer", Audience: "game-service", Key: pub, Now: func() time.Time { return now }}
	_, err = ValidateJoinGrant(grant, JoinGrantExpectation{CampaignID: "campaign-1", InviteID: "invite-1", UserID: "user-1"}, cfg)
	if err == nil || !strings.Contains(err.Error(), "user mismatch") {
		t.Fatalf("expected user mismatch error, got %v", err)
	}
}

func TestValidateJoinGrantInvalidSignature(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	cfg := JoinGrantConfig{Issuer: "issuer", Audience: "game-service", Key: pub, Now: time.Now}
	_, err = ValidateJoinGrant("invalid.token.parts", JoinGrantExpectation{}, cfg)
	if err == nil {
		t.Fatal("expected error for invalid join grant")
	}
}

func signJoinGrant(t *testing.T, privateKey ed25519.PrivateKey, header, payload map[string]any) string {
	t.Helper()

	headerJSON, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := encodedHeader + "." + encodedPayload
	signature := ed25519.Sign(privateKey, []byte(signingInput))
	encodedSig := base64.RawURLEncoding.EncodeToString(signature)
	return signingInput + "." + encodedSig
}
