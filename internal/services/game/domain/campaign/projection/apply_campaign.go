package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (a Applier) applyCampaignCreated(ctx context.Context, evt event.Event) error {
	if a.Campaign == nil {
		return fmt.Errorf("campaign store is not configured")
	}
	if strings.TrimSpace(evt.EntityID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	var payload event.CampaignCreatedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode campaign.created payload: %w", err)
	}

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

	input := campaign.CreateCampaignInput{
		Name:         payload.Name,
		Locale:       locale,
		System:       system,
		GmMode:       gmMode,
		Intent:       intent,
		AccessPolicy: accessPolicy,
		ThemePrompt:  payload.ThemePrompt,
	}
	normalized, err := campaign.NormalizeCreateCampaignInput(input)
	if err != nil {
		return err
	}

	createdAt := ensureTimestamp(evt.Timestamp)
	c := campaign.Campaign{
		ID:               evt.EntityID,
		Name:             normalized.Name,
		Locale:           normalized.Locale,
		System:           normalized.System,
		Status:           campaign.CampaignStatusDraft,
		GmMode:           normalized.GmMode,
		Intent:           normalized.Intent,
		AccessPolicy:     normalized.AccessPolicy,
		ParticipantCount: 0,
		CharacterCount:   0,
		ThemePrompt:      normalized.ThemePrompt,
		CreatedAt:        createdAt,
		UpdatedAt:        createdAt,
	}

	return a.Campaign.Put(ctx, c)
}

func (a Applier) applyCampaignForked(ctx context.Context, evt event.Event) error {
	if a.CampaignFork == nil {
		return fmt.Errorf("campaign fork store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	var payload event.CampaignForkedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode campaign.forked payload: %w", err)
	}
	return a.CampaignFork.SetCampaignForkMetadata(ctx, evt.CampaignID, storage.ForkMetadata{
		ParentCampaignID: payload.ParentCampaignID,
		ForkEventSeq:     payload.ForkEventSeq,
		OriginCampaignID: payload.OriginCampaignID,
	})
}

func (a Applier) applyCampaignUpdated(ctx context.Context, evt event.Event) error {
	if a.Campaign == nil {
		return fmt.Errorf("campaign store is not configured")
	}
	if strings.TrimSpace(evt.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	var payload event.CampaignUpdatedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode campaign.updated payload: %w", err)
	}
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
			statusLabel, ok := value.(string)
			if !ok {
				return fmt.Errorf("campaign.updated status must be string")
			}
			status, err := parseCampaignStatus(statusLabel)
			if err != nil {
				return err
			}
			updated, err = campaign.TransitionCampaignStatus(updated, status, func() time.Time {
				return ensureTimestamp(evt.Timestamp)
			})
			if err != nil {
				return err
			}
		case "name":
			name, ok := value.(string)
			if !ok {
				return fmt.Errorf("campaign.updated name must be string")
			}
			name = strings.TrimSpace(name)
			if name == "" {
				return fmt.Errorf("campaign name is required")
			}
			updated.Name = name
		case "theme_prompt":
			prompt, ok := value.(string)
			if !ok {
				return fmt.Errorf("campaign.updated theme_prompt must be string")
			}
			updated.ThemePrompt = strings.TrimSpace(prompt)
		}
	}

	updatedAt := ensureTimestamp(evt.Timestamp)
	updated.UpdatedAt = updatedAt

	return a.Campaign.Put(ctx, updated)
}
