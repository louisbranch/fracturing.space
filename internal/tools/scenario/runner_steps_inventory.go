package scenario

import (
	"context"
	"fmt"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
)

func (r *Runner) runLevelUpStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureCampaign(state); err != nil {
		return err
	}
	name := requiredString(step.Args, "target")
	if name == "" {
		return r.failf("level_up target is required")
	}
	levelAfter := optionalInt(step.Args, "level_after", 0)
	if levelAfter <= 0 {
		return r.failf("level_up level_after is required")
	}

	characterID, err := actorID(state, name)
	if err != nil {
		return err
	}

	// Build advancement entries from the "advancements" list.
	var advancements []*daggerheartv1.DaggerheartLevelUpAdvancement
	rawAdvancements, ok := step.Args["advancements"].([]any)
	if !ok || len(rawAdvancements) == 0 {
		return r.failf("level_up advancements list is required")
	}
	for _, rawAdv := range rawAdvancements {
		advMap, ok := rawAdv.(map[string]any)
		if !ok {
			return r.failf("level_up advancement entry must be a table")
		}
		entry := &daggerheartv1.DaggerheartLevelUpAdvancement{
			Type:            optionalString(advMap, "type", ""),
			Trait:           optionalString(advMap, "trait", ""),
			DomainCardId:    optionalString(advMap, "domain_card_id", ""),
			DomainCardLevel: int32(optionalInt(advMap, "domain_card_level", 0)),
			SubclassCardId:  optionalString(advMap, "subclass_card_id", ""),
		}
		rawMulticlass, _ := advMap["multiclass"].(map[string]any)
		if rawMulticlass != nil {
			entry.Multiclass = &daggerheartv1.DaggerheartLevelUpMulticlass{
				SecondaryClassId:    optionalString(rawMulticlass, "secondary_class_id", ""),
				SecondarySubclassId: optionalString(rawMulticlass, "secondary_subclass_id", ""),
				FoundationCardId:    optionalString(rawMulticlass, "foundation_card_id", ""),
				SpellcastTrait:      optionalString(rawMulticlass, "spellcast_trait", ""),
				DomainId:            optionalString(rawMulticlass, "domain_id", ""),
			}
		}
		advancements = append(advancements, entry)
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)
	_, err = r.env.daggerheartClient.ApplyLevelUp(ctxWithSession, &daggerheartv1.DaggerheartApplyLevelUpRequest{
		CampaignId:         state.campaignID,
		CharacterId:        characterID,
		LevelAfter:         int32(levelAfter),
		Advancements:       advancements,
		NewDomainCardId:    optionalString(step.Args, "new_domain_card_id", ""),
		NewDomainCardLevel: int32(optionalInt(step.Args, "new_domain_card_level", 0)),
	})
	if err != nil {
		return fmt.Errorf("level_up: %w", err)
	}
	return r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeLevelUpApplied)
}

func (r *Runner) runUpdateGoldStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureCampaign(state); err != nil {
		return err
	}
	name := requiredString(step.Args, "target")
	if name == "" {
		return r.failf("update_gold target is required")
	}

	characterID, err := actorID(state, name)
	if err != nil {
		return err
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)
	_, err = r.env.daggerheartClient.UpdateGold(ctxWithSession, &daggerheartv1.DaggerheartUpdateGoldRequest{
		CampaignId:     state.campaignID,
		CharacterId:    characterID,
		HandfulsBefore: int32(optionalInt(step.Args, "handfuls_before", 0)),
		HandfulsAfter:  int32(optionalInt(step.Args, "handfuls_after", 0)),
		BagsBefore:     int32(optionalInt(step.Args, "bags_before", 0)),
		BagsAfter:      int32(optionalInt(step.Args, "bags_after", 0)),
		ChestsBefore:   int32(optionalInt(step.Args, "chests_before", 0)),
		ChestsAfter:    int32(optionalInt(step.Args, "chests_after", 0)),
		Reason:         optionalString(step.Args, "reason", ""),
	})
	if err != nil {
		return fmt.Errorf("update_gold: %w", err)
	}
	return r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeGoldUpdated)
}

