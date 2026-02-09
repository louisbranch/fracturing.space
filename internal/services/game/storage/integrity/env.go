package integrity

import (
	"fmt"
	"os"
	"strings"
)

const (
	envHMACKeys  = "FRACTURING_SPACE_GAME_EVENT_HMAC_KEYS"
	envHMACKey   = "FRACTURING_SPACE_GAME_EVENT_HMAC_KEY"
	envHMACKeyID = "FRACTURING_SPACE_GAME_EVENT_HMAC_KEY_ID"
	defaultKeyID = "v1"
)

// KeyringFromEnv loads the HMAC keyring configuration from environment variables.
func KeyringFromEnv() (*Keyring, error) {
	keyID := strings.TrimSpace(os.Getenv(envHMACKeyID))
	if keyID == "" {
		keyID = defaultKeyID
	}

	keySpec := strings.TrimSpace(os.Getenv(envHMACKeys))
	if keySpec == "" {
		raw := strings.TrimSpace(os.Getenv(envHMACKey))
		if raw == "" {
			return nil, fmt.Errorf("%s is required", envHMACKey)
		}
		return NewKeyring(map[string][]byte{keyID: []byte(raw)}, keyID)
	}

	keys := make(map[string][]byte)
	entries := strings.Split(keySpec, ",")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid %s entry", envHMACKeys)
		}
		id := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if id == "" || value == "" {
			return nil, fmt.Errorf("invalid %s entry", envHMACKeys)
		}
		keys[id] = []byte(value)
	}
	return NewKeyring(keys, keyID)
}
