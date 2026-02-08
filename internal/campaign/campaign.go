package campaign

import (
	"fmt"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/errors"
	"github.com/louisbranch/fracturing.space/internal/id"
)

// GmMode describes how the GM role is handled for a campaign.
type GmMode int

// CampaignStatus describes the lifecycle of a campaign.
type CampaignStatus int

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

const (
	// CampaignStatusUnspecified represents an invalid campaign status value.
	CampaignStatusUnspecified CampaignStatus = iota
	// CampaignStatusDraft indicates the campaign is in draft mode.
	CampaignStatusDraft
	// CampaignStatusActive indicates the campaign is active.
	CampaignStatusActive
	// CampaignStatusCompleted indicates the campaign is completed.
	CampaignStatusCompleted
	// CampaignStatusArchived indicates the campaign is archived.
	CampaignStatusArchived
)

var (
	// ErrEmptyName indicates a missing campaign name.
	ErrEmptyName = apperrors.New(apperrors.CodeCampaignNameEmpty, "campaign name is required")
	// ErrInvalidGmMode indicates a missing or invalid GM mode.
	ErrInvalidGmMode = apperrors.New(apperrors.CodeCampaignInvalidGmMode, "gm mode is required")
	// ErrInvalidGameSystem indicates a missing or invalid game system.
	ErrInvalidGameSystem = apperrors.New(apperrors.CodeCampaignInvalidGameSystem, "game system is required")
	// ErrInvalidCampaignStatusTransition indicates a disallowed campaign status change.
	ErrInvalidCampaignStatusTransition = apperrors.New(apperrors.CodeCampaignInvalidStatusTransition, "campaign status transition is not allowed")
)

// Campaign represents metadata for a campaign.
// Note: GmFear is managed in the snapshot package, not here.
type Campaign struct {
	ID   string
	Name string
	// System is the game system this campaign uses (required, immutable).
	System           commonv1.GameSystem
	Status           CampaignStatus
	GmMode           GmMode
	ParticipantCount int
	CharacterCount   int
	// ThemePrompt provides optional free-form campaign notes.
	ThemePrompt string
	// CreatedAt is the timestamp when the campaign was created.
	CreatedAt time.Time
	// LastActivityAt is the timestamp of the most recent campaign activity.
	LastActivityAt time.Time
	// UpdatedAt is the timestamp when campaign metadata last changed.
	UpdatedAt time.Time
	// CompletedAt is the timestamp when the campaign was completed.
	CompletedAt *time.Time
	// ArchivedAt is the timestamp when the campaign was archived.
	ArchivedAt *time.Time
}

// CreateCampaignInput describes the metadata needed to create a campaign.
type CreateCampaignInput struct {
	Name        string
	System      commonv1.GameSystem
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
		System:           normalized.System,
		Status:           CampaignStatusDraft,
		GmMode:           normalized.GmMode,
		ParticipantCount: 0,
		CharacterCount:   0,
		ThemePrompt:      normalized.ThemePrompt,
		CreatedAt:        createdAt,
		LastActivityAt:   createdAt,
		UpdatedAt:        createdAt,
	}, nil
}

// TransitionCampaignStatus applies a status transition and updates timestamps.
func TransitionCampaignStatus(campaign Campaign, target CampaignStatus, now func() time.Time) (Campaign, error) {
	if now == nil {
		now = time.Now
	}
	if !isCampaignStatusTransitionAllowed(campaign.Status, target) {
		fromStatus := campaignStatusLabel(campaign.Status)
		toStatus := campaignStatusLabel(target)
		return Campaign{}, apperrors.WithMetadata(
			apperrors.CodeCampaignInvalidStatusTransition,
			fmt.Sprintf("campaign status transition not allowed: %s -> %s", fromStatus, toStatus),
			map[string]string{"FromStatus": fromStatus, "ToStatus": toStatus},
		)
	}

	updated := campaign
	updated.Status = target
	updatedAt := now().UTC()
	updated.UpdatedAt = updatedAt
	if target == CampaignStatusCompleted && updated.CompletedAt == nil {
		updated.CompletedAt = &updatedAt
	}
	if target == CampaignStatusArchived && updated.ArchivedAt == nil {
		updated.ArchivedAt = &updatedAt
	}
	if campaign.Status == CampaignStatusArchived && target == CampaignStatusDraft {
		updated.ArchivedAt = nil
		updated.CompletedAt = nil
	}
	return updated, nil
}

// isCampaignStatusTransitionAllowed reports whether a status transition is permitted.
func isCampaignStatusTransitionAllowed(from, to CampaignStatus) bool {
	switch from {
	case CampaignStatusDraft:
		return to == CampaignStatusActive
	case CampaignStatusActive:
		return to == CampaignStatusCompleted || to == CampaignStatusArchived
	case CampaignStatusCompleted:
		return to == CampaignStatusArchived
	case CampaignStatusArchived:
		return to == CampaignStatusDraft
	default:
		return false
	}
}

// campaignStatusLabel returns a stable label for a campaign status.
func campaignStatusLabel(status CampaignStatus) string {
	switch status {
	case CampaignStatusDraft:
		return "DRAFT"
	case CampaignStatusActive:
		return "ACTIVE"
	case CampaignStatusCompleted:
		return "COMPLETED"
	case CampaignStatusArchived:
		return "ARCHIVED"
	default:
		return "UNSPECIFIED"
	}
}

// NormalizeCreateCampaignInput trims and validates campaign input metadata.
func NormalizeCreateCampaignInput(input CreateCampaignInput) (CreateCampaignInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return CreateCampaignInput{}, ErrEmptyName
	}
	if input.System == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
		return CreateCampaignInput{}, ErrInvalidGameSystem
	}
	if input.GmMode == GmModeUnspecified {
		return CreateCampaignInput{}, ErrInvalidGmMode
	}
	return input, nil
}
