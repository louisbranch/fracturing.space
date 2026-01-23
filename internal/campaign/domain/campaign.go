package domain

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
	"time"
)

// GmMode describes how the GM role is handled for a campaign.
type GmMode int

const (
	// GmModeUnspecified represents an invalid GM mode value.
	GmModeUnspecified GmMode = iota
	// GmModeHuman indicates a human GM.
	GmModeHuman
	// GmModeAI indicates an AI GM.
	GmModeAI
	// GmModeHybrid indicates a mixed human and AI GM.
	GmModeHybrid
)

var (
	// ErrEmptyName indicates a missing campaign name.
	ErrEmptyName = errors.New("campaign name is required")
	// ErrInvalidGmMode indicates a missing or invalid GM mode.
	ErrInvalidGmMode = errors.New("gm mode is required")
	// ErrInvalidPlayerSlots indicates an invalid player slots value.
	ErrInvalidPlayerSlots = errors.New("player slots must be greater than zero")
)

// Campaign represents metadata for a campaign.
type Campaign struct {
	ID          string
	Name        string
	GmMode      GmMode
	PlayerSlots int
	ThemePrompt string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CreateCampaignInput describes the metadata needed to create a campaign.
type CreateCampaignInput struct {
	Name        string
	GmMode      GmMode
	PlayerSlots int
	ThemePrompt string
}

// CreateCampaign creates a new campaign with a generated ID and timestamps.
func CreateCampaign(input CreateCampaignInput, now func() time.Time, idGenerator func() (string, error)) (Campaign, error) {
	if now == nil {
		now = time.Now
	}
	if idGenerator == nil {
		idGenerator = NewCampaignID
	}

	normalized, err := NormalizeCreateCampaignInput(input)
	if err != nil {
		return Campaign{}, err
	}

	campaignID, err := idGenerator()
	if err != nil {
		return Campaign{}, fmt.Errorf("generate campaign id: %w", err)
	}

	createdAt := now().UTC()
	return Campaign{
		ID:          campaignID,
		Name:        normalized.Name,
		GmMode:      normalized.GmMode,
		PlayerSlots: normalized.PlayerSlots,
		ThemePrompt: normalized.ThemePrompt,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}, nil
}

// NormalizeCreateCampaignInput trims and validates campaign input metadata.
func NormalizeCreateCampaignInput(input CreateCampaignInput) (CreateCampaignInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return CreateCampaignInput{}, ErrEmptyName
	}
	if input.GmMode == GmModeUnspecified {
		return CreateCampaignInput{}, ErrInvalidGmMode
	}
	if input.PlayerSlots <= 0 {
		return CreateCampaignInput{}, ErrInvalidPlayerSlots
	}
	return input, nil
}

// NewCampaignID generates a URL-safe campaign identifier.
func NewCampaignID() (string, error) {
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
