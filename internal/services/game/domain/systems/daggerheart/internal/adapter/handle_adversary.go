package adapter

import (
	"context"
	"strings"

	event "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func (a *Adapter) HandleAdversaryCreated(ctx context.Context, evt event.Event, p payload.AdversaryCreatedPayload) error {
	if err := projection.ValidateAdversaryStats(p.HP, p.HPMax, p.Stress, p.StressMax, p.Evasion, p.Major, p.Severe, p.Armor); err != nil {
		return err
	}
	createdAt := evt.Timestamp.UTC()
	return a.store.PutDaggerheartAdversary(ctx, projectionstore.DaggerheartAdversary{
		CampaignID:        string(evt.CampaignID),
		AdversaryID:       strings.TrimSpace(p.AdversaryID.String()),
		AdversaryEntryID:  strings.TrimSpace(p.AdversaryEntryID),
		Name:              strings.TrimSpace(p.Name),
		Kind:              strings.TrimSpace(p.Kind),
		SessionID:         strings.TrimSpace(p.SessionID.String()),
		SceneID:           strings.TrimSpace(p.SceneID.String()),
		Notes:             strings.TrimSpace(p.Notes),
		HP:                p.HP,
		HPMax:             p.HPMax,
		Stress:            p.Stress,
		StressMax:         p.StressMax,
		Evasion:           p.Evasion,
		Major:             p.Major,
		Severe:            p.Severe,
		Armor:             p.Armor,
		FeatureStates:     ToProjectionAdversaryFeatureStates(p.FeatureStates),
		PendingExperience: ToProjectionAdversaryPendingExperience(p.PendingExperience),
		SpotlightGateID:   strings.TrimSpace(p.SpotlightGateID.String()),
		SpotlightCount:    p.SpotlightCount,
		CreatedAt:         createdAt,
		UpdatedAt:         createdAt,
	})
}

func (a *Adapter) HandleAdversaryUpdated(ctx context.Context, evt event.Event, p payload.AdversaryUpdatedPayload) error {
	adversaryID := strings.TrimSpace(p.AdversaryID.String())
	if err := projection.ValidateAdversaryStats(p.HP, p.HPMax, p.Stress, p.StressMax, p.Evasion, p.Major, p.Severe, p.Armor); err != nil {
		return err
	}
	current, err := a.store.GetDaggerheartAdversary(ctx, string(evt.CampaignID), adversaryID)
	if err != nil {
		return err
	}
	updatedAt := evt.Timestamp.UTC()
	return a.store.PutDaggerheartAdversary(ctx, projectionstore.DaggerheartAdversary{
		CampaignID:        string(evt.CampaignID),
		AdversaryID:       adversaryID,
		AdversaryEntryID:  strings.TrimSpace(p.AdversaryEntryID),
		Name:              strings.TrimSpace(p.Name),
		Kind:              strings.TrimSpace(p.Kind),
		SessionID:         strings.TrimSpace(p.SessionID.String()),
		SceneID:           strings.TrimSpace(p.SceneID.String()),
		Notes:             strings.TrimSpace(p.Notes),
		HP:                p.HP,
		HPMax:             p.HPMax,
		Stress:            p.Stress,
		StressMax:         p.StressMax,
		Evasion:           p.Evasion,
		Major:             p.Major,
		Severe:            p.Severe,
		Armor:             p.Armor,
		Conditions:        current.Conditions,
		FeatureStates:     ToProjectionAdversaryFeatureStates(p.FeatureStates),
		PendingExperience: ToProjectionAdversaryPendingExperience(p.PendingExperience),
		SpotlightGateID:   strings.TrimSpace(p.SpotlightGateID.String()),
		SpotlightCount:    p.SpotlightCount,
		CreatedAt:         current.CreatedAt,
		UpdatedAt:         updatedAt,
	})
}

func ToProjectionAdversaryFeatureStates(in []rules.AdversaryFeatureState) []projectionstore.DaggerheartAdversaryFeatureState {
	out := make([]projectionstore.DaggerheartAdversaryFeatureState, 0, len(in))
	for _, featureState := range in {
		out = append(out, projectionstore.DaggerheartAdversaryFeatureState{
			FeatureID:       strings.TrimSpace(featureState.FeatureID),
			Status:          strings.TrimSpace(featureState.Status),
			FocusedTargetID: strings.TrimSpace(featureState.FocusedTargetID),
		})
	}
	return out
}

func ToProjectionAdversaryPendingExperience(in *rules.AdversaryPendingExperience) *projectionstore.DaggerheartAdversaryPendingExperience {
	if in == nil {
		return nil
	}
	return &projectionstore.DaggerheartAdversaryPendingExperience{
		Name:     strings.TrimSpace(in.Name),
		Modifier: in.Modifier,
	}
}

func (a *Adapter) HandleAdversaryDamageApplied(ctx context.Context, evt event.Event, p payload.AdversaryDamageAppliedPayload) error {
	adversaryID := strings.TrimSpace(p.AdversaryID.String())
	current, err := a.store.GetDaggerheartAdversary(ctx, string(evt.CampaignID), adversaryID)
	if err != nil {
		return err
	}
	next, err := projection.ApplyAdversaryDamagePatch(current, p.Hp, p.Armor)
	if err != nil {
		return err
	}
	next.CampaignID = string(evt.CampaignID)
	next.AdversaryID = adversaryID
	next.UpdatedAt = evt.Timestamp.UTC()
	return a.store.PutDaggerheartAdversary(ctx, next)
}

func (a *Adapter) HandleAdversaryDeleted(ctx context.Context, evt event.Event, p payload.AdversaryDeletedPayload) error {
	return a.store.DeleteDaggerheartAdversary(ctx, string(evt.CampaignID), strings.TrimSpace(p.AdversaryID.String()))
}
