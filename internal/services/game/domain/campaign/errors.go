package campaign

import apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"

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
