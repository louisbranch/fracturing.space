package invite

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/golang-jwt/jwt/v5"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
)

// joinGrantEnv holds raw env values before post-parse validation.
type joinGrantEnv struct {
	Issuer    string `env:"FRACTURING_SPACE_JOIN_GRANT_ISSUER"`
	Audience  string `env:"FRACTURING_SPACE_JOIN_GRANT_AUDIENCE"`
	PublicKey string `env:"FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY"`
}

// JoinGrantConfig defines how join grants are verified.
type JoinGrantConfig struct {
	Issuer   string
	Audience string
	Key      ed25519.PublicKey
	Now      func() time.Time
}

// JoinGrantExpectation defines the expected identity for a join grant.
type JoinGrantExpectation struct {
	CampaignID string
	InviteID   string
	UserID     string
}

// JoinGrantClaims captures validated join grant claims.
type JoinGrantClaims struct {
	Issuer     string
	Audience   []string
	ExpiresAt  time.Time
	NotBefore  time.Time
	IssuedAt   time.Time
	JWTID      string
	CampaignID string
	InviteID   string
	UserID     string
}

// joinGrantClaims is the internal claims type used for JWT parsing.
type joinGrantClaims struct {
	jwt.RegisteredClaims
	CampaignID string `json:"campaign_id"`
	InviteID   string `json:"invite_id"`
	UserID     string `json:"user_id"`
}

// LoadJoinGrantConfigFromEnv reads join grant verification configuration.
func LoadJoinGrantConfigFromEnv(now func() time.Time) (JoinGrantConfig, error) {
	var raw joinGrantEnv
	if err := env.Parse(&raw); err != nil {
		return JoinGrantConfig{}, fmt.Errorf("parse join grant env: %w", err)
	}
	issuer := strings.TrimSpace(raw.Issuer)
	audience := strings.TrimSpace(raw.Audience)
	publicKey := strings.TrimSpace(raw.PublicKey)
	if issuer == "" {
		return JoinGrantConfig{}, fmt.Errorf("FRACTURING_SPACE_JOIN_GRANT_ISSUER is required")
	}
	if audience == "" {
		return JoinGrantConfig{}, fmt.Errorf("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE is required")
	}
	if publicKey == "" {
		return JoinGrantConfig{}, fmt.Errorf("FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY is required")
	}
	keyBytes, err := decodeBase64(publicKey)
	if err != nil {
		return JoinGrantConfig{}, fmt.Errorf("decode join grant public key: %w", err)
	}
	if len(keyBytes) != ed25519.PublicKeySize {
		return JoinGrantConfig{}, fmt.Errorf("join grant public key must be %d bytes", ed25519.PublicKeySize)
	}
	if now == nil {
		now = time.Now
	}
	return JoinGrantConfig{
		Issuer:   issuer,
		Audience: audience,
		Key:      ed25519.PublicKey(keyBytes),
		Now:      now,
	}, nil
}

