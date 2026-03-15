package aisessiongrant

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestLoadConfigFromEnvRequiresHMACKey(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_ISSUER", "issuer-test")
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_AUDIENCE", "audience-test")
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_HMAC_KEY", "")
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_TTL", "10m")

	_, err := LoadConfigFromEnv(nil)
	if err == nil {
		t.Fatal("expected error when session grant hmac key is not configured")
	}
}

func TestValidateRejectsFutureIssuedAt(t *testing.T) {
	now := time.Date(2026, 3, 2, 5, 0, 0, 0, time.UTC)
	key := []byte("0123456789abcdef0123456789abcdef")
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsPayload{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "issuer-test",
			Audience:  jwt.ClaimStrings{"audience-test"},
			IssuedAt:  jwt.NewNumericDate(now.Add(maxIssuedAtSkew + time.Second)),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)),
			ID:        "grant-1",
		},
		CampaignID:    "campaign-1",
		SessionID:     "session-1",
		ParticipantID: "gm-1",
		AuthEpoch:     1,
	}).SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	_, err = Validate(Config{
		Issuer:   "issuer-test",
		Audience: "audience-test",
		HMACKey:  key,
		TTL:      10 * time.Minute,
		Now:      func() time.Time { return now },
	}, token)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("validate error = %v, want %v", err, ErrInvalid)
	}
}

func TestLoadConfigFromEnvParsesConfiguredKey(t *testing.T) {
	now := time.Date(2026, 3, 2, 5, 0, 0, 0, time.UTC)
	key := base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_ISSUER", "issuer-test")
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_AUDIENCE", "audience-test")
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_HMAC_KEY", key)
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_TTL", "10m")

	cfg, err := LoadConfigFromEnv(func() time.Time { return now })
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Issuer != "issuer-test" || cfg.Audience != "audience-test" {
		t.Fatalf("unexpected config %+v", cfg)
	}
}
