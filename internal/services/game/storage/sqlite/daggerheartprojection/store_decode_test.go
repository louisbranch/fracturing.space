package daggerheartprojection

import (
	"encoding/json"
	"testing"
	"time"

	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

func TestNormalizeLegacyProjectionConditionStatesJSON(t *testing.T) {
	normalized, changed, err := normalizeLegacyProjectionConditionStatesJSON(`["hidden"]`)
	if err != nil {
		t.Fatalf("normalize legacy conditions: %v", err)
	}
	if !changed {
		t.Fatal("expected legacy conditions to be rewritten")
	}

	var structured []map[string]any
	if err := json.Unmarshal([]byte(normalized), &structured); err != nil {
		t.Fatalf("unmarshal normalized conditions: %v", err)
	}
	if len(structured) != 1 {
		t.Fatalf("expected 1 normalized condition, got %d", len(structured))
	}
	if structured[0]["Code"] != "hidden" || structured[0]["Class"] != "standard" {
		t.Fatalf("expected hidden standard condition, got %+v", structured[0])
	}
}

func TestDBDaggerheartAdversaryToDomainDecodesStructuredConditions(t *testing.T) {
	row := db.DaggerheartAdversary{
		CampaignID:            "camp-1",
		AdversaryID:           "adv-1",
		AdversaryEntryID:      "entry-1",
		Name:                  "Shadow Drake",
		Kind:                  "solo",
		SessionID:             "sess-1",
		SceneID:               "scene-1",
		ConditionsJson:        `[{"ID":"hidden","Class":"standard","Standard":"hidden","Code":"hidden","Label":"Hidden","Source":"","SourceID":"","ClearTriggers":[]}]`,
		FeatureStateJson:      `[]`,
		PendingExperienceJson: "",
		CreatedAt:             toMillis(time.Date(2026, 2, 3, 11, 0, 0, 0, time.UTC)),
		UpdatedAt:             toMillis(time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)),
	}

	adversary, err := dbDaggerheartAdversaryToDomain(row)
	if err != nil {
		t.Fatalf("decode adversary: %v", err)
	}

	if len(adversary.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(adversary.Conditions))
	}
	if adversary.Conditions[0].Code != "hidden" || adversary.Conditions[0].Class != "standard" {
		t.Fatalf("expected hidden standard condition, got %+v", adversary.Conditions[0])
	}
	if len(adversary.FeatureStates) != 0 {
		t.Fatalf("expected empty feature states, got %v", adversary.FeatureStates)
	}
	if adversary.PendingExperience != nil {
		t.Fatalf("expected nil pending experience, got %+v", adversary.PendingExperience)
	}
}

func TestDBDaggerheartCharacterStateToDomainDefaultsOptionalState(t *testing.T) {
	row := db.DaggerheartCharacterState{
		CampaignID:         "camp-1",
		CharacterID:        "char-1",
		ConditionsJson:     `[{"ID":"restrained","Class":"standard","Standard":"restrained","Code":"restrained","Label":"Restrained","Source":"","SourceID":"","ClearTriggers":[]}]`,
		TemporaryArmorJson: `[]`,
		ClassStateJson:     `{"AttackBonusUntilRest":2}`,
		SubclassStateJson:  `{}`,
		CompanionStateJson: `null`,
		LifeState:          "",
		StatModifiersJson:  `[]`,
	}

	state, err := dbDaggerheartCharacterStateToDomain(row)
	if err != nil {
		t.Fatalf("decode character state: %v", err)
	}

	if state.LifeState != daggerheartstate.LifeStateAlive {
		t.Fatalf("expected default life state %q, got %q", daggerheartstate.LifeStateAlive, state.LifeState)
	}
	if len(state.Conditions) != 1 || state.Conditions[0].Code != "restrained" {
		t.Fatalf("expected restrained condition, got %v", state.Conditions)
	}
	if len(state.TemporaryArmor) != 0 {
		t.Fatalf("expected empty temporary armor, got %v", state.TemporaryArmor)
	}
	if state.SubclassState != nil {
		t.Fatalf("expected nil subclass state, got %+v", state.SubclassState)
	}
	if state.CompanionState != nil {
		t.Fatalf("expected nil companion state, got %+v", state.CompanionState)
	}
	if state.ClassState.AttackBonusUntilRest != 2 {
		t.Fatalf("expected decoded class state attack bonus 2, got %+v", state.ClassState)
	}
	if state.StatModifiers != nil {
		t.Fatalf("expected nil stat modifiers for empty array, got %v", state.StatModifiers)
	}
}
