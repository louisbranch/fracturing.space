package server

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	setIfMissing("FRACTURING_SPACE_AI_SESSION_GRANT_ISSUER", "fracturing-space-game")
	setIfMissing("FRACTURING_SPACE_AI_SESSION_GRANT_AUDIENCE", "fracturing-space-ai")
	setIfMissing(
		"FRACTURING_SPACE_AI_SESSION_GRANT_HMAC_KEY",
		base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")),
	)
	setIfMissing("FRACTURING_SPACE_AI_SESSION_GRANT_TTL", "10m")
	os.Exit(m.Run())
}

func setIfMissing(key string, value string) {
	if strings.TrimSpace(os.Getenv(key)) != "" {
		return
	}
	_ = os.Setenv(key, value)
}
