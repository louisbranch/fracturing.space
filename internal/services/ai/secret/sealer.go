package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// AESGCMSealer seals and opens secrets using AES-GCM.
type AESGCMSealer struct {
	aead cipher.AEAD
}

// NewAESGCMSealer builds an AES-GCM sealer from a raw AES key.
// key must be a valid AES length (16/24/32 bytes).
func NewAESGCMSealer(key []byte) (*AESGCMSealer, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}
	return &AESGCMSealer{aead: aead}, nil
}

// Seal encrypts one plaintext value and returns a base64-encoded payload.
func (s *AESGCMSealer) Seal(value string) (string, error) {
	if s == nil || s.aead == nil {
		return "", fmt.Errorf("sealer is not configured")
	}

	// AES-GCM requires a unique nonce per encryption under the same key.
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("read nonce: %w", err)
	}

	ciphertext := s.aead.Seal(nil, nonce, []byte(value), nil)
	// Persist as nonce || ciphertext, encoded in raw base64 for storage.
	payload := append(nonce, ciphertext...)
	return base64.RawStdEncoding.EncodeToString(payload), nil
}

// Open decrypts one previously sealed value.
func (s *AESGCMSealer) Open(sealed string) (string, error) {
	if s == nil || s.aead == nil {
		return "", fmt.Errorf("sealer is not configured")
	}

	payload, err := base64.RawStdEncoding.DecodeString(sealed)
	if err != nil {
		return "", fmt.Errorf("decode sealed value: %w", err)
	}

	nonceSize := s.aead.NonceSize()
	if len(payload) < nonceSize {
		return "", fmt.Errorf("sealed value is too short")
	}
	// Payload format is nonce || ciphertext.
	nonce := payload[:nonceSize]
	ciphertext := payload[nonceSize:]
	plaintext, err := s.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt sealed value: %w", err)
	}
	return string(plaintext), nil
}
