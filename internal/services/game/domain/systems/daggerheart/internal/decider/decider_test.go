package decider

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func TestDeciderDecideLevelUpApplyAcceptsAndTransforms(t *testing.T) {
	t.Parallel()

	now := func() time.Time { return time.Date(2025, 2, 3, 4, 5, 6, 0, time.UTC) }
	decider := NewDecider([]command.Type{commandTypeLevelUpApply})
	cmd := command.Command{
		CampaignID:    ids.CampaignID("camp-1"),
		Type:          commandTypeLevelUpApply,
		EntityType:    "character",
		EntityID:      "char-1",
		SystemID:      "daggerheart",
		SystemVersion: "v1",
		PayloadJSON: mustMarshalJSON(t, payload.LevelUpApplyPayload{
			CharacterID: ids.CharacterID("char-1"),
			LevelBefore: 1,
			LevelAfter:  2,
			Advancements: []payload.LevelUpAdvancementPayload{
				{Type: string(mechanics.AdvTraitIncrease), Trait: "agility"},
				{Type: string(mechanics.AdvAddHPSlots)},
			},
		}),
	}

	decision := decider.Decide(nil, cmd, now)
	if len(decision.Rejections) != 0 {
		t.Fatalf("unexpected rejection: %+v", decision.Rejections)
	}
	if len(decision.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != payload.EventTypeLevelUpApplied {
		t.Fatalf("event type = %q, want %q", evt.Type, payload.EventTypeLevelUpApplied)
	}
	if evt.EntityID != "char-1" {
		t.Fatalf("entity id = %q, want char-1", evt.EntityID)
	}
	if !evt.Timestamp.Equal(now()) {
		t.Fatalf("timestamp = %v, want %v", evt.Timestamp, now())
	}

	var got payload.LevelUpAppliedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &got); err != nil {
		t.Fatalf("json.Unmarshal(event payload): %v", err)
	}
	if got.Level != 2 {
		t.Fatalf("Level = %d, want 2", got.Level)
	}
	if got.Tier != 2 || !got.IsTierEntry {
		t.Fatalf("tier result = %+v, want tier 2 tier-entry true", got)
	}
	if got.ClearMarks {
		t.Fatalf("ClearMarks = true, want false for level 2")
	}
	if got.ThresholdDelta != 1 {
		t.Fatalf("ThresholdDelta = %d, want 1", got.ThresholdDelta)
	}
}

