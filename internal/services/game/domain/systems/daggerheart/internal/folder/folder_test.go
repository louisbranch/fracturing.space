package folder

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func TestFoldSceneCountdownLifecycle(t *testing.T) {
	t.Parallel()

	state := daggerheartstate.NewSnapshotState("camp-1")
	f := &Folder{}

	err := f.foldSceneCountdownCreated(&state, payload.SceneCountdownCreatedPayload{
		SessionID:         ids.SessionID("sess-1"),
		SceneID:           ids.SceneID("scene-1"),
		CountdownID:       dhids.CountdownID("count-1"),
		Name:              "Escalation",
		Tone:              "grim",
		AdvancementPolicy: "manual",
		StartingValue:     6,
		RemainingValue:    4,
		LoopBehavior:      "none",
		Status:            "active",
		LinkedCountdownID: dhids.CountdownID("linked-1"),
		StartingRoll: &payload.CountdownStartingRollPayload{
			Min:   1,
			Max:   6,
			Value: 4,
		},
	})
	if err != nil {
		t.Fatalf("foldSceneCountdownCreated() error = %v", err)
	}

	got := state.SceneCountdownStates[dhids.CountdownID("count-1")]
	if got.CampaignID != ids.CampaignID("camp-1") {
		t.Fatalf("CampaignID = %q, want camp-1", got.CampaignID)
	}
	if got.SessionID != ids.SessionID("sess-1") || got.SceneID != ids.SceneID("scene-1") {
		t.Fatalf("session/scene = %q/%q, want sess-1/scene-1", got.SessionID, got.SceneID)
	}
	if got.Name != "Escalation" || got.RemainingValue != 4 || got.Status != "active" {
		t.Fatalf("countdown = %+v", got)
	}
	if got.StartingRoll == nil || got.StartingRoll.Value != 4 {
		t.Fatalf("StartingRoll = %+v, want value 4", got.StartingRoll)
	}

	err = f.foldSceneCountdownDeleted(&state, payload.SceneCountdownDeletedPayload{
		CountdownID: dhids.CountdownID("count-1"),
	})
	if err != nil {
		t.Fatalf("foldSceneCountdownDeleted() error = %v", err)
	}
	if _, ok := state.SceneCountdownStates[dhids.CountdownID("count-1")]; ok {
		t.Fatal("scene countdown still present after delete")
	}
}

func TestFoldCampaignCountdownLifecycle(t *testing.T) {
	t.Parallel()

	state := daggerheartstate.NewSnapshotState("camp-1")
	f := &Folder{}

	err := f.foldCampaignCountdownCreated(&state, payload.CampaignCountdownCreatedPayload{
		CountdownID:       dhids.CountdownID("camp-count-1"),
		Name:              "Long Rest Clock",
		Tone:              "ominous",
		AdvancementPolicy: "manual",
		StartingValue:     8,
		RemainingValue:    6,
		LoopBehavior:      "loop",
		Status:            "active",
	})
	if err != nil {
		t.Fatalf("foldCampaignCountdownCreated() error = %v", err)
	}

	err = f.foldCampaignCountdownAdvanced(&state, payload.CampaignCountdownAdvancedPayload{
		CountdownID:     dhids.CountdownID("camp-count-1"),
		BeforeRemaining: 6,
		AfterRemaining:  3,
		StatusBefore:    "active",
		StatusAfter:     "triggered",
	})
	if err != nil {
		t.Fatalf("foldCampaignCountdownAdvanced() error = %v", err)
	}

	got := state.CampaignCountdownStates[dhids.CountdownID("camp-count-1")]
	if got.RemainingValue != 3 || got.Status != "triggered" {
		t.Fatalf("campaign countdown = %+v, want remaining 3 triggered", got)
	}

	err = f.foldCampaignCountdownTriggerResolved(&state, payload.CampaignCountdownTriggerResolvedPayload{
		CountdownID:          dhids.CountdownID("camp-count-1"),
		StartingValueBefore:  8,
		StartingValueAfter:   10,
		RemainingValueBefore: 3,
		RemainingValueAfter:  10,
		StatusBefore:         "triggered",
		StatusAfter:          "active",
	})
	if err != nil {
		t.Fatalf("foldCampaignCountdownTriggerResolved() error = %v", err)
	}

	got = state.CampaignCountdownStates[dhids.CountdownID("camp-count-1")]
	if got.StartingValue != 10 || got.RemainingValue != 10 || got.Status != "active" {
		t.Fatalf("campaign countdown after resolve = %+v", got)
	}
}

