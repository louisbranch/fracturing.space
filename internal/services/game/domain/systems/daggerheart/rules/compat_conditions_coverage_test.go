package rules

import "testing"

func TestWithConditionSource(t *testing.T) {
	state, err := StandardConditionState(ConditionHidden, WithConditionSource("spell", "spell-1"))
	if err != nil {
		t.Fatalf("StandardConditionState: %v", err)
	}
	if state.Source != "spell" {
		t.Fatalf("source = %q, want %q", state.Source, "spell")
	}
	if state.SourceID != "spell-1" {
		t.Fatalf("source_id = %q, want %q", state.SourceID, "spell-1")
	}
}

func TestWithConditionSource_NilState(t *testing.T) {
	// WithConditionSource should not panic on nil state.
	fn := WithConditionSource("spell", "spell-1")
	fn(nil) // should be a no-op
}

func TestWithConditionClearTriggers(t *testing.T) {
	state, err := StandardConditionState(ConditionRestrained,
		WithConditionClearTriggers(ConditionClearTriggerShortRest, ConditionClearTriggerDamageTaken))
	if err != nil {
		t.Fatalf("StandardConditionState: %v", err)
	}
	if len(state.ClearTriggers) != 2 {
		t.Fatalf("triggers len = %d, want 2", len(state.ClearTriggers))
	}
}

func TestWithConditionClearTriggers_NilState(t *testing.T) {
	fn := WithConditionClearTriggers(ConditionClearTriggerShortRest)
	fn(nil) // should be a no-op
}

func TestClearConditionStatesByTrigger(t *testing.T) {
	states := []ConditionState{
		{ID: "hidden", Class: ConditionClassStandard, Standard: "hidden", Code: "hidden", Label: "Hidden",
			ClearTriggers: []ConditionClearTrigger{ConditionClearTriggerShortRest}},
		{ID: "restrained", Class: ConditionClassStandard, Standard: "restrained", Code: "restrained", Label: "Restrained"},
	}

	remaining, removed := ClearConditionStatesByTrigger(states, ConditionClearTriggerShortRest)
	if len(remaining) != 1 || remaining[0].ID != "restrained" {
		t.Fatalf("remaining = %v, want [restrained]", remaining)
	}
	if len(removed) != 1 || removed[0].ID != "hidden" {
		t.Fatalf("removed = %v, want [hidden]", removed)
	}

	// Empty trigger returns all.
	remaining, removed = ClearConditionStatesByTrigger(states, "")
	if len(remaining) != 2 {
		t.Fatalf("empty trigger remaining = %d, want 2", len(remaining))
	}
	if len(removed) != 0 {
		t.Fatalf("empty trigger removed = %d, want 0", len(removed))
	}

	// Empty values returns empty.
	remaining, removed = ClearConditionStatesByTrigger(nil, ConditionClearTriggerShortRest)
	if len(remaining) != 0 || len(removed) != 0 {
		t.Fatalf("nil values: remaining=%d, removed=%d, want 0, 0", len(remaining), len(removed))
	}
}

func TestHasConditionCode(t *testing.T) {
	states := []ConditionState{
		{ID: "hidden", Class: ConditionClassStandard, Standard: "hidden", Code: "hidden", Label: "Hidden"},
		{ID: "custom-1", Class: ConditionClassSpecial, Code: "burning", Label: "Burning"},
	}

	if !HasConditionCode(states, "Hidden") {
		t.Fatal("expected HasConditionCode(Hidden) = true")
	}
	if !HasConditionCode(states, "burning") {
		t.Fatal("expected HasConditionCode(burning) = true")
	}
	if HasConditionCode(states, "frozen") {
		t.Fatal("expected HasConditionCode(frozen) = false")
	}
	if HasConditionCode(states, "") {
		t.Fatal("expected HasConditionCode('') = false")
	}
	if HasConditionCode(states, "  ") {
		t.Fatal("expected HasConditionCode('  ') = false")
	}
}

func TestRemoveConditionCode(t *testing.T) {
	states := []ConditionState{
		{ID: "hidden", Class: ConditionClassStandard, Standard: "hidden", Code: "hidden", Label: "Hidden"},
		{ID: "custom-1", Class: ConditionClassSpecial, Code: "burning", Label: "Burning"},
	}

	got := RemoveConditionCode(states, "Hidden")
	if len(got) != 1 || got[0].Code != "burning" {
		t.Fatalf("RemoveConditionCode(Hidden) = %v, want [burning]", got)
	}

	// Empty code returns copy.
	got = RemoveConditionCode(states, "")
	if len(got) != 2 {
		t.Fatalf("RemoveConditionCode('') = %d, want 2", len(got))
	}

	// Non-existent code returns all.
	got = RemoveConditionCode(states, "frozen")
	if len(got) != 2 {
		t.Fatalf("RemoveConditionCode(frozen) = %d, want 2", len(got))
	}
}

