package snapshottransport

import (
	"testing"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
)

func TestBuildDaggerheartCharacterStatePatch_DefaultsAndConditionNormalization(t *testing.T) {
	current := projectionstore.DaggerheartCharacterState{HopeMax: 0}
	profile := projectionstore.DaggerheartCharacterProfile{ArmorMax: -1}

	patch, err := buildDaggerheartCharacterStatePatch(current, profile, &daggerheartv1.DaggerheartCharacterState{
		Hp:         6,
		Hope:       0,
		Stress:     6,
		Armor:      0,
		Conditions: []daggerheartv1.DaggerheartCondition{daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE, daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN},
	})
	if err != nil {
		t.Fatalf("build patch: %v", err)
	}
	if patch.hopeMax != daggerheart.HopeMax {
		t.Fatalf("hopeMax = %d, want %d", patch.hopeMax, daggerheart.HopeMax)
	}
	if patch.stressMax != 6 {
		t.Fatalf("stressMax = %d, want %d", patch.stressMax, 6)
	}
	if patch.lifeState != daggerheart.LifeStateAlive {
		t.Fatalf("lifeState = %q, want %q", patch.lifeState, daggerheart.LifeStateAlive)
	}
	if !patch.conditionPatch {
		t.Fatal("conditionPatch = false, want true")
	}
	if len(patch.normalizedConditions) != 2 || patch.normalizedConditions[0] != "hidden" || patch.normalizedConditions[1] != "vulnerable" {
		t.Fatalf("normalizedConditions = %v, want [hidden vulnerable]", patch.normalizedConditions)
	}
}

func TestDaggerheartCharacterStatePatchStateUnchanged_DefaultsEmptyLifeStateToAlive(t *testing.T) {
	current := projectionstore.DaggerheartCharacterState{
		Hp:      10,
		Hope:    4,
		HopeMax: 6,
		Stress:  2,
		Armor:   1,
	}
	patch := daggerheartCharacterStatePatch{
		hp:        10,
		hope:      4,
		hopeMax:   6,
		stress:    2,
		armor:     1,
		lifeState: daggerheart.LifeStateAlive,
	}
	if !patch.stateUnchanged(current) {
		t.Fatal("stateUnchanged = false, want true")
	}
}
