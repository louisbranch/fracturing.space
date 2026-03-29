package playlaunchgrant

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestLoadConfigFromEnvBranches(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")

	t.Run("standard base64 and default now", func(t *testing.T) {
		t.Setenv("FRACTURING_SPACE_PLAY_LAUNCH_GRANT_ISSUER", " issuer-test ")
		t.Setenv("FRACTURING_SPACE_PLAY_LAUNCH_GRANT_AUDIENCE", " audience-test ")
		t.Setenv("FRACTURING_SPACE_PLAY_LAUNCH_GRANT_HMAC_KEY", base64.StdEncoding.EncodeToString(key))
		t.Setenv("FRACTURING_SPACE_PLAY_LAUNCH_GRANT_TTL", "90s")

		cfg, err := LoadConfigFromEnv(nil)
		if err != nil {
			t.Fatalf("LoadConfigFromEnv() error = %v", err)
		}
		if cfg.Issuer != "issuer-test" || cfg.Audience != "audience-test" || cfg.TTL != 90*time.Second {
			t.Fatalf("LoadConfigFromEnv() cfg = %#v", cfg)
		}
		if cfg.Now == nil {
			t.Fatal("LoadConfigFromEnv() Now = nil, want default time source")
		}
	})

	t.Run("invalid base64", func(t *testing.T) {
		t.Setenv("FRACTURING_SPACE_PLAY_LAUNCH_GRANT_HMAC_KEY", "%%%")
		_, err := LoadConfigFromEnv(func() time.Time { return time.Unix(0, 0).UTC() })
		if err == nil || !strings.Contains(err.Error(), "decode play launch grant hmac key") {
			t.Fatalf("LoadConfigFromEnv() error = %v, want decode error", err)
		}
	})

	t.Run("short key", func(t *testing.T) {
		t.Setenv("FRACTURING_SPACE_PLAY_LAUNCH_GRANT_HMAC_KEY", base64.RawStdEncoding.EncodeToString([]byte("short")))
		_, err := LoadConfigFromEnv(func() time.Time { return time.Unix(0, 0).UTC() })
		if err == nil || !strings.Contains(err.Error(), "at least 32 bytes") {
			t.Fatalf("LoadConfigFromEnv() error = %v, want short-key error", err)
		}
	})

	t.Run("negative ttl", func(t *testing.T) {
		t.Setenv("FRACTURING_SPACE_PLAY_LAUNCH_GRANT_HMAC_KEY", base64.RawStdEncoding.EncodeToString(key))
		t.Setenv("FRACTURING_SPACE_PLAY_LAUNCH_GRANT_TTL", "-1s")
		_, err := LoadConfigFromEnv(func() time.Time { return time.Unix(0, 0).UTC() })
		if err == nil || !strings.Contains(err.Error(), "must be positive") {
			t.Fatalf("LoadConfigFromEnv() error = %v, want ttl error", err)
		}
	})
}

func TestIssueValidateAndHelpersBranches(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 28, 18, 0, 0, 0, time.UTC)
	cfg := Config{
		Issuer:   "issuer-test",
		Audience: "audience-test",
		HMACKey:  []byte("0123456789abcdef0123456789abcdef"),
		TTL:      time.Minute,
		Now:      func() time.Time { return now },
	}

	if err := ValidateConfig(Config{}); err == nil {
		t.Fatal("ValidateConfig(invalid) error = nil, want invalid config")
	}
	if _, _, err := Issue(cfg, IssueInput{GrantID: "grant-1"}); err == nil {
		t.Fatal("Issue(missing fields) error = nil, want required-field error")
	}
	if _, err := Validate(cfg, " "); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Validate(empty token) error = %v, want %v", err, ErrInvalid)
	}

	token, claims, err := Issue(cfg, IssueInput{
		GrantID:    " grant-1 ",
		CampaignID: " camp-1 ",
		UserID:     " user-1 ",
	})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if claims.GrantID != "grant-1" || claims.CampaignID != "camp-1" || claims.UserID != "user-1" {
		t.Fatalf("Issue() claims = %#v", claims)
	}

	validated, err := Validate(cfg, token)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if validated != claims {
		t.Fatalf("Validate() claims = %#v, want %#v", validated, claims)
	}

	missingTimeToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsPayload{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:   cfg.Issuer,
			Audience: jwt.ClaimStrings{cfg.Audience},
			ID:       "grant-1",
		},
		CampaignID: "camp-1",
		UserID:     "user-1",
	}).SignedString(cfg.HMACKey)
	if err != nil {
		t.Fatalf("sign missing-time token: %v", err)
	}
	if _, err := Validate(cfg, missingTimeToken); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Validate(missing times) error = %v, want %v", err, ErrInvalid)
	}

	futureToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsPayload{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			Audience:  jwt.ClaimStrings{cfg.Audience},
			IssuedAt:  jwt.NewNumericDate(now.Add(maxIssuedAtSkew + time.Second)),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
			ID:        "grant-1",
		},
		CampaignID: "camp-1",
		UserID:     "user-1",
	}).SignedString(cfg.HMACKey)
	if err != nil {
		t.Fatalf("sign future token: %v", err)
	}
	if _, err := Validate(cfg, futureToken); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Validate(future issued at) error = %v, want %v", err, ErrInvalid)
	}

	notBeforeToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsPayload{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			Audience:  jwt.ClaimStrings{cfg.Audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(time.Second)),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
			ID:        "grant-1",
		},
		CampaignID: "camp-1",
		UserID:     "user-1",
	}).SignedString(cfg.HMACKey)
	if err != nil {
		t.Fatalf("sign not-before token: %v", err)
	}
	if _, err := Validate(cfg, notBeforeToken); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Validate(not before) error = %v, want %v", err, ErrInvalid)
	}

	wrongAudienceToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsPayload{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			Audience:  jwt.ClaimStrings{"other"},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
			ID:        "grant-1",
		},
		CampaignID: "camp-1",
		UserID:     "user-1",
	}).SignedString(cfg.HMACKey)
	if err != nil {
		t.Fatalf("sign wrong-audience token: %v", err)
	}
	if _, err := Validate(cfg, wrongAudienceToken); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Validate(wrong audience) error = %v, want %v", err, ErrInvalid)
	}

	blankClaimsToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsPayload{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			Audience:  jwt.ClaimStrings{cfg.Audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
			ID:        " ",
		},
		CampaignID: " ",
		UserID:     " ",
	}).SignedString(cfg.HMACKey)
	if err != nil {
		t.Fatalf("sign blank-claims token: %v", err)
	}
	if _, err := Validate(cfg, blankClaimsToken); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Validate(blank claims) error = %v, want %v", err, ErrInvalid)
	}

	if got := audienceContains(jwt.ClaimStrings{" one ", "two"}, "one"); !got {
		t.Fatal("audienceContains(trimmed match) = false, want true")
	}
	if got := audienceContains(jwt.ClaimStrings{"one"}, " "); got {
		t.Fatal("audienceContains(blank expected) = true, want false")
	}

	if decoded, err := decodeBase64(base64.StdEncoding.EncodeToString([]byte("hello world"))); err != nil || string(decoded) != "hello world" {
		t.Fatalf("decodeBase64(std) = (%q, %v)", string(decoded), err)
	}
	if _, err := decodeBase64(""); err == nil {
		t.Fatal("decodeBase64(empty) error = nil, want error")
	}
}
