package decider

import (
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func TestDeciderCoverageBranches(t *testing.T) {
	t.Parallel()

	now := func() time.Time { return time.Date(2026, 3, 27, 14, 0, 0, 0, time.UTC) }

	t.Run("gm fear set rejects nil, out of range, and unchanged values", func(t *testing.T) {
		t.Parallel()

		decider := NewDecider([]command.Type{commandTypeGMFearSet})
		snapshot := daggerheartstate.NewSnapshotState("camp-1")
		snapshot.GMFear = 2

		singleRejection(t, decider.Decide(snapshot, command.Command{
			CampaignID:  ids.CampaignID("camp-1"),
			Type:        commandTypeGMFearSet,
			PayloadJSON: mustMarshalJSON(t, payload.GMFearSetPayload{}),
		}, now), rejectionCodeGMFearAfterRequired)

		tooHigh := daggerheartstate.GMFearMax + 1
		singleRejection(t, decider.Decide(snapshot, command.Command{
			CampaignID:  ids.CampaignID("camp-1"),
			Type:        commandTypeGMFearSet,
			PayloadJSON: mustMarshalJSON(t, payload.GMFearSetPayload{After: &tooHigh}),
		}, now), rejectionCodeGMFearOutOfRange)

		unchanged := 2
		singleRejection(t, decider.Decide(snapshot, command.Command{
			CampaignID:  ids.CampaignID("camp-1"),
			Type:        commandTypeGMFearSet,
			PayloadJSON: mustMarshalJSON(t, payload.GMFearSetPayload{After: &unchanged}),
		}, now), rejectionCodeGMFearUnchanged)

		after := 3
		got := singleEventPayload[payload.GMFearChangedPayload](t, decider.Decide(snapshot, command.Command{
			CampaignID:  ids.CampaignID("camp-1"),
			Type:        commandTypeGMFearSet,
			PayloadJSON: mustMarshalJSON(t, payload.GMFearSetPayload{After: &after, Reason: " advance "}),
		}, now))
		if got.Value != 3 || got.Reason != "advance" {
			t.Fatalf("GMFearChangedPayload = %#v", got)
		}
	})

	t.Run("gm move apply covers normalization and rejection branches", func(t *testing.T) {
		t.Parallel()

		decider := NewDecider([]command.Type{CommandTypeGMMoveApply})
		snapshot := daggerheartstate.NewSnapshotState("camp-1")
		snapshot.GMFear = 3

		singleRejection(t, decider.Decide(snapshot, command.Command{
			CampaignID:  ids.CampaignID("camp-1"),
			SessionID:   ids.SessionID("sess-1"),
			Type:        CommandTypeGMMoveApply,
			PayloadJSON: mustMarshalJSON(t, payload.GMMoveApplyPayload{FearSpent: 0}),
		}, now), rejectionCodeGMMoveKindUnsupported)

		singleRejection(t, decider.Decide(snapshot, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			SessionID:  ids.SessionID("sess-1"),
			Type:       CommandTypeGMMoveApply,
			PayloadJSON: mustMarshalJSON(t, payload.GMMoveApplyPayload{
				FearSpent: 1,
				Target: payload.GMMoveTarget{
					Type:  rules.GMMoveTargetTypeDirectMove,
					Kind:  rules.GMMoveKindInterruptAndMove,
					Shape: rules.GMMoveShapeCustom,
				},
			}),
		}, now), rejectionCodeGMMoveDescriptionRequired)

		singleRejection(t, decider.Decide(snapshot, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			SessionID:  ids.SessionID("sess-1"),
			Type:       CommandTypeGMMoveApply,
			PayloadJSON: mustMarshalJSON(t, payload.GMMoveApplyPayload{
				FearSpent: 4,
				Target: payload.GMMoveTarget{
					Type:          rules.GMMoveTargetTypeAdversaryFeature,
					AdversaryID:   dhids.AdversaryID(" adv-1 "),
					FeatureID:     " feature-1 ",
					Description:   " press the attack ",
					EnvironmentID: "",
				},
			}),
		}, now), rejectionCodeGMMoveInsufficientFear)

		decision := decider.Decide(snapshot, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			SessionID:  ids.SessionID("sess-1"),
			Type:       CommandTypeGMMoveApply,
			PayloadJSON: mustMarshalJSON(t, payload.GMMoveApplyPayload{
				FearSpent: 1,
				Target: payload.GMMoveTarget{
					Type:        rules.GMMoveTargetTypeAdversaryFeature,
					AdversaryID: dhids.AdversaryID(" adv-1 "),
					FeatureID:   " feature-1 ",
					Description: " press the attack ",
				},
			}),
		}, now)
		if len(decision.Rejections) != 0 {
			t.Fatalf("unexpected rejections: %+v", decision.Rejections)
		}
		if len(decision.Events) != 2 {
			t.Fatalf("events = %d, want 2", len(decision.Events))
		}

		applied := decodeEventPayload[payload.GMMoveAppliedPayload](t, decision.Events[0])
		if applied.Target.Type != rules.GMMoveTargetTypeAdversaryFeature || applied.Target.AdversaryID != dhids.AdversaryID("adv-1") || applied.Target.FeatureID != "feature-1" {
			t.Fatalf("GMMoveAppliedPayload = %#v", applied)
		}
		fear := decodeEventPayload[payload.GMFearChangedPayload](t, decision.Events[1])
		if fear.Value != 2 || fear.Reason != "gm_move" {
			t.Fatalf("GMFearChangedPayload = %#v", fear)
		}
	})

	t.Run("condition change rejects missing removals and normalizes source", func(t *testing.T) {
		t.Parallel()

		snapshot := daggerheartstate.NewSnapshotState("camp-1")
		snapshot.CharacterStates[ids.CharacterID("char-1")] = daggerheartstate.CharacterState{
			Conditions: []string{rules.ConditionHidden},
		}
		hidden := mustConditionState(t, rules.ConditionHidden)
		restrained := mustConditionState(t, rules.ConditionRestrained)
		rollSeq := uint64(9)
		decider := NewDecider([]command.Type{CommandTypeConditionChange})

		singleRejection(t, decider.Decide(snapshot, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeConditionChange,
			PayloadJSON: mustMarshalJSON(t, payload.ConditionChangePayload{
				CharacterID: ids.CharacterID("char-1"),
				Removed:     []rules.ConditionState{restrained},
			}),
		}, now), rejectionCodeConditionChangeRemoveMissing)

		got := singleEventPayload[payload.ConditionChangedPayload](t, decider.Decide(snapshot, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeConditionChange,
			PayloadJSON: mustMarshalJSON(t, payload.ConditionChangePayload{
				CharacterID:     ids.CharacterID(" char-1 "),
				ConditionsAfter: []rules.ConditionState{hidden, restrained},
				Added:           []rules.ConditionState{restrained},
				Source:          " feature pulse ",
				RollSeq:         &rollSeq,
			}),
		}, now))
		if got.CharacterID != ids.CharacterID("char-1") || got.Source != "feature pulse" || got.RollSeq == nil || *got.RollSeq != rollSeq {
			t.Fatalf("ConditionChangedPayload = %#v", got)
		}
	})

	t.Run("optional commands cover success and rejection edges", func(t *testing.T) {
		t.Parallel()

		gold := NewDecider([]command.Type{CommandTypeGoldUpdate})
		singleRejection(t, gold.Decide(nil, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeGoldUpdate,
			PayloadJSON: mustMarshalJSON(t, payload.GoldUpdatePayload{
				CharacterID: ids.CharacterID("char-1"),
				ChestsAfter: 2,
			}),
		}, now), rejectionCodeGoldInvalid)
		singleRejection(t, gold.Decide(nil, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeGoldUpdate,
			PayloadJSON: mustMarshalJSON(t, payload.GoldUpdatePayload{
				CharacterID:    ids.CharacterID("char-1"),
				HandfulsBefore: 1,
				HandfulsAfter:  1,
			}),
		}, now), rejectionCodeGoldInvalid)
		goldPayload := singleEventPayload[payload.GoldUpdatedPayload](t, gold.Decide(nil, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeGoldUpdate,
			PayloadJSON: mustMarshalJSON(t, payload.GoldUpdatePayload{
				CharacterID:    ids.CharacterID(" char-1 "),
				HandfulsBefore: 1,
				HandfulsAfter:  2,
				Reason:         " loot ",
			}),
		}, now))
		if goldPayload.CharacterID != ids.CharacterID("char-1") || goldPayload.Reason != "loot" {
			t.Fatalf("GoldUpdatedPayload = %#v", goldPayload)
		}

		domainCard := NewDecider([]command.Type{CommandTypeDomainCardAcquire})
		singleRejection(t, domainCard.Decide(nil, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeDomainCardAcquire,
			PayloadJSON: mustMarshalJSON(t, payload.DomainCardAcquirePayload{
				CharacterID: ids.CharacterID("char-1"),
				CardID:      "card-1",
				Destination: "stash",
			}),
		}, now), rejectionCodeDomainCardAcquireInvalid)
		cardPayload := singleEventPayload[payload.DomainCardAcquiredPayload](t, domainCard.Decide(nil, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeDomainCardAcquire,
			PayloadJSON: mustMarshalJSON(t, payload.DomainCardAcquirePayload{
				CharacterID: ids.CharacterID(" char-1 "),
				CardID:      " card-1 ",
				Destination: " loadout ",
			}),
		}, now))
		if cardPayload.CharacterID != ids.CharacterID("char-1") || cardPayload.CardID != "card-1" || cardPayload.Destination != "loadout" {
			t.Fatalf("DomainCardAcquiredPayload = %#v", cardPayload)
		}

		swap := NewDecider([]command.Type{CommandTypeEquipmentSwap})
		singleRejection(t, swap.Decide(nil, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeEquipmentSwap,
			PayloadJSON: mustMarshalJSON(t, payload.EquipmentSwapPayload{
				CharacterID: ids.CharacterID("char-1"),
				ItemID:      "item-1",
				ItemType:    "trinket",
				From:        "inventory",
				To:          "active",
			}),
		}, now), rejectionCodeEquipmentSwapInvalid)
		singleRejection(t, swap.Decide(nil, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeEquipmentSwap,
			PayloadJSON: mustMarshalJSON(t, payload.EquipmentSwapPayload{
				CharacterID: ids.CharacterID("char-1"),
				ItemID:      "item-1",
				ItemType:    "weapon",
				From:        "active",
				To:          "active",
			}),
		}, now), rejectionCodeEquipmentSwapInvalid)

		use := NewDecider([]command.Type{CommandTypeConsumableUse})
		singleRejection(t, use.Decide(nil, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeConsumableUse,
			PayloadJSON: mustMarshalJSON(t, payload.ConsumableUsePayload{
				CharacterID:  ids.CharacterID("char-1"),
				ConsumableID: "potion-1",
			}),
		}, now), rejectionCodeConsumableInvalid)
		singleRejection(t, use.Decide(nil, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeConsumableUse,
			PayloadJSON: mustMarshalJSON(t, payload.ConsumableUsePayload{
				CharacterID:    ids.CharacterID("char-1"),
				ConsumableID:   "potion-1",
				QuantityBefore: 2,
				QuantityAfter:  2,
			}),
		}, now), rejectionCodeConsumableInvalid)
		used := singleEventPayload[payload.ConsumableUsedPayload](t, use.Decide(nil, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeConsumableUse,
			PayloadJSON: mustMarshalJSON(t, payload.ConsumableUsePayload{
				CharacterID:    ids.CharacterID(" char-1 "),
				ConsumableID:   " potion-1 ",
				QuantityBefore: 2,
				QuantityAfter:  1,
			}),
		}, now))
		if used.CharacterID != ids.CharacterID("char-1") || used.ConsumableID != "potion-1" || used.Quantity != 1 {
			t.Fatalf("ConsumableUsedPayload = %#v", used)
		}

		acquire := NewDecider([]command.Type{CommandTypeConsumableAcquire})
		singleRejection(t, acquire.Decide(nil, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeConsumableAcquire,
			PayloadJSON: mustMarshalJSON(t, payload.ConsumableAcquirePayload{
				CharacterID:    ids.CharacterID("char-1"),
				ConsumableID:   "potion-1",
				QuantityBefore: 5,
				QuantityAfter:  6,
			}),
		}, now), rejectionCodeConsumableInvalid)
		singleRejection(t, acquire.Decide(nil, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeConsumableAcquire,
			PayloadJSON: mustMarshalJSON(t, payload.ConsumableAcquirePayload{
				CharacterID:    ids.CharacterID("char-1"),
				ConsumableID:   "potion-1",
				QuantityBefore: 1,
				QuantityAfter:  3,
			}),
		}, now), rejectionCodeConsumableInvalid)
		acquired := singleEventPayload[payload.ConsumableAcquiredPayload](t, acquire.Decide(nil, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeConsumableAcquire,
			PayloadJSON: mustMarshalJSON(t, payload.ConsumableAcquirePayload{
				CharacterID:    ids.CharacterID(" char-1 "),
				ConsumableID:   " potion-1 ",
				QuantityBefore: 1,
				QuantityAfter:  2,
			}),
		}, now))
		if acquired.CharacterID != ids.CharacterID("char-1") || acquired.ConsumableID != "potion-1" || acquired.Quantity != 2 {
			t.Fatalf("ConsumableAcquiredPayload = %#v", acquired)
		}
	})
}

