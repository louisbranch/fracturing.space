package aisessiongrant

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
	// ErrInvalid indicates the session grant is malformed or fails verification.
	ErrInvalid = errors.New("session grant is invalid")
	// ErrExpired indicates the session grant is expired.
	ErrExpired = errors.New("session grant is expired")
)

// Config configures AI session grant signing and validation.
type Config struct {
	Issuer   string
	Audience string
	HMACKey  []byte
	TTL      time.Duration
	Now      func() time.Time
}

// Claims captures validated AI session grant claims.
type Claims struct {
	GrantID         string
	CampaignID      string
	SessionID       string
	ParticipantID   string
	AuthEpoch       uint64
	IssuedForUserID string
	IssuedAt        time.Time
	ExpiresAt       time.Time
}

// IssueInput is the payload required to mint one AI session grant.
type IssueInput struct {
	GrantID         string
	CampaignID      string
	SessionID       string
	ParticipantID   string
	AuthEpoch       uint64
	IssuedForUserID string
}

type envConfig struct {
	Issuer   string        `env:"FRACTURING_SPACE_AI_SESSION_GRANT_ISSUER" envDefault:"fracturing-space-game"`
	Audience string        `env:"FRACTURING_SPACE_AI_SESSION_GRANT_AUDIENCE" envDefault:"fracturing-space-ai"`
	HMACKey  string        `env:"FRACTURING_SPACE_AI_SESSION_GRANT_HMAC_KEY"`
	TTL      time.Duration `env:"FRACTURING_SPACE_AI_SESSION_GRANT_TTL" envDefault:"10m"`
}

type claimsPayload struct {
	jwt.RegisteredClaims
	CampaignID      string `json:"campaign_id"`
	SessionID       string `json:"session_id"`
	ParticipantID   string `json:"participant_id,omitempty"`
	AuthEpoch       uint64 `json:"auth_epoch"`
	IssuedForUserID string `json:"issued_for_user_id,omitempty"`
}

const maxIssuedAtSkew = 30 * time.Second

// LoadConfigFromEnv loads AI session grant configuration from environment.
func LoadConfigFromEnv(now func() time.Time) (Config, error) {
	var raw envConfig
	if err := env.Parse(&raw); err != nil {
		return Config{}, fmt.Errorf("parse ai session grant env: %w", err)
	}
	issuer := strings.TrimSpace(raw.Issuer)
	audience := strings.TrimSpace(raw.Audience)
	if issuer == "" {
		return Config{}, fmt.Errorf("FRACTURING_SPACE_AI_SESSION_GRANT_ISSUER is required")
	}
	if audience == "" {
		return Config{}, fmt.Errorf("FRACTURING_SPACE_AI_SESSION_GRANT_AUDIENCE is required")
	}
	encodedKey := strings.TrimSpace(raw.HMACKey)
	if encodedKey == "" {
		return Config{}, fmt.Errorf("FRACTURING_SPACE_AI_SESSION_GRANT_HMAC_KEY is required")
	}
	key, err := decodeBase64(encodedKey)
	if err != nil {
		return Config{}, fmt.Errorf("decode ai session grant hmac key: %w", err)
	}
	if len(key) < 32 {
		return Config{}, fmt.Errorf("ai session grant hmac key must be at least 32 bytes")
	}
	if raw.TTL <= 0 {
		return Config{}, fmt.Errorf("FRACTURING_SPACE_AI_SESSION_GRANT_TTL must be positive")
	}
	if now == nil {
		now = time.Now
	}
	return Config{
		Issuer:   issuer,
		Audience: audience,
		HMACKey:  key,
		TTL:      raw.TTL,
		Now:      now,
	}, nil
}

// Issue creates a signed session grant token and returns normalized claims.
func Issue(cfg Config, in IssueInput) (string, Claims, error) {
	if err := validateConfig(cfg); err != nil {
		return "", Claims{}, err
	}
	in.GrantID = strings.TrimSpace(in.GrantID)
	in.CampaignID = strings.TrimSpace(in.CampaignID)
	in.SessionID = strings.TrimSpace(in.SessionID)
	in.ParticipantID = strings.TrimSpace(in.ParticipantID)
	in.IssuedForUserID = strings.TrimSpace(in.IssuedForUserID)
	if in.GrantID == "" || in.CampaignID == "" || in.SessionID == "" {
		return "", Claims{}, fmt.Errorf("grant id, campaign id, and session id are required")
	}

	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn().UTC()
	expiresAt := now.Add(cfg.TTL)

	payload := claimsPayload{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			Audience:  jwt.ClaimStrings{cfg.Audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        in.GrantID,
		},
		CampaignID:      in.CampaignID,
		SessionID:       in.SessionID,
		ParticipantID:   in.ParticipantID,
		AuthEpoch:       in.AuthEpoch,
		IssuedForUserID: in.IssuedForUserID,
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	token, err := jwtToken.SignedString(cfg.HMACKey)
	if err != nil {
		return "", Claims{}, fmt.Errorf("sign session grant: %w", err)
	}

	claims := Claims{
		GrantID:         in.GrantID,
		CampaignID:      in.CampaignID,
		SessionID:       in.SessionID,
		ParticipantID:   in.ParticipantID,
		AuthEpoch:       in.AuthEpoch,
		IssuedForUserID: in.IssuedForUserID,
		IssuedAt:        now,
		ExpiresAt:       expiresAt,
	}
	return token, claims, nil
}

// Validate verifies a session grant token and returns normalized claims.
func Validate(cfg Config, token string) (Claims, error) {
	if err := validateConfig(cfg); err != nil {
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
	if strings.TrimSpace(parsed.ID) == "" ||
		strings.TrimSpace(parsed.CampaignID) == "" ||
		strings.TrimSpace(parsed.SessionID) == "" {
		return Claims{}, ErrInvalid
	}

	return Claims{
		GrantID:         strings.TrimSpace(parsed.ID),
		CampaignID:      strings.TrimSpace(parsed.CampaignID),
		SessionID:       strings.TrimSpace(parsed.SessionID),
		ParticipantID:   strings.TrimSpace(parsed.ParticipantID),
		AuthEpoch:       parsed.AuthEpoch,
		IssuedForUserID: strings.TrimSpace(parsed.IssuedForUserID),
		IssuedAt:        parsed.IssuedAt.Time.UTC(),
		ExpiresAt:       parsed.ExpiresAt.Time.UTC(),
	}, nil
}

func validateConfig(cfg Config) error {
	if strings.TrimSpace(cfg.Issuer) == "" || strings.TrimSpace(cfg.Audience) == "" || len(cfg.HMACKey) < 32 {
		return errors.New("ai session grant config is invalid")
	}
	if cfg.TTL <= 0 {
		return errors.New("ai session grant ttl must be positive")
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
