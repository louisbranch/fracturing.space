package decider

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func TestDeciderAdditionalCommandCoverage(t *testing.T) {
	t.Parallel()

	now := func() time.Time { return time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC) }

	t.Run("beastform transform emits normalized event", func(t *testing.T) {
		t.Parallel()

		hopeBefore := 2
		hopeAfter := 1
		stressBefore := 0
		stressAfter := 1
		cmd := command.Command{
			CampaignID:    ids.CampaignID("camp-1"),
			Type:          commandTypeBeastformTransform,
			EntityType:    "character",
			EntityID:      "char-1",
			SystemID:      "daggerheart",
			SystemVersion: "v1",
			PayloadJSON: mustMarshalJSON(t, payload.BeastformTransformPayload{
				ActorCharacterID: ids.CharacterID(" actor-1 "),
				CharacterID:      ids.CharacterID(" char-1 "),
				BeastformID:      " wolf-form ",
				HopeBefore:       &hopeBefore,
				HopeAfter:        &hopeAfter,
				StressBefore:     &stressBefore,
				StressAfter:      &stressAfter,
				ClassStateAfter: &daggerheartstate.CharacterClassState{
					ActiveBeastform: &daggerheartstate.CharacterActiveBeastformState{
						BeastformID: " wolf-form ",
						BaseTrait:   " agility ",
					},
				},
			}),
		}

		decision := NewDecider([]command.Type{commandTypeBeastformTransform}).Decide(nil, cmd, now)
		eventPayload := singleEventPayload[payload.BeastformTransformedPayload](t, decision)
		if eventPayload.CharacterID != ids.CharacterID("char-1") {
			t.Fatalf("CharacterID = %q, want %q", eventPayload.CharacterID, ids.CharacterID("char-1"))
		}
		if eventPayload.BeastformID != "wolf-form" {
			t.Fatalf("BeastformID = %q, want %q", eventPayload.BeastformID, "wolf-form")
		}
		if eventPayload.Source != "beastform.transform" {
			t.Fatalf("Source = %q, want beastform.transform", eventPayload.Source)
		}
		if eventPayload.ActiveBeastform == nil || eventPayload.ActiveBeastform.BeastformID != "wolf-form" {
			t.Fatalf("ActiveBeastform = %#v", eventPayload.ActiveBeastform)
		}
	})

	t.Run("beastform drop rejects unchanged snapshot", func(t *testing.T) {
		t.Parallel()

		snapshot := daggerheartstate.NewSnapshotState("camp-1")
		snapshot.CharacterStates[ids.CharacterID("char-1")] = daggerheartstate.CharacterState{}
		snapshot.CharacterClassStates[ids.CharacterID("char-1")] = daggerheartstate.CharacterClassState{
			ActiveBeastform: &daggerheartstate.CharacterActiveBeastformState{BeastformID: "wolf-form"},
		}
		before := &daggerheartstate.CharacterClassState{
			ActiveBeastform: &daggerheartstate.CharacterActiveBeastformState{BeastformID: "wolf-form"},
		}

		decision := NewDecider([]command.Type{commandTypeBeastformDrop}).Decide(snapshot, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       commandTypeBeastformDrop,
			PayloadJSON: mustMarshalJSON(t, payload.BeastformDropPayload{
				ActorCharacterID: ids.CharacterID("actor-1"),
				CharacterID:      ids.CharacterID("char-1"),
				BeastformID:      "wolf-form",
				ClassStateBefore: before,
				ClassStateAfter:  before,
			}),
		}, now)

		singleRejection(t, decision, rejectionCodeCharacterStatePatchNoMutation)
	})

	t.Run("companion begin emits normalized companion state", func(t *testing.T) {
		t.Parallel()

		cmd := command.Command{
			CampaignID:    ids.CampaignID("camp-1"),
			Type:          commandTypeCompanionExperienceBegin,
			EntityType:    "character",
			EntityID:      "char-1",
			SystemID:      "daggerheart",
			SystemVersion: "v1",
			PayloadJSON: mustMarshalJSON(t, payload.CompanionExperienceBeginPayload{
				ActorCharacterID: ids.CharacterID(" actor-1 "),
				CharacterID:      ids.CharacterID(" char-1 "),
				ExperienceID:     " scouting ",
				CompanionStateAfter: &daggerheartstate.CharacterCompanionState{
					Status:             " AWAY ",
					ActiveExperienceID: " scouting ",
				},
			}),
		}

		decision := NewDecider([]command.Type{commandTypeCompanionExperienceBegin}).Decide(nil, cmd, now)
		eventPayload := singleEventPayload[payload.CompanionExperienceBegunPayload](t, decision)
		if eventPayload.ExperienceID != "scouting" {
			t.Fatalf("ExperienceID = %q, want scouting", eventPayload.ExperienceID)
		}
		if eventPayload.CompanionState == nil || eventPayload.CompanionState.Status != daggerheartstate.CompanionStatusAway {
			t.Fatalf("CompanionState = %#v", eventPayload.CompanionState)
		}
	})

	t.Run("companion return rejects unchanged payload", func(t *testing.T) {
		t.Parallel()

		before := &daggerheartstate.CharacterCompanionState{
			Status:             daggerheartstate.CompanionStatusAway,
			ActiveExperienceID: "scouting",
		}

		decision := NewDecider([]command.Type{commandTypeCompanionReturn}).Decide(nil, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       commandTypeCompanionReturn,
			PayloadJSON: mustMarshalJSON(t, payload.CompanionReturnPayload{
				ActorCharacterID:     ids.CharacterID("actor-1"),
				CharacterID:          ids.CharacterID("char-1"),
				Resolution:           " complete ",
				CompanionStateBefore: before,
				CompanionStateAfter:  before,
			}),
		}, now)

		singleRejection(t, decision, rejectionCodeCharacterStatePatchNoMutation)
	})

	t.Run("multi target damage emits one event per target", func(t *testing.T) {
		t.Parallel()

		hpBeforeA := 6
		hpAfterA := 4
		armorBeforeA := 1
		armorAfterA := 0
		hpBeforeB := 8
		hpAfterB := 6
		stressAfterB := 1
		snapshot := daggerheartstate.NewSnapshotState("camp-1")
		snapshot.CharacterStates[ids.CharacterID("char-1")] = daggerheartstate.CharacterState{HP: hpBeforeA, Armor: armorBeforeA}
		snapshot.CharacterStates[ids.CharacterID("char-2")] = daggerheartstate.CharacterState{HP: hpBeforeB}

		decision := NewDecider([]command.Type{commandTypeMultiTargetDamageApply}).Decide(snapshot, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       commandTypeMultiTargetDamageApply,
			PayloadJSON: mustMarshalJSON(t, payload.MultiTargetDamageApplyPayload{
				Targets: []payload.DamageApplyPayload{
					{
						CharacterID: ids.CharacterID(" char-1 "),
						HpBefore:    &hpBeforeA,
						HpAfter:     &hpAfterA,
						ArmorBefore: &armorBeforeA,
						ArmorAfter:  &armorAfterA,
						DamageType:  " physical ",
						Source:      " trap ",
					},
					{
						CharacterID: ids.CharacterID(" char-2 "),
						HpBefore:    &hpBeforeB,
						HpAfter:     &hpAfterB,
						StressAfter: &stressAfterB,
						DamageType:  " magic ",
					},
				},
			}),
		}, now)

		if len(decision.Rejections) != 0 {
			t.Fatalf("unexpected rejections: %+v", decision.Rejections)
		}
		if len(decision.Events) != 2 {
			t.Fatalf("events = %d, want 2", len(decision.Events))
		}

		first := decodeEventPayload[payload.DamageAppliedPayload](t, decision.Events[0])
		second := decodeEventPayload[payload.DamageAppliedPayload](t, decision.Events[1])
		if first.CharacterID != ids.CharacterID("char-1") || first.DamageType != "physical" || first.Source != "trap" {
			t.Fatalf("first event payload = %#v", first)
		}
		if second.CharacterID != ids.CharacterID("char-2") || second.DamageType != "magic" {
			t.Fatalf("second event payload = %#v", second)
		}
	})

	t.Run("environment update and delete normalize payloads", func(t *testing.T) {
		t.Parallel()

		update := NewDecider([]command.Type{commandTypeEnvironmentEntityUpdate}).Decide(nil, command.Command{
			CampaignID:    ids.CampaignID("camp-1"),
			Type:          commandTypeEnvironmentEntityUpdate,
			SystemID:      "daggerheart",
			SystemVersion: "v1",
			PayloadJSON: mustMarshalJSON(t, payload.EnvironmentEntityUpdatePayload{
				EnvironmentEntityID: dhids.EnvironmentEntityID(" env-1 "),
				EnvironmentID:       " fog-bank ",
				Name:                " Fog Bank ",
				Type:                " hazard ",
				Tier:                2,
				Difficulty:          14,
				SessionID:           ids.SessionID(" sess-1 "),
				SceneID:             ids.SceneID(" scene-1 "),
				Notes:               " heavy fog ",
			}),
		}, now)
		updatePayload := singleEventPayload[payload.EnvironmentEntityUpdatedPayload](t, update)
		if updatePayload.EnvironmentEntityID != dhids.EnvironmentEntityID("env-1") || updatePayload.Name != "Fog Bank" {
			t.Fatalf("update payload = %#v", updatePayload)
		}
		if updatePayload.Notes != "heavy fog" {
			t.Fatalf("update notes = %q, want heavy fog", updatePayload.Notes)
		}

		deleteDecision := NewDecider([]command.Type{commandTypeEnvironmentEntityDelete}).Decide(nil, command.Command{
			CampaignID:    ids.CampaignID("camp-1"),
			Type:          commandTypeEnvironmentEntityDelete,
			SystemID:      "daggerheart",
			SystemVersion: "v1",
			PayloadJSON: mustMarshalJSON(t, payload.EnvironmentEntityDeletePayload{
				EnvironmentEntityID: dhids.EnvironmentEntityID(" env-1 "),
				Reason:              " resolved ",
			}),
		}, now)
		deletePayload := singleEventPayload[payload.EnvironmentEntityDeletedPayload](t, deleteDecision)
		if deletePayload.EnvironmentEntityID != dhids.EnvironmentEntityID("env-1") || deletePayload.Reason != "resolved" {
			t.Fatalf("delete payload = %#v", deletePayload)
		}
	})

	t.Run("character profile delete preserves actor type and trims reason", func(t *testing.T) {
		t.Parallel()

		decision := NewDecider([]command.Type{commandTypeCharacterProfileDelete}).Decide(nil, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       commandTypeCharacterProfileDelete,
			ActorType:  command.ActorTypeGM,
			PayloadJSON: mustMarshalJSON(t, daggerheartstate.CharacterProfileDeletePayload{
				CharacterID: ids.CharacterID(" char-1 "),
				Reason:      " archived ",
			}),
		}, now)

		if len(decision.Events) != 1 {
			t.Fatalf("events = %d, want 1", len(decision.Events))
		}
		if decision.Events[0].ActorType != event.ActorType(command.ActorTypeGM) {
			t.Fatalf("ActorType = %q, want %q", decision.Events[0].ActorType, event.ActorType(command.ActorTypeGM))
		}
		eventPayload := decodeEventPayload[daggerheartstate.CharacterProfileDeletedPayload](t, decision.Events[0])
		if eventPayload.CharacterID != ids.CharacterID("char-1") || eventPayload.Reason != "archived" {
			t.Fatalf("event payload = %#v", eventPayload)
		}
	})

	t.Run("subclass feature emits character and condition events", func(t *testing.T) {
		t.Parallel()

		hpBefore := 6
		hpAfter := 5
		hidden := mustConditionState(t, rules.ConditionHidden)
		restrained := mustConditionState(t, rules.ConditionRestrained)
		rollSeq := uint64(7)
		snapshot := daggerheartstate.NewSnapshotState("camp-1")
		snapshot.CharacterStates[ids.CharacterID("char-1")] = daggerheartstate.CharacterState{HP: hpBefore}

		decision := NewDecider([]command.Type{commandTypeSubclassFeatureApply}).Decide(snapshot, command.Command{
			CampaignID: ids.CampaignID("camp-1"),
			Type:       commandTypeSubclassFeatureApply,
			PayloadJSON: mustMarshalJSON(t, payload.SubclassFeatureApplyPayload{
				ActorCharacterID: ids.CharacterID(" actor-1 "),
				Feature:          " elemental nova ",
				Targets: []payload.SubclassFeatureTargetPatchPayload{
					{
						CharacterID: ids.CharacterID("char-1"),
						HPBefore:    &hpBefore,
						HPAfter:     &hpAfter,
					},
				},
				CharacterConditionTargets: []payload.ConditionChangePayload{
					{
						CharacterID:     ids.CharacterID("char-1"),
						ConditionsAfter: []rules.ConditionState{hidden, restrained},
						Added:           []rules.ConditionState{restrained},
						RollSeq:         &rollSeq,
					},
				},
				AdversaryConditionTargets: []payload.AdversaryConditionChangePayload{
					{
						AdversaryID:     dhids.AdversaryID("adv-1"),
						ConditionsAfter: []rules.ConditionState{restrained},
						Added:           []rules.ConditionState{restrained},
						Source:          " custom-source ",
					},
				},
			}),
		}, now)

		if len(decision.Rejections) != 0 {
			t.Fatalf("unexpected rejections: %+v", decision.Rejections)
		}
		if len(decision.Events) != 3 {
			t.Fatalf("events = %d, want 3", len(decision.Events))
		}

		patchPayload := decodeEventPayload[payload.CharacterStatePatchedPayload](t, decision.Events[0])
		if patchPayload.CharacterID != ids.CharacterID("char-1") || patchPayload.Source != "subclass_feature:elemental nova:actor-1" {
			t.Fatalf("patch payload = %#v", patchPayload)
		}

		conditionPayload := decodeEventPayload[payload.ConditionChangedPayload](t, decision.Events[1])
		if conditionPayload.Source != "subclass_feature:elemental nova:actor-1" || conditionPayload.RollSeq == nil || *conditionPayload.RollSeq != rollSeq {
			t.Fatalf("condition payload = %#v", conditionPayload)
		}

		adversaryPayload := decodeEventPayload[payload.AdversaryConditionChangedPayload](t, decision.Events[2])
		if adversaryPayload.AdversaryID != dhids.AdversaryID("adv-1") || adversaryPayload.Source != "custom-source" {
			t.Fatalf("adversary payload = %#v", adversaryPayload)
		}
	})
}

func singleEventPayload[T any](t *testing.T, decision command.Decision) T {
	t.Helper()

	if len(decision.Rejections) != 0 {
		t.Fatalf("unexpected rejections: %+v", decision.Rejections)
	}
	if len(decision.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(decision.Events))
	}
	return decodeEventPayload[T](t, decision.Events[0])
}

func decodeEventPayload[T any](t *testing.T, evt event.Event) T {
	t.Helper()

	var got T
	if err := json.Unmarshal(evt.PayloadJSON, &got); err != nil {
		t.Fatalf("json.Unmarshal(%T): %v", got, err)
	}
	return got
}

func singleRejection(t *testing.T, decision command.Decision, wantCode string) {
	t.Helper()

	if len(decision.Events) != 0 {
		t.Fatalf("events = %d, want 0", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("rejections = %d, want 1", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != wantCode {
		t.Fatalf("rejection code = %q, want %q", decision.Rejections[0].Code, wantCode)
	}
}
