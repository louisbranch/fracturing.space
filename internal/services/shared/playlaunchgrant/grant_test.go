package playlaunchgrant

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestIssueAndValidate(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 13, 16, 0, 0, 0, time.UTC)
	cfg := Config{
		Issuer:   "issuer-test",
		Audience: "audience-test",
		HMACKey:  []byte("0123456789abcdef0123456789abcdef"),
		TTL:      time.Minute,
		Now:      func() time.Time { return now },
	}

	token, claims, err := Issue(cfg, IssueInput{
		GrantID:    "grant-1",
		CampaignID: "camp-1",
		UserID:     "user-1",
	})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if claims.CampaignID != "camp-1" || claims.UserID != "user-1" {
		t.Fatalf("Issue() claims = %#v", claims)
	}

	got, err := Validate(cfg, token)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if got.GrantID != "grant-1" || got.CampaignID != "camp-1" || got.UserID != "user-1" {
		t.Fatalf("Validate() claims = %#v", got)
	}
}

func TestValidateRejectsExpiredToken(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 13, 16, 0, 0, 0, time.UTC)
	key := []byte("0123456789abcdef0123456789abcdef")
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsPayload{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "issuer-test",
			Audience:  jwt.ClaimStrings{"audience-test"},
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Minute)),
			NotBefore: jwt.NewNumericDate(now.Add(-2 * time.Minute)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-time.Minute)),
			ID:        "grant-1",
		},
		CampaignID: "camp-1",
		UserID:     "user-1",
	}).SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	_, err = Validate(Config{
		Issuer:   "issuer-test",
		Audience: "audience-test",
		HMACKey:  key,
		TTL:      time.Minute,
		Now:      func() time.Time { return now },
	}, token)
	if !errors.Is(err, ErrExpired) {
		t.Fatalf("Validate() error = %v, want %v", err, ErrExpired)
	}
}

func TestLoadConfigFromEnvParsesConfiguredKey(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_PLAY_LAUNCH_GRANT_ISSUER", "issuer-test")
	t.Setenv("FRACTURING_SPACE_PLAY_LAUNCH_GRANT_AUDIENCE", "audience-test")
	t.Setenv("FRACTURING_SPACE_PLAY_LAUNCH_GRANT_HMAC_KEY", base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	t.Setenv("FRACTURING_SPACE_PLAY_LAUNCH_GRANT_TTL", "45s")

	cfg, err := LoadConfigFromEnv(func() time.Time { return time.Unix(0, 0).UTC() })
	if err != nil {
		t.Fatalf("LoadConfigFromEnv() error = %v", err)
	}
	if cfg.Issuer != "issuer-test" || cfg.Audience != "audience-test" || cfg.TTL != 45*time.Second {
		t.Fatalf("LoadConfigFromEnv() cfg = %#v", cfg)
	}
}