func (r *Runner) runAcquireDomainCardStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureCampaign(state); err != nil {
		return err
	}
	name := requiredString(step.Args, "target")
	if name == "" {
		return r.failf("acquire_domain_card target is required")
	}
	cardID := requiredString(step.Args, "card_id")
	if cardID == "" {
		return r.failf("acquire_domain_card card_id is required")
	}

	characterID, err := actorID(state, name)
	if err != nil {
		return err
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)
	_, err = r.env.daggerheartClient.AcquireDomainCard(ctxWithSession, &daggerheartv1.DaggerheartAcquireDomainCardRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
		CardId:      cardID,
		CardLevel:   int32(optionalInt(step.Args, "card_level", 1)),
		Destination: optionalString(step.Args, "destination", "vault"),
	})
	if err != nil {
		return fmt.Errorf("acquire_domain_card: %w", err)
	}
	return r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeDomainCardAcquired)
}

func (r *Runner) runSwapEquipmentStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureCampaign(state); err != nil {
		return err
	}
	name := requiredString(step.Args, "target")
	if name == "" {
		return r.failf("swap_equipment target is required")
	}
	itemID := requiredString(step.Args, "item_id")
	if itemID == "" {
		return r.failf("swap_equipment item_id is required")
	}

	characterID, err := actorID(state, name)
	if err != nil {
		return err
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)
	_, err = r.env.daggerheartClient.SwapEquipment(ctxWithSession, &daggerheartv1.DaggerheartSwapEquipmentRequest{
		CampaignId:  state.campaignID,
		CharacterId: characterID,
		ItemId:      itemID,
		ItemType:    optionalString(step.Args, "item_type", "weapon"),
		From:        optionalString(step.Args, "from", "inventory"),
		To:          optionalString(step.Args, "to", "active"),
		StressCost:  int32(optionalInt(step.Args, "stress_cost", 0)),
	})
	if err != nil {
		return fmt.Errorf("swap_equipment: %w", err)
	}
	return r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeEquipmentSwapped)
}

func (r *Runner) runUseConsumableStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureCampaign(state); err != nil {
		return err
	}
	name := requiredString(step.Args, "target")
	if name == "" {
		return r.failf("use_consumable target is required")
	}
	consumableID := requiredString(step.Args, "consumable_id")
	if consumableID == "" {
		return r.failf("use_consumable consumable_id is required")
	}

	characterID, err := actorID(state, name)
	if err != nil {
		return err
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)
	_, err = r.env.daggerheartClient.UseConsumable(ctxWithSession, &daggerheartv1.DaggerheartUseConsumableRequest{
		CampaignId:     state.campaignID,
		CharacterId:    characterID,
		ConsumableId:   consumableID,
		QuantityBefore: int32(optionalInt(step.Args, "quantity_before", 0)),
		QuantityAfter:  int32(optionalInt(step.Args, "quantity_after", 0)),
	})
	if err != nil {
		return fmt.Errorf("use_consumable: %w", err)
	}
	return r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeConsumableUsed)
}

func (r *Runner) runAcquireConsumableStep(ctx context.Context, state *scenarioState, step Step) error {
	if err := r.ensureCampaign(state); err != nil {
		return err
	}
	name := requiredString(step.Args, "target")
	if name == "" {
		return r.failf("acquire_consumable target is required")
	}
	consumableID := requiredString(step.Args, "consumable_id")
	if consumableID == "" {
		return r.failf("acquire_consumable consumable_id is required")
	}

	characterID, err := actorID(state, name)
	if err != nil {
		return err
	}

	before, err := r.latestSeq(ctx, state)
	if err != nil {
		return err
	}
	ctxWithSession := withSessionID(ctx, state.sessionID)
	_, err = r.env.daggerheartClient.AcquireConsumable(ctxWithSession, &daggerheartv1.DaggerheartAcquireConsumableRequest{
		CampaignId:     state.campaignID,
		CharacterId:    characterID,
		ConsumableId:   consumableID,
		QuantityBefore: int32(optionalInt(step.Args, "quantity_before", 0)),
		QuantityAfter:  int32(optionalInt(step.Args, "quantity_after", 0)),
	})
	if err != nil {
		return fmt.Errorf("acquire_consumable: %w", err)
	}
	return r.requireDaggerheartEventTypesAfterSeq(ctx, state, before, daggerheart.EventTypeConsumableAcquired)
}
