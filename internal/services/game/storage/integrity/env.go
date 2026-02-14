package integrity

import (
	"fmt"
	"strings"

	"github.com/caarlos0/env/v11"
)

// keyringEnv holds raw env values for HMAC keyring configuration.
type keyringEnv struct {
	Keys  string `env:"FRACTURING_SPACE_GAME_EVENT_HMAC_KEYS"`
	Key   string `env:"FRACTURING_SPACE_GAME_EVENT_HMAC_KEY"`
	KeyID string `env:"FRACTURING_SPACE_GAME_EVENT_HMAC_KEY_ID" envDefault:"v1"`
}

// KeyringFromEnv loads the HMAC keyring configuration from environment variables.
func KeyringFromEnv() (*Keyring, error) {
	var raw keyringEnv
	if err := env.Parse(&raw); err != nil {
		return nil, fmt.Errorf("parse hmac keyring env: %w", err)
	}
	raw.Keys = strings.TrimSpace(raw.Keys)
	raw.Key = strings.TrimSpace(raw.Key)
	raw.KeyID = strings.TrimSpace(raw.KeyID)
	if raw.KeyID == "" {
		raw.KeyID = "v1"
	}

	if raw.Keys == "" {
		if raw.Key == "" {
			return nil, fmt.Errorf("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY is required")
		}
		return NewKeyring(map[string][]byte{raw.KeyID: []byte(raw.Key)}, raw.KeyID)
	}

	keys := make(map[string][]byte)
	entries := strings.Split(raw.Keys, ",")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid FRACTURING_SPACE_GAME_EVENT_HMAC_KEYS entry")
		}
		id := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if id == "" || value == "" {
			return nil, fmt.Errorf("invalid FRACTURING_SPACE_GAME_EVENT_HMAC_KEYS entry")
		}
		keys[id] = []byte(value)
	}
	return NewKeyring(keys, raw.KeyID)
}
