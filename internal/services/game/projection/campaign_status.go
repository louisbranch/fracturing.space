package projection

import (
	"fmt"
	"time"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// applyCampaignStatusTransition enforces campaign lifecycle rules while
// preserving timestamps for history-sensitive read models (completed/archived at).
func applyCampaignStatusTransition(record storage.CampaignRecord, target campaign.Status, now time.Time) (storage.CampaignRecord, error) {
	if !campaign.IsStatusTransitionAllowed(record.Status, target) {
		fromStatus := campaignStatusLabel(record.Status)
		toStatus := campaignStatusLabel(target)
		return storage.CampaignRecord{}, apperrors.WithMetadata(
			apperrors.CodeCampaignInvalidStatusTransition,
			fmt.Sprintf("campaign status transition not allowed: %s -> %s", fromStatus, toStatus),
			map[string]string{"FromStatus": fromStatus, "ToStatus": toStatus},
		)
	}

	updated := record
	updated.Status = target
	updatedAt := now.UTC()
	updated.UpdatedAt = updatedAt
	if target == campaign.StatusCompleted && updated.CompletedAt == nil {
		updated.CompletedAt = &updatedAt
	}
	if target == campaign.StatusArchived && updated.ArchivedAt == nil {
		updated.ArchivedAt = &updatedAt
	}
	if record.Status == campaign.StatusArchived && target == campaign.StatusDraft {
		updated.ArchivedAt = nil
		updated.CompletedAt = nil
	}
	return updated, nil
}

// campaignStatusLabel centralizes stable status labels for error/reporting context.
func campaignStatusLabel(status campaign.Status) string {
	switch status {
	case campaign.StatusDraft:
		return "DRAFT"
	case campaign.StatusActive:
		return "ACTIVE"
	case campaign.StatusCompleted:
		return "COMPLETED"
	case campaign.StatusArchived:
		return "ARCHIVED"
	default:
		return "UNSPECIFIED"
	}
}
