// Package recoverycode generates and verifies offline single-use recovery codes.
package recoverycode

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	saltBytes     = 16
	codeBytes     = 20
	argonTime     = 3
	argonMemoryKB = 64 * 1024
	argonThreads  = 4
	argonKeyBytes = 32
)

// Generate returns one display-once recovery code and its stored hash.
func Generate(reader io.Reader) (string, string, error) {
	if reader == nil {
		reader = rand.Reader
	}
	codeRaw := make([]byte, codeBytes)
	if _, err := io.ReadFull(reader, codeRaw); err != nil {
		return "", "", fmt.Errorf("Read recovery code entropy: %w", err)
	}
	code := encodeDisplayCode(codeRaw)
	hash, err := Hash(code, reader)
	if err != nil {
		return "", "", err
	}
	return code, hash, nil
}

// Hash derives the stored hash for one recovery code.
func Hash(code string, reader io.Reader) (string, error) {
	if reader == nil {
		reader = rand.Reader
	}
	code = Normalize(code)
	if code == "" {
		return "", fmt.Errorf("Recovery code is required.")
	}
	salt := make([]byte, saltBytes)
	if _, err := io.ReadFull(reader, salt); err != nil {
		return "", fmt.Errorf("Read recovery salt entropy: %w", err)
	}
	key := deriveKey(code, salt)
	return fmt.Sprintf(
		"argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemoryKB,
		argonTime,
		argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// Verify checks whether one recovery code matches the stored hash.
func Verify(code string, encoded string) bool {
	code = Normalize(code)
	if code == "" || strings.TrimSpace(encoded) == "" {
		return false
	}
	salt, expected, err := parseHash(encoded)
	if err != nil {
		return false
	}
	actual := deriveKey(code, salt)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

// Normalize removes formatting separators so users can paste grouped codes.
func Normalize(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	replacer := strings.NewReplacer("-", "", " ", "", "\n", "", "\r", "", "\t", "")
	return replacer.Replace(code)
}

func encodeDisplayCode(raw []byte) string {
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
	var parts []string
	for len(encoded) > 0 {
		n := 4
		if len(encoded) < n {
			n = len(encoded)
		}
		parts = append(parts, encoded[:n])
		encoded = encoded[n:]
	}
	return strings.Join(parts, "-")
}

func parseHash(encoded string) ([]byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 || parts[0] != "argon2id" || parts[1] != "v=19" {
		return nil, nil, fmt.Errorf("Invalid recovery hash format.")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return nil, nil, fmt.Errorf("Decode recovery salt: %w", err)
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, fmt.Errorf("Decode recovery key: %w", err)
	}
	return salt, key, nil
}

func deriveKey(code string, salt []byte) []byte {
	return argon2.IDKey([]byte(code), salt, argonTime, argonMemoryKB, argonThreads, argonKeyBytes)
}