// ValidateJoinGrant verifies a join grant token and validates expected claims.
func ValidateJoinGrant(grant string, expected JoinGrantExpectation, cfg JoinGrantConfig) (JoinGrantClaims, error) {
	grant = strings.TrimSpace(grant)
	if grant == "" {
		return JoinGrantClaims{}, apperrors.New(apperrors.CodeInviteJoinGrantInvalid, "join grant is required")
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Issuer == "" || cfg.Audience == "" || len(cfg.Key) != ed25519.PublicKeySize {
		return JoinGrantClaims{}, errors.New("join grant verifier is not configured")
	}

	var parsed joinGrantClaims
	_, err := jwt.ParseWithClaims(grant, &parsed, func(token *jwt.Token) (any, error) {
		return cfg.Key, nil
	},
		jwt.WithValidMethods([]string{"EdDSA"}),
		jwt.WithoutClaimsValidation(),
	)
	if err != nil {
		return JoinGrantClaims{}, mapJWTError(err)
	}

	if parsed.Issuer == "" || parsed.Issuer != cfg.Issuer {
		return JoinGrantClaims{}, apperrors.WithMetadata(
			apperrors.CodeInviteJoinGrantMismatch,
			"join grant issuer mismatch",
			map[string]string{"Field": "issuer"},
		)
	}
	if !audienceContains(parsed.Audience, cfg.Audience) {
		return JoinGrantClaims{}, apperrors.WithMetadata(
			apperrors.CodeInviteJoinGrantMismatch,
			"join grant audience mismatch",
			map[string]string{"Field": "audience"},
		)
	}

	if parsed.ID == "" {
		return JoinGrantClaims{}, apperrors.New(apperrors.CodeInviteJoinGrantInvalid, "join grant jti is required")
	}
	if parsed.ExpiresAt == nil {
		return JoinGrantClaims{}, apperrors.New(apperrors.CodeInviteJoinGrantInvalid, "join grant exp is required")
	}

	now := cfg.Now().UTC()
	exp := parsed.ExpiresAt.Time.UTC()
	if !exp.After(now) {
		return JoinGrantClaims{}, apperrors.New(apperrors.CodeInviteJoinGrantExpired, "join grant is expired")
	}
	if parsed.NotBefore != nil {
		nbf := parsed.NotBefore.Time.UTC()
		if now.Before(nbf) {
			return JoinGrantClaims{}, apperrors.New(apperrors.CodeInviteJoinGrantInvalid, "join grant not active yet")
		}
	}

	if strings.TrimSpace(parsed.CampaignID) == "" || parsed.CampaignID != expected.CampaignID {
		return JoinGrantClaims{}, apperrors.WithMetadata(
			apperrors.CodeInviteJoinGrantMismatch,
			"join grant campaign mismatch",
			map[string]string{"Field": "campaign_id"},
		)
	}
	if strings.TrimSpace(parsed.InviteID) == "" || parsed.InviteID != expected.InviteID {
		return JoinGrantClaims{}, apperrors.WithMetadata(
			apperrors.CodeInviteJoinGrantMismatch,
			"join grant invite mismatch",
			map[string]string{"Field": "invite_id"},
		)
	}
	if strings.TrimSpace(parsed.UserID) == "" || parsed.UserID != expected.UserID {
		return JoinGrantClaims{}, apperrors.WithMetadata(
			apperrors.CodeInviteJoinGrantMismatch,
			"join grant user mismatch",
			map[string]string{"Field": "user_id"},
		)
	}

	claims := JoinGrantClaims{
		Issuer:     parsed.Issuer,
		Audience:   []string(parsed.Audience),
		ExpiresAt:  exp,
		JWTID:      parsed.ID,
		CampaignID: parsed.CampaignID,
		InviteID:   parsed.InviteID,
		UserID:     parsed.UserID,
	}
	if parsed.NotBefore != nil {
		claims.NotBefore = parsed.NotBefore.Time.UTC()
	}
	if parsed.IssuedAt != nil {
		claims.IssuedAt = parsed.IssuedAt.Time.UTC()
	}
	return claims, nil
}

// mapJWTError translates jwt library errors to application errors.
func mapJWTError(err error) error {
	if errors.Is(err, jwt.ErrTokenSignatureInvalid) || errors.Is(err, jwt.ErrEd25519Verification) {
		return apperrors.New(apperrors.CodeInviteJoinGrantInvalid, "join grant signature is invalid")
	}
	if errors.Is(err, jwt.ErrTokenUnverifiable) {
		return apperrors.New(apperrors.CodeInviteJoinGrantInvalid, "join grant alg is invalid")
	}
	return apperrors.New(apperrors.CodeInviteJoinGrantInvalid, "join grant is invalid")
}

// audienceContains reports whether the audience list contains the given value.
func audienceContains(aud jwt.ClaimStrings, value string) bool {
	for _, item := range aud {
		if item == value {
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
	return base64.StdEncoding.DecodeString(value)
}
