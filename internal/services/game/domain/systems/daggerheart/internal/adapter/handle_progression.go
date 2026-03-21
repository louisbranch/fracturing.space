package adapter

import (
	"context"
	"errors"
	"fmt"
	"strings"

	event "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (a *Adapter) HandleRestTaken(ctx context.Context, evt event.Event, p payload.RestTakenPayload) error {
	if err := a.PutSnapshot(ctx, string(evt.CampaignID), p.GMFear, p.ShortRests); err != nil {
		return err
	}
	for _, participantID := range p.Participants {
		characterID := strings.TrimSpace(participantID.String())
		if p.RefreshRest || p.RefreshLongRest {
			if err := a.ClearRestTemporaryArmor(ctx, string(evt.CampaignID), characterID, p.RefreshRest, p.RefreshLongRest); err != nil {
				return err
			}
		}
		if err := a.ClearRestStatModifiers(ctx, string(evt.CampaignID), characterID, p.RefreshRest, p.RefreshLongRest); err != nil {
			return err
		}
	}
	return nil
}

func (a *Adapter) HandleLevelUpApplied(ctx context.Context, evt event.Event, p payload.LevelUpAppliedPayload) error {
	characterID := strings.TrimSpace(p.CharacterID.String())
	storedProfile, err := a.store.GetDaggerheartCharacterProfile(ctx, string(evt.CampaignID), characterID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return fmt.Errorf("get daggerheart character profile for level-up: %w", err)
		}
		return nil
	}
	profile := daggerheartstate.CharacterProfileFromStorage(storedProfile)
	a.applyLevelUp(&profile, p)
	return a.store.PutDaggerheartCharacterProfile(ctx, profile.ToStorage(string(evt.CampaignID), characterID))
}

func (a *Adapter) HandleGoldUpdated(ctx context.Context, evt event.Event, p payload.GoldUpdatedPayload) error {
	characterID := strings.TrimSpace(p.CharacterID.String())
	profile, err := a.store.GetDaggerheartCharacterProfile(ctx, string(evt.CampaignID), characterID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return fmt.Errorf("get daggerheart character profile for gold update: %w", err)
		}
		return nil
	}
	profile.GoldHandfuls = p.Handfuls
	profile.GoldBags = p.Bags
	profile.GoldChests = p.Chests
	return a.store.PutDaggerheartCharacterProfile(ctx, profile)
}

func (a *Adapter) HandleDomainCardAcquired(ctx context.Context, evt event.Event, p payload.DomainCardAcquiredPayload) error {
	characterID := strings.TrimSpace(p.CharacterID.String())
	profile, err := a.store.GetDaggerheartCharacterProfile(ctx, string(evt.CampaignID), characterID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return fmt.Errorf("get daggerheart character profile for domain card acquire: %w", err)
		}
		return nil
	}
	profile.DomainCardIDs = daggerheartstate.AppendUnique(profile.DomainCardIDs, strings.TrimSpace(p.CardID))
	return a.store.PutDaggerheartCharacterProfile(ctx, profile)
}

func (a *Adapter) HandleEquipmentSwapped(ctx context.Context, evt event.Event, p payload.EquipmentSwappedPayload) error {
	characterID := strings.TrimSpace(p.CharacterID.String())
	if characterID == "" {
		return nil
	}
	if strings.TrimSpace(p.ItemType) == "armor" {
		profile, err := a.store.GetDaggerheartCharacterProfile(ctx, string(evt.CampaignID), characterID)
		if err != nil {
			if !errors.Is(err, storage.ErrNotFound) {
				return fmt.Errorf("get daggerheart character profile for equipment swap: %w", err)
			}
		} else {
			profile.EquippedArmorID = strings.TrimSpace(p.EquippedArmorID)
			if p.EvasionAfter != nil {
				profile.Evasion = *p.EvasionAfter
			}
			if p.MajorThresholdAfter != nil {
				profile.MajorThreshold = *p.MajorThresholdAfter
			}
			if p.SevereThresholdAfter != nil {
				profile.SevereThreshold = *p.SevereThresholdAfter
			}
			if p.ArmorScoreAfter != nil {
				profile.ArmorScore = *p.ArmorScoreAfter
			}
			if p.ArmorMaxAfter != nil {
				profile.ArmorMax = *p.ArmorMaxAfter
			}
			if p.SpellcastRollBonusAfter != nil {
				profile.SpellcastRollBonus = *p.SpellcastRollBonusAfter
			}
			if p.AgilityAfter != nil {
				profile.Agility = *p.AgilityAfter
			}
			if p.StrengthAfter != nil {
				profile.Strength = *p.StrengthAfter
			}
			if p.FinesseAfter != nil {
				profile.Finesse = *p.FinesseAfter
			}
			if p.InstinctAfter != nil {
				profile.Instinct = *p.InstinctAfter
			}
			if p.PresenceAfter != nil {
				profile.Presence = *p.PresenceAfter
			}
			if p.KnowledgeAfter != nil {
				profile.Knowledge = *p.KnowledgeAfter
			}
			if err := a.store.PutDaggerheartCharacterProfile(ctx, profile); err != nil {
				return fmt.Errorf("put daggerheart character profile for equipment swap: %w", err)
			}
		}
		if p.ArmorAfter != nil {
			if err := a.ApplyStatePatch(ctx, string(evt.CampaignID), characterID, StatePatch{Armor: p.ArmorAfter}); err != nil {
				return err
			}
		}
	}
	if p.StressCost > 0 {
		state, err := a.GetCharacterStateOrDefault(ctx, string(evt.CampaignID), characterID)
		if err != nil {
			return err
		}
		stressAfter := state.Stress + p.StressCost
		if err := a.ApplyStatePatch(ctx, string(evt.CampaignID), characterID, StatePatch{Stress: &stressAfter}); err != nil {
			return err
		}
	}
	return nil
}

func (a *Adapter) HandleConsumableUsed(_ context.Context, _ event.Event, _ payload.ConsumableUsedPayload) error {
	return nil
}

func (a *Adapter) HandleConsumableAcquired(_ context.Context, _ event.Event, _ payload.ConsumableAcquiredPayload) error {
	return nil
}

func (a *Adapter) HandleStatModifierChanged(ctx context.Context, evt event.Event, p payload.StatModifierChangedPayload) error {
	characterID := strings.TrimSpace(p.CharacterID.String())
	state, err := a.GetCharacterStateOrDefault(ctx, string(evt.CampaignID), characterID)
	if err != nil {
		return err
	}
	state.StatModifiers = StatModifiersToProjection(p.Modifiers)
	return a.PutCharacterState(ctx, state)
}