func TestDeciderFeatureAndHelperCoverage(t *testing.T) {
	t.Parallel()

	now := func() time.Time { return time.Date(2026, 3, 27, 15, 0, 0, 0, time.UTC) }

	t.Run("helpers normalize and reject as expected", func(t *testing.T) {
		t.Parallel()

		if rejection := rejectArmorSpendLimit(2); rejection == nil || rejection.Code != rejectionCodeDamageArmorSpendLimit {
			t.Fatalf("rejectArmorSpendLimit(2) = %+v", rejection)
		}
		if rejection := rejectArmorSpendLimit(1); rejection != nil {
			t.Fatalf("rejectArmorSpendLimit(1) = %+v, want nil", rejection)
		}
		if rejection := rejectDamageBeforeMismatch(intPtr(2), 3, nil, 0, "HP_MISMATCH", "mismatch"); rejection == nil || rejection.Code != "HP_MISMATCH" {
			t.Fatalf("rejectDamageBeforeMismatch(hp) = %+v", rejection)
		}
		if rejection := rejectDamageBeforeMismatch(nil, 3, intPtr(2), 1, "ARMOR_MISMATCH", "mismatch"); rejection == nil || rejection.Code != "ARMOR_MISMATCH" {
			t.Fatalf("rejectDamageBeforeMismatch(armor) = %+v", rejection)
		}
		if rejection := rejectDamageBeforeMismatch(intPtr(3), 3, intPtr(1), 1, "MATCH", "match"); rejection != nil {
			t.Fatalf("rejectDamageBeforeMismatch(match) = %+v, want nil", rejection)
		}
		if got := derefInt(nil, 7); got != 7 {
			t.Fatalf("derefInt(nil, 7) = %d, want 7", got)
		}
		if got := derefInt(intPtr(3), 7); got != 3 {
			t.Fatalf("derefInt(value, 7) = %d, want 3", got)
		}

		snapshot := daggerheartstate.NewSnapshotState("camp-1")
		snapshot.CharacterStates[ids.CharacterID("char-1")] = daggerheartstate.CharacterState{HP: 5}
		character, ok := snapshotCharacterState(snapshot, ids.CharacterID(" char-1 "))
		if !ok || character.CharacterID != "char-1" || character.CampaignID != "camp-1" || character.LifeState != daggerheartstate.LifeStateAlive {
			t.Fatalf("snapshotCharacterState() = (%#v, %v)", character, ok)
		}
		if _, ok := snapshotCharacterState(snapshot, ids.CharacterID(" ")); ok {
			t.Fatal("snapshotCharacterState(blank) = ok, want false")
		}

		companion := companionStatePtrValue(&daggerheartstate.CharacterCompanionState{
			Status:             " AWAY ",
			ActiveExperienceID: " scouting ",
		})
		if companion == nil || companion.Status != daggerheartstate.CompanionStatusAway || companion.ActiveExperienceID != "scouting" {
			t.Fatalf("companionStatePtrValue() = %#v", companion)
		}
		if companionStatePtrValue(nil) != nil {
			t.Fatal("companionStatePtrValue(nil) != nil")
		}
	})

	t.Run("class and subclass feature branches", func(t *testing.T) {
		t.Parallel()

		featureDecider := NewDecider([]command.Type{CommandTypeClassFeatureApply})
		snapshot := daggerheartstate.NewSnapshotState("camp-1")
		snapshot.CharacterStates[ids.CharacterID("char-1")] = daggerheartstate.CharacterState{HP: 5, Hope: 2, Armor: 1}

		singleRejection(t, featureDecider.Decide(snapshot, command.Command{
			CampaignID:  ids.CampaignID("camp-1"),
			Type:        CommandTypeClassFeatureApply,
			PayloadJSON: mustMarshalJSON(t, payload.ClassFeatureApplyPayload{}),
		}, now), "CLASS_FEATURE_ACTOR_REQUIRED")
		singleRejection(t, featureDecider.Decide(snapshot, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeClassFeatureApply,
			PayloadJSON: mustMarshalJSON(t, payload.ClassFeatureApplyPayload{
				ActorCharacterID: ids.CharacterID("actor-1"),
				Feature:          "frontline",
				Targets: []payload.ClassFeatureTargetPatchPayload{
					{CharacterID: ids.CharacterID("char-1"), HPBefore: intPtr(4), HPAfter: intPtr(3)},
				},
			}),
		}, now), rejectionCodeDamageBeforeMismatch)
		singleRejection(t, featureDecider.Decide(snapshot, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeClassFeatureApply,
			PayloadJSON: mustMarshalJSON(t, payload.ClassFeatureApplyPayload{
				ActorCharacterID: ids.CharacterID("actor-1"),
				Feature:          "frontline",
				Targets: []payload.ClassFeatureTargetPatchPayload{
					{CharacterID: ids.CharacterID("char-1"), HPBefore: intPtr(5), HPAfter: intPtr(5)},
				},
			}),
		}, now), rejectionCodeCharacterStatePatchNoMutation)
		patch := singleEventPayload[payload.CharacterStatePatchedPayload](t, featureDecider.Decide(snapshot, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeClassFeatureApply,
			PayloadJSON: mustMarshalJSON(t, payload.ClassFeatureApplyPayload{
				ActorCharacterID: ids.CharacterID(" actor-1 "),
				Feature:          " frontline ",
				Targets: []payload.ClassFeatureTargetPatchPayload{
					{CharacterID: ids.CharacterID(" char-1 "), HPBefore: intPtr(5), HPAfter: intPtr(4)},
				},
			}),
		}, now))
		if patch.Source != "class_feature:frontline:actor-1" || patch.HP == nil || *patch.HP != 4 {
			t.Fatalf("CharacterStatePatchedPayload = %#v", patch)
		}

		subclassDecider := NewDecider([]command.Type{CommandTypeSubclassFeatureApply})
		singleRejection(t, subclassDecider.Decide(snapshot, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeSubclassFeatureApply,
			PayloadJSON: mustMarshalJSON(t, payload.SubclassFeatureApplyPayload{
				ActorCharacterID: ids.CharacterID("actor-1"),
				Feature:          "bind",
			}),
		}, now), "SUBCLASS_FEATURE_TARGET_REQUIRED")
		singleRejection(t, subclassDecider.Decide(snapshot, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeSubclassFeatureApply,
			PayloadJSON: mustMarshalJSON(t, payload.SubclassFeatureApplyPayload{
				ActorCharacterID: ids.CharacterID("actor-1"),
				Feature:          "bind",
				Targets: []payload.SubclassFeatureTargetPatchPayload{
					{CharacterID: ids.CharacterID("char-1"), StressBefore: intPtr(1), StressAfter: intPtr(0)},
				},
			}),
		}, now), rejectionCodeDamageBeforeMismatch)
		singleRejection(t, subclassDecider.Decide(snapshot, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeSubclassFeatureApply,
			PayloadJSON: mustMarshalJSON(t, payload.SubclassFeatureApplyPayload{
				ActorCharacterID: ids.CharacterID("actor-1"),
				Feature:          "bind",
				Targets: []payload.SubclassFeatureTargetPatchPayload{
					{CharacterID: ids.CharacterID("char-1"), HPBefore: intPtr(5), HPAfter: intPtr(5)},
				},
			}),
		}, now), rejectionCodeCharacterStatePatchNoMutation)
		subclassPatch := singleEventPayload[payload.CharacterStatePatchedPayload](t, subclassDecider.Decide(snapshot, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeSubclassFeatureApply,
			PayloadJSON: mustMarshalJSON(t, payload.SubclassFeatureApplyPayload{
				ActorCharacterID: ids.CharacterID(" actor-1 "),
				Feature:          " bind ",
				Targets: []payload.SubclassFeatureTargetPatchPayload{
					{CharacterID: ids.CharacterID(" char-1 "), ArmorBefore: intPtr(1), ArmorAfter: intPtr(0)},
				},
			}),
		}, now))
		if subclassPatch.Source != "subclass_feature:bind:actor-1" || subclassPatch.Armor == nil || *subclassPatch.Armor != 0 {
			t.Fatalf("Subclass CharacterStatePatchedPayload = %#v", subclassPatch)
		}
	})

	t.Run("companion, profile, and adversary branches", func(t *testing.T) {
		t.Parallel()

		companionBefore := &daggerheartstate.CharacterCompanionState{Status: daggerheartstate.CompanionStatusPresent}
		companionAfter := &daggerheartstate.CharacterCompanionState{
			Status:             daggerheartstate.CompanionStatusAway,
			ActiveExperienceID: "scouting",
		}

		begin := NewDecider([]command.Type{CommandTypeCompanionExperienceBegin})
		singleRejection(t, begin.Decide(nil, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeCompanionExperienceBegin,
			PayloadJSON: mustMarshalJSON(t, payload.CompanionExperienceBeginPayload{
				CharacterID:  ids.CharacterID("char-1"),
				ExperienceID: "scouting",
			}),
		}, now), "COMPANION_ACTOR_REQUIRED")
		singleRejection(t, begin.Decide(nil, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeCompanionExperienceBegin,
			PayloadJSON: mustMarshalJSON(t, payload.CompanionExperienceBeginPayload{
				ActorCharacterID:     ids.CharacterID("actor-1"),
				CharacterID:          ids.CharacterID("char-1"),
				ExperienceID:         "scouting",
				CompanionStateBefore: companionBefore,
				CompanionStateAfter:  companionBefore,
			}),
		}, now), rejectionCodeCharacterStatePatchNoMutation)
		begun := singleEventPayload[payload.CompanionExperienceBegunPayload](t, begin.Decide(nil, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeCompanionExperienceBegin,
			PayloadJSON: mustMarshalJSON(t, payload.CompanionExperienceBeginPayload{
				ActorCharacterID:     ids.CharacterID(" actor-1 "),
				CharacterID:          ids.CharacterID(" char-1 "),
				ExperienceID:         " scouting ",
				CompanionStateBefore: companionBefore,
				CompanionStateAfter:  companionAfter,
			}),
		}, now))
		if begun.CharacterID != ids.CharacterID("char-1") || begun.ExperienceID != "scouting" || begun.Source != "companion.experience.begin" {
			t.Fatalf("CompanionExperienceBegunPayload = %#v", begun)
		}

		returnDecider := NewDecider([]command.Type{CommandTypeCompanionReturn})
		singleRejection(t, returnDecider.Decide(nil, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeCompanionReturn,
			PayloadJSON: mustMarshalJSON(t, payload.CompanionReturnPayload{
				ActorCharacterID: ids.CharacterID("actor-1"),
				CharacterID:      ids.CharacterID("char-1"),
			}),
		}, now), "COMPANION_RETURN_RESOLUTION_REQUIRED")
		stressBefore := 1
		stressAfter := 0
		returned := singleEventPayload[payload.CompanionReturnedPayload](t, returnDecider.Decide(nil, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeCompanionReturn,
			PayloadJSON: mustMarshalJSON(t, payload.CompanionReturnPayload{
				ActorCharacterID:     ids.CharacterID(" actor-1 "),
				CharacterID:          ids.CharacterID(" char-1 "),
				Resolution:           " complete ",
				StressBefore:         &stressBefore,
				StressAfter:          &stressAfter,
				CompanionStateBefore: companionAfter,
				CompanionStateAfter:  companionBefore,
			}),
		}, now))
		if returned.CharacterID != ids.CharacterID("char-1") || returned.Resolution != "complete" || returned.Source != "companion.return" {
			t.Fatalf("CompanionReturnedPayload = %#v", returned)
		}

		replace := NewDecider([]command.Type{CommandTypeCharacterProfileReplace})
		singleRejection(t, replace.Decide(nil, command.Command{
			Type:        CommandTypeCharacterProfileReplace,
			PayloadJSON: mustMarshalJSON(t, daggerheartstate.CharacterProfileReplacePayload{}),
		}, now), rejectionCodePayloadDecodeFailed)

		adversarySnapshot := daggerheartstate.NewSnapshotState("camp-1")
		adversarySnapshot.AdversaryStates[dhids.AdversaryID("adv-1")] = daggerheartstate.AdversaryState{
			AdversaryID: dhids.AdversaryID("adv-1"),
			Conditions:  []string{rules.ConditionHidden},
		}
		hidden := mustConditionState(t, rules.ConditionHidden)
		restrained := mustConditionState(t, rules.ConditionRestrained)
		adversary := NewDecider([]command.Type{CommandTypeAdversaryConditionChange})
		singleRejection(t, adversary.Decide(adversarySnapshot, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeAdversaryConditionChange,
			PayloadJSON: mustMarshalJSON(t, payload.AdversaryConditionChangePayload{
				AdversaryID:      dhids.AdversaryID("adv-1"),
				ConditionsAfter:  []rules.ConditionState{hidden},
				ConditionsBefore: []rules.ConditionState{hidden},
			}),
		}, now), rejectionCodeAdversaryConditionNoMutation)
		rollSeq := uint64(5)
		changed := singleEventPayload[payload.AdversaryConditionChangedPayload](t, adversary.Decide(adversarySnapshot, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       CommandTypeAdversaryConditionChange,
			PayloadJSON: mustMarshalJSON(t, payload.AdversaryConditionChangePayload{
				AdversaryID:     dhids.AdversaryID(" adv-1 "),
				ConditionsAfter: []rules.ConditionState{hidden, restrained},
				Added:           []rules.ConditionState{restrained},
				Source:          " fear move ",
				RollSeq:         &rollSeq,
			}),
		}, now))
		if changed.AdversaryID != dhids.AdversaryID("adv-1") || changed.Source != "fear move" || changed.RollSeq == nil || *changed.RollSeq != rollSeq {
			t.Fatalf("AdversaryConditionChangedPayload = %#v", changed)
		}
	})
}

func intPtr(value int) *int {
	return &value
}
