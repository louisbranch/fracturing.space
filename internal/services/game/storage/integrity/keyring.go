package integrity

import (
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Keyring stores root HMAC keys and tracks the currently active signing key id.
//
// It intentionally isolates key selection and campaign-specific derivation so
// callers can rotate keys without touching hash verification logic.
type Keyring struct {
	keys        map[string][]byte
	activeKeyID string
}

// NewKeyring validates and constructs the campaign keyring used for chain signatures.
func NewKeyring(keys map[string][]byte, activeKeyID string) (*Keyring, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("hmac keys are required")
	}
	activeKeyID = strings.TrimSpace(activeKeyID)
	if activeKeyID == "" {
		return nil, fmt.Errorf("active hmac key id is required")
	}
	if _, ok := keys[activeKeyID]; !ok {
		return nil, fmt.Errorf("active hmac key id is not configured")
	}
	return &Keyring{keys: keys, activeKeyID: activeKeyID}, nil
}

// ActiveKeyID returns the currently configured signing key id.
func (k *Keyring) ActiveKeyID() string {
	if k == nil {
		return ""
	}
	return k.activeKeyID
}

// SignChainHash signs a chain hash with the active key.
func (k *Keyring) SignChainHash(campaignID, chainHash string) (string, string, error) {
	if k == nil {
		return "", "", fmt.Errorf("hmac keyring is not configured")
	}
	keyID := k.activeKeyID
	key, err := k.deriveKey(keyID, campaignID)
	if err != nil {
		return "", "", err
	}
	sig := hmacSHA256Hex(key, chainHash)
	return sig, keyID, nil
}

// VerifyChainHash validates a chain hash signature for the campaign using the
// supplied key id. This is the integrity checkpoint for replay acceptance.
func (k *Keyring) VerifyChainHash(campaignID, chainHash, signature, keyID string) error {
	if k == nil {
		return fmt.Errorf("hmac keyring is not configured")
	}
	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return fmt.Errorf("signature key id is required")
	}
	rootKey, ok := k.keys[keyID]
	if !ok {
		return fmt.Errorf("signature key id is unknown")
	}
	key, err := deriveCampaignKey(rootKey, campaignID)
	if err != nil {
		return err
	}
	expected := hmacSHA256Hex(key, chainHash)
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

func (k *Keyring) deriveKey(keyID, campaignID string) ([]byte, error) {
	rootKey, ok := k.keys[keyID]
	if !ok {
		return nil, fmt.Errorf("hmac key id is unknown")
	}
	return deriveCampaignKey(rootKey, campaignID)
}

func deriveCampaignKey(rootKey []byte, campaignID string) ([]byte, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	key, err := hkdf.Key(sha256.New, rootKey, nil, "campaign:"+campaignID, 32)
	if err != nil {
		return nil, fmt.Errorf("derive campaign key: %w", err)
	}
	return key, nil
}

func hmacSHA256Hex(key []byte, value string) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}