func TestFoldEnvironmentEntityLifecycle(t *testing.T) {
	t.Parallel()

	state := daggerheartstate.NewSnapshotState("camp-1")
	f := &Folder{}

	err := f.foldEnvironmentEntityCreated(&state, payload.EnvironmentEntityCreatedPayload{
		EnvironmentEntityID: dhids.EnvironmentEntityID("env-1"),
		EnvironmentID:       "fog-bank",
		Name:                "Fog Bank",
		Type:                "hazard",
		Tier:                2,
		Difficulty:          14,
		SessionID:           ids.SessionID("sess-1"),
		SceneID:             ids.SceneID("scene-1"),
		Notes:               "obscures the bridge",
	})
	if err != nil {
		t.Fatalf("foldEnvironmentEntityCreated() error = %v", err)
	}

	err = f.foldEnvironmentEntityUpdated(&state, payload.EnvironmentEntityUpdatedPayload{
		EnvironmentEntityID: dhids.EnvironmentEntityID("env-1"),
		EnvironmentID:       "fog-bank",
		Name:                "Heavy Fog",
		Type:                "hazard",
		Tier:                3,
		Difficulty:          16,
		SessionID:           ids.SessionID("sess-1"),
		SceneID:             ids.SceneID("scene-2"),
		Notes:               "now covers the entire deck",
	})
	if err != nil {
		t.Fatalf("foldEnvironmentEntityUpdated() error = %v", err)
	}

	got := state.EnvironmentStates[dhids.EnvironmentEntityID("env-1")]
	if got.Name != "Heavy Fog" || got.Tier != 3 || got.SceneID != ids.SceneID("scene-2") {
		t.Fatalf("environment = %+v", got)
	}

	err = f.foldEnvironmentEntityDeleted(&state, payload.EnvironmentEntityDeletedPayload{
		EnvironmentEntityID: dhids.EnvironmentEntityID("env-1"),
	})
	if err != nil {
		t.Fatalf("foldEnvironmentEntityDeleted() error = %v", err)
	}
	if _, ok := state.EnvironmentStates[dhids.EnvironmentEntityID("env-1")]; ok {
		t.Fatal("environment entity still present after delete")
	}
}

func TestFoldRestTakenClearsSelectedRestEffects(t *testing.T) {
	t.Parallel()

	state := daggerheartstate.NewSnapshotState("camp-1")
	characterID := ids.CharacterID("char-1")
	characterState := mechanics.NewCharacterState(mechanics.CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          mechanics.HPDefault,
		HPMax:       mechanics.HPMaxDefault,
		Hope:        mechanics.HopeDefault,
		HopeMax:     mechanics.HopeMaxDefault,
		Stress:      mechanics.StressDefault,
		StressMax:   mechanics.StressMaxDefault,
		Armor:       mechanics.ArmorDefault,
		ArmorMax:    mechanics.ArmorMaxDefault,
	})
	characterState.ApplyTemporaryArmor(mechanics.TemporaryArmorBucket{Source: "spell", Duration: "short_rest", Amount: 2})
	characterState.ApplyTemporaryArmor(mechanics.TemporaryArmorBucket{Source: "ritual", Duration: "long_rest", Amount: 1})
	state.CharacterStates[characterID] = *characterState
	state.CharacterStatModifiers[characterID] = []rules.StatModifierState{
		{
			ID:            "short-rest",
			Target:        rules.StatModifierTargetEvasion,
			Delta:         1,
			ClearTriggers: []rules.ConditionClearTrigger{rules.ConditionClearTriggerShortRest},
		},
		{
			ID:            "long-rest",
			Target:        rules.StatModifierTargetMajorThreshold,
			Delta:         1,
			ClearTriggers: []rules.ConditionClearTrigger{rules.ConditionClearTriggerLongRest},
		},
	}

	f := &Folder{}
	err := f.foldRestTaken(&state, payload.RestTakenPayload{
		GMFear:      daggerheartstate.GMFearDefault + 1,
		RefreshRest: true,
		Participants: []ids.CharacterID{
			characterID,
		},
	})
	if err != nil {
		t.Fatalf("foldRestTaken() error = %v", err)
	}

	got := state.CharacterStates[characterID]
	if got.Armor != 1 {
		t.Fatalf("Armor = %d, want 1 after clearing short-rest armor", got.Armor)
	}
	if len(got.ArmorBonus) != 1 || got.ArmorBonus[0].Duration != "long_rest" {
		t.Fatalf("ArmorBonus = %+v, want only long_rest bucket", got.ArmorBonus)
	}

	modifiers := state.CharacterStatModifiers[characterID]
	if len(modifiers) != 1 || modifiers[0].ID != "long-rest" {
		t.Fatalf("CharacterStatModifiers = %+v, want only long-rest modifier", modifiers)
	}
}
