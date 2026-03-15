package creationworkflow

import (
	"testing"

	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
)

func TestProgressFromDaggerheart(t *testing.T) {
	dhProgress := daggerheart.CreationProgress{
		Steps: []daggerheart.CreationStepProgress{
			{Step: 1, Key: "class_subclass", Complete: true},
			{Step: 2, Key: "heritage", Complete: false},
		},
		NextStep:     2,
		Ready:        false,
		UnmetReasons: []string{"heritage required"},
	}

	result := progressFromDaggerheart(dhProgress)

	if len(result.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(result.Steps))
	}
	if result.Steps[0].Key != "class_subclass" || !result.Steps[0].Complete {
		t.Fatalf("step 0 = %+v, want class_subclass/complete", result.Steps[0])
	}
	if result.NextStep != 2 {
		t.Fatalf("NextStep = %d, want 2", result.NextStep)
	}
	if result.Ready {
		t.Fatal("Ready = true, want false")
	}
	if len(result.UnmetReasons) != 1 || result.UnmetReasons[0] != "heritage required" {
		t.Fatalf("UnmetReasons = %v, want [heritage required]", result.UnmetReasons)
	}
}

func TestProgressFromDaggerheart_Empty(t *testing.T) {
	result := progressFromDaggerheart(daggerheart.CreationProgress{})

	if len(result.Steps) != 0 {
		t.Fatalf("steps = %d, want 0", len(result.Steps))
	}
	if result.Ready {
		t.Fatal("Ready = true, want false")
	}
}

func TestProgressFromDaggerheart_UnmetReasonsIsolated(t *testing.T) {
	reasons := []string{"a", "b"}
	dhProgress := daggerheart.CreationProgress{UnmetReasons: reasons}

	result := progressFromDaggerheart(dhProgress)

	reasons[0] = "mutated"
	if result.UnmetReasons[0] != "a" {
		t.Fatal("UnmetReasons not copied; mutation leaked")
	}
}
