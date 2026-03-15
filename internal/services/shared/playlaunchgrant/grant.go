package playlaunchgrant

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalid indicates the play launch grant is malformed or fails verification.
	ErrInvalid = errors.New("play launch grant is invalid")
	// ErrExpired indicates the play launch grant is expired.
	ErrExpired = errors.New("play launch grant is expired")
)

// Config configures play launch grant signing and validation.
type Config struct {
	Issuer   string
	Audience string
	HMACKey  []byte
	TTL      time.Duration
	Now      func() time.Time
}

// Claims captures validated play launch grant claims.
type Claims struct {
	GrantID    string
	CampaignID string
	UserID     string
	IssuedAt   time.Time
	ExpiresAt  time.Time
}

// IssueInput stores the data required to mint one play launch grant.
type IssueInput struct {
	GrantID    string
	CampaignID string
	UserID     string
}

type envConfig struct {
	Issuer   string        `env:"FRACTURING_SPACE_PLAY_LAUNCH_GRANT_ISSUER" envDefault:"fracturing-space-web"`
	Audience string        `env:"FRACTURING_SPACE_PLAY_LAUNCH_GRANT_AUDIENCE" envDefault:"fracturing-space-play"`
	HMACKey  string        `env:"FRACTURING_SPACE_PLAY_LAUNCH_GRANT_HMAC_KEY"`
	TTL      time.Duration `env:"FRACTURING_SPACE_PLAY_LAUNCH_GRANT_TTL" envDefault:"2m"`
}

type claimsPayload struct {
	jwt.RegisteredClaims
	CampaignID string `json:"campaign_id"`
	UserID     string `json:"user_id"`
}

const maxIssuedAtSkew = 30 * time.Second

// LoadConfigFromEnv loads play launch grant configuration from process env.
func LoadConfigFromEnv(now func() time.Time) (Config, error) {
	var raw envConfig
	if err := env.Parse(&raw); err != nil {
		return Config{}, fmt.Errorf("parse play launch grant env: %w", err)
	}
	key, err := decodeBase64(strings.TrimSpace(raw.HMACKey))
	if err != nil {
		return Config{}, fmt.Errorf("decode play launch grant hmac key: %w", err)
	}
	if len(key) < 32 {
		return Config{}, fmt.Errorf("play launch grant hmac key must be at least 32 bytes")
	}
	if raw.TTL <= 0 {
		return Config{}, fmt.Errorf("FRACTURING_SPACE_PLAY_LAUNCH_GRANT_TTL must be positive")
	}
	if now == nil {
		now = time.Now
	}
	return Config{
		Issuer:   strings.TrimSpace(raw.Issuer),
		Audience: strings.TrimSpace(raw.Audience),
		HMACKey:  key,
		TTL:      raw.TTL,
		Now:      now,
	}, nil
}

// ValidateConfig verifies that play launch grant config is usable.
func ValidateConfig(cfg Config) error {
	return validateConfig(cfg)
}

// Issue creates a signed play launch grant and returns normalized claims.
func Issue(cfg Config, in IssueInput) (string, Claims, error) {
	if err := ValidateConfig(cfg); err != nil {
		return "", Claims{}, err
	}
	in.GrantID = strings.TrimSpace(in.GrantID)
	in.CampaignID = strings.TrimSpace(in.CampaignID)
	in.UserID = strings.TrimSpace(in.UserID)
	if in.GrantID == "" || in.CampaignID == "" || in.UserID == "" {
		return "", Claims{}, fmt.Errorf("grant id, campaign id, and user id are required")
	}

	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	issuedAt := nowFn().UTC()
	expiresAt := issuedAt.Add(cfg.TTL)
	payload := claimsPayload{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			Audience:  jwt.ClaimStrings{cfg.Audience},
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			NotBefore: jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        in.GrantID,
		},
		CampaignID: in.CampaignID,
		UserID:     in.UserID,
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, payload).SignedString(cfg.HMACKey)
	if err != nil {
		return "", Claims{}, fmt.Errorf("sign play launch grant: %w", err)
	}
	return token, Claims{
		GrantID:    in.GrantID,
		CampaignID: in.CampaignID,
		UserID:     in.UserID,
		IssuedAt:   issuedAt,
		ExpiresAt:  expiresAt,
	}, nil
}

// Validate verifies a play launch grant token and returns normalized claims.
func Validate(cfg Config, token string) (Claims, error) {
	if err := ValidateConfig(cfg); err != nil {
		return Claims{}, err
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return Claims{}, ErrInvalid
	}

	var parsed claimsPayload
	_, err := jwt.ParseWithClaims(token, &parsed, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalid
		}
		return cfg.HMACKey, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithoutClaimsValidation())
	if err != nil {
		return Claims{}, ErrInvalid
	}

	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn().UTC()
	if parsed.Issuer != cfg.Issuer || !audienceContains(parsed.Audience, cfg.Audience) {
		return Claims{}, ErrInvalid
	}
	if parsed.ExpiresAt == nil || parsed.IssuedAt == nil || parsed.NotBefore == nil {
		return Claims{}, ErrInvalid
	}
	if !parsed.ExpiresAt.Time.UTC().After(now) {
		return Claims{}, ErrExpired
	}
	if now.Before(parsed.NotBefore.Time.UTC()) {
		return Claims{}, ErrInvalid
	}
	if parsed.IssuedAt.Time.UTC().After(now.Add(maxIssuedAtSkew)) {
		return Claims{}, ErrInvalid
	}

	grantID := strings.TrimSpace(parsed.ID)
	campaignID := strings.TrimSpace(parsed.CampaignID)
	userID := strings.TrimSpace(parsed.UserID)
	if grantID == "" || campaignID == "" || userID == "" {
		return Claims{}, ErrInvalid
	}

	return Claims{
		GrantID:    grantID,
		CampaignID: campaignID,
		UserID:     userID,
		IssuedAt:   parsed.IssuedAt.Time.UTC(),
		ExpiresAt:  parsed.ExpiresAt.Time.UTC(),
	}, nil
}

func validateConfig(cfg Config) error {
	if strings.TrimSpace(cfg.Issuer) == "" || strings.TrimSpace(cfg.Audience) == "" || len(cfg.HMACKey) < 32 {
		return errors.New("play launch grant config is invalid")
	}
	if cfg.TTL <= 0 {
		return errors.New("play launch grant ttl must be positive")
	}
	return nil
}

func audienceContains(values jwt.ClaimStrings, expected string) bool {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return false
	}
	for _, item := range values {
		if strings.TrimSpace(item) == expected {
			return true
		}
	}
	return false
}

func decodeBase64(value string) ([]byte, error) {
	if value == "" {
		return nil, errors.New("empty base64 value")
	}
	decoded, err := base64.RawStdEncoding.DecodeString(value)
	if err == nil {
		return decoded, nil
	}
	decoded, err = base64.StdEncoding.DecodeString(value)
	if err == nil {
		return decoded, nil
	}
	return nil, err
}
