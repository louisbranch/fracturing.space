package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/id"
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
	// TODO: move GM fear cap to a config file.
	gmFearMax = 12
)

var (
	// ErrEmptyName indicates a missing campaign name.
	ErrEmptyName = errors.New("campaign name is required")
	// ErrInvalidGmMode indicates a missing or invalid GM mode.
	ErrInvalidGmMode = errors.New("gm mode is required")
	// ErrInvalidGMFearAmount indicates a non-positive fear mutation amount.
	ErrInvalidGMFearAmount = errors.New("gm fear amount must be greater than zero")
	// ErrInsufficientGMFear indicates the campaign has too little fear to spend.
	ErrInsufficientGMFear = errors.New("gm fear is insufficient")
	// ErrGMFearExceedsCap indicates the campaign fear would exceed the maximum.
	ErrGMFearExceedsCap = errors.New("gm fear exceeds cap")
)

// Campaign represents metadata for a campaign.
type Campaign struct {
	ID               string
	Name             string
	GmMode           GmMode
	ParticipantCount int
	CharacterCount   int
	// GmFear tracks the campaign-scoped GM fear resource.
	GmFear      int
	ThemePrompt string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CreateCampaignInput describes the metadata needed to create a campaign.
type CreateCampaignInput struct {
	Name        string
	GmMode      GmMode
	ThemePrompt string
}

// CreateCampaign creates a new campaign with a generated ID and timestamps.
func CreateCampaign(input CreateCampaignInput, now func() time.Time, idGenerator func() (string, error)) (Campaign, error) {
	if now == nil {
		now = time.Now
	}
	if idGenerator == nil {
		idGenerator = id.NewID
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
		ID:               campaignID,
		Name:             normalized.Name,
		GmMode:           normalized.GmMode,
		ParticipantCount: 0,
		CharacterCount:   0,
		GmFear:           0,
		ThemePrompt:      normalized.ThemePrompt,
		CreatedAt:        createdAt,
		UpdatedAt:        createdAt,
	}, nil
}

// ApplyGMFearGain returns a campaign with increased GM fear.
// Amount must be greater than zero.
func ApplyGMFearGain(campaign Campaign, amount int) (Campaign, int, int, error) {
	if amount <= 0 {
		return Campaign{}, 0, 0, ErrInvalidGMFearAmount
	}
	before := campaign.GmFear
	after := before + amount
	if after > gmFearMax {
		return Campaign{}, 0, 0, ErrGMFearExceedsCap
	}
	updated := campaign
	updated.GmFear = after
	return updated, before, updated.GmFear, nil
}

// ApplyGMFearSpend returns a campaign with reduced GM fear.
// Amount must be greater than zero and cannot exceed the current fear.
func ApplyGMFearSpend(campaign Campaign, amount int) (Campaign, int, int, error) {
	if amount <= 0 {
		return Campaign{}, 0, 0, ErrInvalidGMFearAmount
	}
	if campaign.GmFear < amount {
		return Campaign{}, 0, 0, ErrInsufficientGMFear
	}
	before := campaign.GmFear
	updated := campaign
	updated.GmFear = before - amount
	return updated, before, updated.GmFear, nil
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
	return input, nil
}