func TestDeciderDecideLevelUpApplyRejectsInvalidRequest(t *testing.T) {
	t.Parallel()

	decider := NewDecider([]command.Type{commandTypeLevelUpApply})
	cmd := command.Command{
		CampaignID: ids.CampaignID("camp-1"),
		Type:       commandTypeLevelUpApply,
		PayloadJSON: mustMarshalJSON(t, payload.LevelUpApplyPayload{
			CharacterID: ids.CharacterID("char-1"),
			LevelBefore: 1,
			LevelAfter:  3,
			Advancements: []payload.LevelUpAdvancementPayload{
				{Type: string(mechanics.AdvTraitIncrease), Trait: "agility"},
			},
		}),
	}

	decision := decider.Decide(nil, cmd, time.Now)
	if len(decision.Rejections) != 1 {
		t.Fatalf("rejections = %d, want 1", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "LEVEL_UP_INVALID" {
		t.Fatalf("rejection code = %q, want LEVEL_UP_INVALID", decision.Rejections[0].Code)
	}
	if !strings.Contains(decision.Rejections[0].Message, "level_after must be level_before + 1") {
		t.Fatalf("rejection message = %q", decision.Rejections[0].Message)
	}
}

func TestDeciderRejectsUnsupportedCommandType(t *testing.T) {
	t.Parallel()

	decision := NewDecider(nil).Decide(nil, command.Command{Type: command.Type("sys.daggerheart.unknown")}, time.Now)
	if len(decision.Rejections) != 1 {
		t.Fatalf("rejections = %d, want 1", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCommandTypeUnsupported {
		t.Fatalf("rejection code = %q, want %q", decision.Rejections[0].Code, rejectionCodeCommandTypeUnsupported)
	}
}

func TestDeciderHelperFunctions(t *testing.T) {
	t.Parallel()

	t.Run("character state no mutation", func(t *testing.T) {
		t.Parallel()

		snapshot := daggerheartstate.NewSnapshotState("camp-1")
		snapshot.CharacterStates[ids.CharacterID("char-1")] = daggerheartstate.CharacterState{
			HP:        5,
			Hope:      2,
			Armor:     1,
			LifeState: daggerheartstate.LifeStateAlive,
		}

		sameHP := 5
		changedHP := 4
		if !isCharacterStatePatchNoMutation(snapshot, payload.CharacterStatePatchPayload{
			CharacterID: ids.CharacterID("char-1"),
			HPAfter:     &sameHP,
		}) {
			t.Fatal("isCharacterStatePatchNoMutation() = false, want true")
		}
		if isCharacterStatePatchNoMutation(snapshot, payload.CharacterStatePatchPayload{
			CharacterID: ids.CharacterID("char-1"),
			HPAfter:     &changedHP,
		}) {
			t.Fatal("isCharacterStatePatchNoMutation() = true, want false")
		}
	})

	t.Run("condition helpers", func(t *testing.T) {
		t.Parallel()

		snapshot := daggerheartstate.NewSnapshotState("camp-1")
		snapshot.CharacterStates[ids.CharacterID("char-1")] = daggerheartstate.CharacterState{
			Conditions: []string{rules.ConditionHidden},
		}
		hidden := mustConditionState(t, rules.ConditionHidden)
		restrained := mustConditionState(t, rules.ConditionRestrained)

		if !isConditionChangeNoMutation(snapshot, payload.ConditionChangePayload{
			CharacterID:     ids.CharacterID("char-1"),
			ConditionsAfter: []rules.ConditionState{hidden},
		}) {
			t.Fatal("isConditionChangeNoMutation() = false, want true")
		}
		if !hasMissingCharacterConditionRemovals(snapshot, payload.ConditionChangePayload{
			CharacterID: ids.CharacterID("char-1"),
			Removed:     []rules.ConditionState{restrained},
		}) {
			t.Fatal("hasMissingCharacterConditionRemovals() = false, want true")
		}
	})

	t.Run("countdown snapshot rejections", func(t *testing.T) {
		t.Parallel()

		snapshot := daggerheartstate.NewSnapshotState("camp-1")
		snapshot.SceneCountdownStates[dhids.CountdownID("scene-count")] = daggerheartstate.SceneCountdownState{
			CountdownID:    dhids.CountdownID("scene-count"),
			RemainingValue: 3,
			Status:         rules.CountdownStatusActive,
		}
		snapshot.CampaignCountdownStates[dhids.CountdownID("camp-count")] = daggerheartstate.CampaignCountdownState{
			CountdownID:    dhids.CountdownID("camp-count"),
			RemainingValue: 4,
			Status:         rules.CountdownStatusActive,
		}

		if rejection := sceneCountdownAdvanceSnapshotRejection(snapshot, payload.SceneCountdownAdvancePayload{
			CountdownID:     dhids.CountdownID("scene-count"),
			BeforeRemaining: 2,
			AfterRemaining:  1,
			StatusAfter:     rules.CountdownStatusActive,
		}); rejection == nil || rejection.Code != rejectionCodeCountdownBeforeMismatch {
			t.Fatalf("scene rejection = %+v, want before mismatch", rejection)
		}

		if rejection := campaignCountdownAdvanceSnapshotRejection(snapshot, payload.CampaignCountdownAdvancePayload{
			CountdownID:     dhids.CountdownID("camp-count"),
			BeforeRemaining: 4,
			AfterRemaining:  4,
			StatusAfter:     rules.CountdownStatusActive,
		}); rejection == nil || rejection.Code != rejectionCodeCountdownAdvanceNoMutation {
			t.Fatalf("campaign rejection = %+v, want no mutation", rejection)
		}

		if rejection := sceneCountdownAdvanceSnapshotRejection(snapshot, payload.SceneCountdownAdvancePayload{
			CountdownID:     dhids.CountdownID("scene-count"),
			BeforeRemaining: 3,
			AfterRemaining:  1,
			StatusAfter:     rules.CountdownStatusTriggerPending,
		}); rejection != nil {
			t.Fatalf("scene rejection = %+v, want nil", rejection)
		}
	})

	t.Run("environment and adversary helpers", func(t *testing.T) {
		t.Parallel()

		snapshot := daggerheartstate.NewSnapshotState("camp-1")
		snapshot.EnvironmentStates[dhids.EnvironmentEntityID("env-1")] = daggerheartstate.EnvironmentEntityState{
			EnvironmentEntityID: dhids.EnvironmentEntityID("env-1"),
			EnvironmentID:       "fog-bank",
			Name:                "Fog Bank",
			Type:                "hazard",
			Tier:                2,
			Difficulty:          14,
			SessionID:           ids.SessionID("sess-1"),
			SceneID:             ids.SceneID("scene-1"),
			Notes:               "heavy fog",
		}
		pending := &rules.AdversaryPendingExperience{Name: "rage", Modifier: 1}
		snapshot.AdversaryStates[dhids.AdversaryID("adv-1")] = daggerheartstate.AdversaryState{
			AdversaryID:       dhids.AdversaryID("adv-1"),
			AdversaryEntryID:  "entry-1",
			Name:              "Hunter",
			Kind:              "solo",
			SessionID:         ids.SessionID("sess-1"),
			SceneID:           ids.SceneID("scene-1"),
			HP:                10,
			HPMax:             10,
			Stress:            2,
			StressMax:         4,
			Evasion:           12,
			Major:             18,
			Severe:            24,
			Armor:             1,
			Conditions:        []string{rules.ConditionHidden},
			FeatureStates:     []rules.AdversaryFeatureState{{FeatureID: "momentum", Status: "ready"}},
			PendingExperience: pending,
		}

		if !isEnvironmentEntityCreateNoMutation(snapshot, payload.EnvironmentEntityCreatePayload{
			EnvironmentEntityID: dhids.EnvironmentEntityID("env-1"),
			EnvironmentID:       "fog-bank",
			Name:                "Fog Bank",
			Type:                "hazard",
			Tier:                2,
			Difficulty:          14,
			SessionID:           ids.SessionID("sess-1"),
			SceneID:             ids.SceneID("scene-1"),
			Notes:               "heavy fog",
		}) {
			t.Fatal("isEnvironmentEntityCreateNoMutation() = false, want true")
		}

		if !isAdversaryCreateNoMutation(snapshot, payload.AdversaryCreatePayload{
			AdversaryID:       dhids.AdversaryID("adv-1"),
			AdversaryEntryID:  "entry-1",
			Name:              "Hunter",
			Kind:              "solo",
			SessionID:         ids.SessionID("sess-1"),
			SceneID:           ids.SceneID("scene-1"),
			HP:                10,
			HPMax:             10,
			Stress:            2,
			StressMax:         4,
			Evasion:           12,
			Major:             18,
			Severe:            24,
			Armor:             1,
			FeatureStates:     []rules.AdversaryFeatureState{{FeatureID: "momentum", Status: "ready"}},
			PendingExperience: &rules.AdversaryPendingExperience{Name: "rage", Modifier: 1},
			SpotlightGateID:   "",
			SpotlightCount:    0,
		}) {
			t.Fatal("isAdversaryCreateNoMutation() = false, want true")
		}

		if !hasMissingAdversaryConditionRemovals(snapshot, payload.AdversaryConditionChangePayload{
			AdversaryID: dhids.AdversaryID("adv-1"),
			Removed:     []rules.ConditionState{mustConditionState(t, rules.ConditionRestrained)},
		}) {
			t.Fatal("hasMissingAdversaryConditionRemovals() = false, want true")
		}
	})
}

func mustMarshalJSON(t *testing.T, value any) []byte {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(%T): %v", value, err)
	}
	return data
}

func mustConditionState(t *testing.T, code string) rules.ConditionState {
	t.Helper()

	state, err := rules.StandardConditionState(code)
	if err != nil {
		t.Fatalf("rules.StandardConditionState(%q): %v", code, err)
	}
	return state
}
