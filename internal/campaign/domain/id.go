package domain

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
)

// NewID generates a URL-safe identifier using UUIDv4 bytes encoded as base32.
// The identifier is 26 characters long, lowercase, and contains no padding.
func NewID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}

	// RFC 4122 variant and version bits for a v4 UUID.
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80

	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw[:])
	return strings.ToLower(encoded), nil
}
