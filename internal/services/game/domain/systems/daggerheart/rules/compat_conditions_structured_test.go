package rules

import (
	"encoding/json"
	"testing"
)

func TestConditionStateUnmarshalJSONRequiresStructuredPayloads(t *testing.T) {
	t.Parallel()

	var legacyRejected ConditionState
	if err := json.Unmarshal([]byte(`"hidden"`), &legacyRejected); err == nil {
		t.Fatal("expected legacy string payload to be rejected")
	}

	var structured ConditionState
	if err := json.Unmarshal([]byte(`{"id":"tag-1","class":"tag","label":"Marked"}`), &structured); err != nil {
		t.Fatalf("structured unmarshal: %v", err)
	}
	if structured.ID != "tag-1" || structured.Class != ConditionClassTag || structured.Label != "Marked" {
		t.Fatalf("structured condition = %#v, want tag payload", structured)
	}

	if err := (*ConditionState)(nil).UnmarshalJSON([]byte(`"hidden"`)); err == nil {
		t.Fatal("expected nil receiver error")
	}
}

func TestConditionStateCollectionHelpers(t *testing.T) {
	t.Parallel()

	normalized, err := NormalizeConditionStates([]ConditionState{
		{ID: " custom-1 ", Class: ConditionClassSpecial, Code: " burning ", Label: " Burning "},
		{ID: "vulnerable", Class: ConditionClassStandard, Standard: ConditionVulnerable},
		{ID: "hidden", Class: ConditionClassStandard, Standard: ConditionHidden},
		{ID: "hidden-duplicate", Class: ConditionClassStandard, Standard: ConditionHidden},
	})
	if err != nil {
		t.Fatalf("NormalizeConditionStates() error = %v", err)
	}
	if len(normalized) != 3 {
		t.Fatalf("len(normalized) = %d, want 3", len(normalized))
	}
	if normalized[0].Code != "burning" || normalized[1].Code != ConditionHidden || normalized[2].Code != ConditionVulnerable {
		t.Fatalf("normalized order = %#v, want class/code ordering", normalized)
	}

	left := []ConditionState{
		normalized[0],
		normalized[2],
	}
	right := []ConditionState{
		normalized[0],
		normalized[2],
	}
	if !ConditionStatesEqual(left, right) {
		t.Fatal("expected equal structured condition slices")
	}
	if ConditionStatesEqual(left, normalized) {
		t.Fatal("expected different-length structured condition slices to differ")
	}

	added, removed := DiffConditionStates(
		[]ConditionState{normalized[0]},
		[]ConditionState{normalized[0], normalized[2]},
	)
	if len(added) != 1 || added[0].ID != normalized[2].ID {
		t.Fatalf("added = %#v, want custom condition", added)
	}
	if len(removed) != 0 {
		t.Fatalf("removed = %#v, want empty", removed)
	}

	codes := ConditionCodes([]ConditionState{
		normalized[2],
		normalized[0],
		normalized[1],
	})
	if len(codes) != 3 || codes[0] != ConditionVulnerable || codes[1] != "burning" || codes[2] != ConditionHidden {
		t.Fatalf("codes = %#v, want input-order condition codes", codes)
	}
}
