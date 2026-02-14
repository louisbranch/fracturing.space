package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

// joinGrantEnv holds raw env values before post-parse validation.
type joinGrantEnv struct {
	Issuer     string        `env:"FRACTURING_SPACE_JOIN_GRANT_ISSUER"`
	Audience   string        `env:"FRACTURING_SPACE_JOIN_GRANT_AUDIENCE"`
	PrivateKey string        `env:"FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY"`
	TTL        time.Duration `env:"FRACTURING_SPACE_JOIN_GRANT_TTL"         envDefault:"5m"`
}

type joinGrantConfig struct {
	issuer   string
	audience string
	key      ed25519.PrivateKey
	ttl      time.Duration
}

func loadJoinGrantConfigFromEnv() (joinGrantConfig, error) {
	var raw joinGrantEnv
	if err := env.Parse(&raw); err != nil {
		return joinGrantConfig{}, fmt.Errorf("parse join grant env: %w", err)
	}
	issuer := strings.TrimSpace(raw.Issuer)
	audience := strings.TrimSpace(raw.Audience)
	privateKey := strings.TrimSpace(raw.PrivateKey)
	if issuer == "" {
		return joinGrantConfig{}, fmt.Errorf("FRACTURING_SPACE_JOIN_GRANT_ISSUER is required")
	}
	if audience == "" {
		return joinGrantConfig{}, fmt.Errorf("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE is required")
	}
	if privateKey == "" {
		return joinGrantConfig{}, fmt.Errorf("FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY is required")
	}
	keyBytes, err := decodeBase64(privateKey)
	if err != nil {
		return joinGrantConfig{}, fmt.Errorf("decode join grant private key: %w", err)
	}
	if len(keyBytes) != ed25519.PrivateKeySize {
		return joinGrantConfig{}, fmt.Errorf("join grant private key must be %d bytes", ed25519.PrivateKeySize)
	}
	if raw.TTL <= 0 {
		return joinGrantConfig{}, fmt.Errorf("join grant ttl must be positive")
	}

	return joinGrantConfig{
		issuer:   issuer,
		audience: audience,
		key:      ed25519.PrivateKey(keyBytes),
		ttl:      raw.TTL,
	}, nil
}

func encodeJoinGrant(cfg joinGrantConfig, payload map[string]any) (string, error) {
	if cfg.issuer == "" || cfg.audience == "" || len(cfg.key) != ed25519.PrivateKeySize {
		return "", errors.New("join grant signer is not configured")
	}
	headerJSON, err := json.Marshal(map[string]string{
		"alg": "EdDSA",
		"typ": "JWT",
	})
	if err != nil {
		return "", fmt.Errorf("encode join grant header: %w", err)
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode join grant payload: %w", err)
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := encodedHeader + "." + encodedPayload
	signature := ed25519.Sign(cfg.key, []byte(signingInput))
	encodedSig := base64.RawURLEncoding.EncodeToString(signature)
	return signingInput + "." + encodedSig, nil
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
