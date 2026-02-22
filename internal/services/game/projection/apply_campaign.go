package projection

import (
	"context"
	"fmt"
	"strings"

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// ProjectionHandledTypes returns the core event types handled by the projection
// layer. The list is derived from the handler registry map so there is a single
// source of truth for which event types have projection handlers.
func ProjectionHandledTypes() []event.Type {
	return registeredHandlerTypes()
}

func (a Applier) applyCampaignCreated(ctx context.Context, evt event.Event, payload campaign.CreatePayload) error {
	system, err := parseGameSystem(payload.GameSystem)
	if err != nil {
		return err
	}
	gmMode, err := parseGmMode(payload.GmMode)
	if err != nil {
		return err
	}
	intent := parseCampaignIntent(payload.Intent)
	accessPolicy := parseCampaignAccessPolicy(payload.AccessPolicy)
	locale := platformi18n.DefaultLocale()
	if parsed, ok := platformi18n.ParseLocale(payload.Locale); ok {
		locale = parsed
	}

	input := campaign.CreateInput{
		Name:         payload.Name,
		Locale:       locale,
		System:       system,
		GmMode:       gmMode,
		Intent:       intent,
		AccessPolicy: accessPolicy,
		ThemePrompt:  payload.ThemePrompt,
	}
	normalized, err := campaign.NormalizeCreateInput(input)
	if err != nil {
		return err
	}

	createdAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	return a.Campaign.Put(ctx, storage.CampaignRecord{
		ID:               evt.EntityID,
		Name:             normalized.Name,
		Locale:           normalized.Locale,
		System:           normalized.System,
		Status:           campaign.StatusDraft,
		GmMode:           normalized.GmMode,
		Intent:           normalized.Intent,
		AccessPolicy:     normalized.AccessPolicy,
		ParticipantCount: 0,
		CharacterCount:   0,
		ThemePrompt:      normalized.ThemePrompt,
		CoverAssetID:     strings.TrimSpace(payload.CoverAssetID),
		CoverSetID:       strings.TrimSpace(payload.CoverSetID),
		CreatedAt:        createdAt,
		UpdatedAt:        createdAt,
	})
}

func (a Applier) applyCampaignUpdated(ctx context.Context, evt event.Event, payload campaign.UpdatePayload) error {
	if len(payload.Fields) == 0 {
		return nil
	}

	current, err := a.Campaign.Get(ctx, evt.CampaignID)
	if err != nil {
		return err
	}

	updated := current
	for key, value := range payload.Fields {
		switch key {
		case "status":
			status, err := parseCampaignStatus(value)
			if err != nil {
				return err
			}
			statusTS, tsErr := ensureTimestamp(evt.Timestamp)
			if tsErr != nil {
				return tsErr
			}
			updated, err = applyCampaignStatusTransition(updated, status, statusTS)
			if err != nil {
				return err
			}
		case "name":
			name := strings.TrimSpace(value)
			if name == "" {
				return fmt.Errorf("campaign name is required")
			}
			updated.Name = name
		case "theme_prompt":
			updated.ThemePrompt = strings.TrimSpace(value)
		case "cover_asset_id":
			updated.CoverAssetID = strings.TrimSpace(value)
		case "cover_set_id":
			updated.CoverSetID = strings.TrimSpace(value)
		}
	}

	updatedAt, err := ensureTimestamp(evt.Timestamp)
	if err != nil {
		return err
	}
	updated.UpdatedAt = updatedAt

	return a.Campaign.Put(ctx, updated)
}

func (a Applier) applyCampaignForked(ctx context.Context, evt event.Event, payload campaign.ForkPayload) error {
	return a.CampaignFork.SetCampaignForkMetadata(ctx, evt.CampaignID, storage.ForkMetadata{
		ParentCampaignID: payload.ParentCampaignID,
		ForkEventSeq:     payload.ForkEventSeq,
		OriginCampaignID: payload.OriginCampaignID,
	})
}