func TestNormalizeConditionClearTriggers(t *testing.T) {
	// nil returns nil.
	if got := normalizeConditionClearTriggers(nil); got != nil {
		t.Fatalf("normalizeConditionClearTriggers(nil) = %v, want nil", got)
	}

	// Deduplication and filtering.
	triggers := []ConditionClearTrigger{
		ConditionClearTriggerShortRest,
		ConditionClearTriggerShortRest, // duplicate
		"invalid_trigger",
		ConditionClearTriggerLongRest,
		ConditionClearTriggerDamageTaken,
		ConditionClearTriggerSessionEnd,
	}
	got := normalizeConditionClearTriggers(triggers)
	if len(got) != 4 {
		t.Fatalf("len = %d, want 4", len(got))
	}
	// Should be sorted.
	for i := 1; i < len(got); i++ {
		if got[i] < got[i-1] {
			t.Fatalf("triggers not sorted: %v", got)
		}
	}
}

func TestNormalizeConditionState_SpecialClass(t *testing.T) {
	state := ConditionState{
		ID:    "custom-1",
		Class: ConditionClassSpecial,
		Code:  "burning",
		Label: "Burning",
	}
	got, err := normalizeConditionState(state)
	if err != nil {
		t.Fatalf("normalizeConditionState: %v", err)
	}
	if got.Standard != "" {
		t.Fatalf("special condition standard = %q, want empty", got.Standard)
	}
	if got.Code != "burning" {
		t.Fatalf("code = %q, want %q", got.Code, "burning")
	}
}

func TestNormalizeConditionState_TagClass(t *testing.T) {
	state := ConditionState{
		ID:    "tag-1",
		Class: ConditionClassTag,
		Label: "Marked",
	}
	got, err := normalizeConditionState(state)
	if err != nil {
		t.Fatalf("normalizeConditionState: %v", err)
	}
	if got.Code != "marked" {
		t.Fatalf("tag code = %q, want %q", got.Code, "marked")
	}
}

func TestNormalizeConditionState_InferClassFromFields(t *testing.T) {
	// Standard inferred from non-empty Standard field.
	state := ConditionState{ID: "hidden", Standard: "hidden"}
	got, err := normalizeConditionState(state)
	if err != nil {
		t.Fatalf("normalizeConditionState: %v", err)
	}
	if got.Class != ConditionClassStandard {
		t.Fatalf("class = %q, want %q", got.Class, ConditionClassStandard)
	}

	// Special inferred from non-empty Code field.
	state = ConditionState{ID: "burn-1", Code: "burning"}
	got, err = normalizeConditionState(state)
	if err != nil {
		t.Fatalf("normalizeConditionState: %v", err)
	}
	if got.Class != ConditionClassSpecial {
		t.Fatalf("class = %q, want %q", got.Class, ConditionClassSpecial)
	}
}

func TestNormalizeConditionState_Errors(t *testing.T) {
	// Invalid class.
	_, err := normalizeConditionState(ConditionState{ID: "x", Class: "invalid"})
	if err == nil {
		t.Fatal("expected error for invalid class")
	}

	// Standard class with unknown code.
	_, err = normalizeConditionState(ConditionState{ID: "x", Class: ConditionClassStandard, Standard: "unknown"})
	if err == nil {
		t.Fatal("expected error for unknown standard code")
	}

	// Special class with empty code and label.
	_, err = normalizeConditionState(ConditionState{ID: "", Class: ConditionClassSpecial})
	if err == nil {
		t.Fatal("expected error for special condition missing code")
	}
}

func TestConditionHasTrigger(t *testing.T) {
	state := ConditionState{
		ClearTriggers: []ConditionClearTrigger{ConditionClearTriggerShortRest, ConditionClearTriggerDamageTaken},
	}
	if !conditionHasTrigger(state, ConditionClearTriggerShortRest) {
		t.Fatal("expected short_rest trigger to be present")
	}
	if conditionHasTrigger(state, ConditionClearTriggerLongRest) {
		t.Fatal("expected long_rest trigger to be absent")
	}
	// No triggers.
	if conditionHasTrigger(ConditionState{}, ConditionClearTriggerShortRest) {
		t.Fatal("expected no triggers to return false")
	}
}

func TestNormalizeConditionState_FillsIDFromCode(t *testing.T) {
	state := ConditionState{
		Class: ConditionClassStandard,
		Code:  "hidden",
	}
	got, err := normalizeConditionState(state)
	if err != nil {
		t.Fatalf("normalizeConditionState: %v", err)
	}
	if got.ID != "hidden" {
		t.Fatalf("ID = %q, want %q", got.ID, "hidden")
	}
}

func TestNormalizeConditionState_SpecialFillsLabelFromCode(t *testing.T) {
	state := ConditionState{
		ID:    "x",
		Class: ConditionClassSpecial,
		Code:  "burning",
	}
	got, err := normalizeConditionState(state)
	if err != nil {
		t.Fatalf("normalizeConditionState: %v", err)
	}
	if got.Label != "burning" {
		t.Fatalf("label = %q, want %q", got.Label, "burning")
	}
}
