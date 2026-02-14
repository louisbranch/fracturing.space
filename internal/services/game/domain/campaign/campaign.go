package campaign

import (
	"fmt"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
)

// GmMode describes how the GM role is handled for a campaign.
type GmMode int

// CampaignStatus describes the lifecycle of a campaign.
type CampaignStatus int

// CampaignIntent describes the purpose of a campaign.
type CampaignIntent int

// CampaignAccessPolicy describes who can discover a campaign.
type CampaignAccessPolicy int

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

const (
	// CampaignIntentUnspecified represents an invalid campaign intent value.
	CampaignIntentUnspecified CampaignIntent = iota
	// CampaignIntentStandard indicates a regular campaign.
	CampaignIntentStandard
	// CampaignIntentStarter indicates a beginner-friendly starter campaign.
	CampaignIntentStarter
	// CampaignIntentSandbox indicates an ephemeral campaign for testing.
	CampaignIntentSandbox
)

const (
	// CampaignAccessPolicyUnspecified represents an invalid access policy value.
	CampaignAccessPolicyUnspecified CampaignAccessPolicy = iota
	// CampaignAccessPolicyPrivate limits access to participants.
	CampaignAccessPolicyPrivate
	// CampaignAccessPolicyRestricted limits access to an allowlist.
	CampaignAccessPolicyRestricted
	// CampaignAccessPolicyPublic allows public discovery.
	CampaignAccessPolicyPublic
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
	// Locale is the preferred locale for the campaign.
	Locale commonv1.Locale
	// System is the game system this campaign uses (required, immutable).
	System           commonv1.GameSystem
	Status           CampaignStatus
	GmMode           GmMode
	Intent           CampaignIntent
	AccessPolicy     CampaignAccessPolicy
	ParticipantCount int
	CharacterCount   int
	// ThemePrompt provides optional free-form campaign notes.
	ThemePrompt string
	// CreatedAt is the timestamp when the campaign was created.
	CreatedAt time.Time
	// UpdatedAt is the timestamp when campaign metadata last changed.
	UpdatedAt time.Time
	// CompletedAt is the timestamp when the campaign was completed.
	CompletedAt *time.Time
	// ArchivedAt is the timestamp when the campaign was archived.
	ArchivedAt *time.Time
}

// CreateCampaignInput describes the metadata needed to create a campaign.
type CreateCampaignInput struct {
	Name         string
	Locale       commonv1.Locale
	System       commonv1.GameSystem
	GmMode       GmMode
	Intent       CampaignIntent
	AccessPolicy CampaignAccessPolicy
	ThemePrompt  string
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
		Locale:           normalized.Locale,
		System:           normalized.System,
		Status:           CampaignStatusDraft,
		GmMode:           normalized.GmMode,
		Intent:           normalized.Intent,
		AccessPolicy:     normalized.AccessPolicy,
		ParticipantCount: 0,
		CharacterCount:   0,
		ThemePrompt:      normalized.ThemePrompt,
		CreatedAt:        createdAt,
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

// CampaignStatusFromLabel parses a string label into a CampaignStatus.
// It trims whitespace and matches case-insensitively. Both short ("DRAFT")
// and prefixed ("CAMPAIGN_STATUS_DRAFT") forms are accepted.
func CampaignStatusFromLabel(value string) (CampaignStatus, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return CampaignStatusUnspecified, fmt.Errorf("campaign status is required")
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "DRAFT", "CAMPAIGN_STATUS_DRAFT":
		return CampaignStatusDraft, nil
	case "ACTIVE", "CAMPAIGN_STATUS_ACTIVE":
		return CampaignStatusActive, nil
	case "COMPLETED", "CAMPAIGN_STATUS_COMPLETED":
		return CampaignStatusCompleted, nil
	case "ARCHIVED", "CAMPAIGN_STATUS_ARCHIVED":
		return CampaignStatusArchived, nil
	default:
		return CampaignStatusUnspecified, fmt.Errorf("unknown campaign status: %s", trimmed)
	}
}

// GmModeFromLabel parses a string label into a GmMode.
// It trims whitespace and matches case-insensitively. Both short ("HUMAN")
// and prefixed ("GM_MODE_HUMAN") forms are accepted.
func GmModeFromLabel(value string) (GmMode, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return GmModeUnspecified, fmt.Errorf("gm mode is required")
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "HUMAN", "GM_MODE_HUMAN":
		return GmModeHuman, nil
	case "AI", "GM_MODE_AI":
		return GmModeAI, nil
	case "HYBRID", "GM_MODE_HYBRID":
		return GmModeHybrid, nil
	default:
		return GmModeUnspecified, fmt.Errorf("unknown gm mode: %s", trimmed)
	}
}

// CampaignIntentFromLabel parses a string label into a CampaignIntent.
// It trims whitespace and matches case-insensitively. Returns
// CampaignIntentStandard for empty or unrecognized values.
func CampaignIntentFromLabel(value string) CampaignIntent {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return CampaignIntentStandard
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "STANDARD", "CAMPAIGN_INTENT_STANDARD":
		return CampaignIntentStandard
	case "STARTER", "CAMPAIGN_INTENT_STARTER":
		return CampaignIntentStarter
	case "SANDBOX", "CAMPAIGN_INTENT_SANDBOX":
		return CampaignIntentSandbox
	default:
		return CampaignIntentStandard
	}
}

// CampaignAccessPolicyFromLabel parses a string label into a CampaignAccessPolicy.
// It trims whitespace and matches case-insensitively. Returns
// CampaignAccessPolicyPrivate for empty or unrecognized values.
func CampaignAccessPolicyFromLabel(value string) CampaignAccessPolicy {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return CampaignAccessPolicyPrivate
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "PRIVATE", "CAMPAIGN_ACCESS_POLICY_PRIVATE":
		return CampaignAccessPolicyPrivate
	case "RESTRICTED", "CAMPAIGN_ACCESS_POLICY_RESTRICTED":
		return CampaignAccessPolicyRestricted
	case "PUBLIC", "CAMPAIGN_ACCESS_POLICY_PUBLIC":
		return CampaignAccessPolicyPublic
	default:
		return CampaignAccessPolicyPrivate
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
	input.Locale = platformi18n.NormalizeLocale(input.Locale)
	if input.Intent == CampaignIntentUnspecified {
		input.Intent = CampaignIntentStandard
	}
	if input.AccessPolicy == CampaignAccessPolicyUnspecified {
		input.AccessPolicy = CampaignAccessPolicyPrivate
	}
	return input, nil
}
