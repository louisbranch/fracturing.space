package rules

import "testing"

func TestNormalizeGMMoveHelpers(t *testing.T) {
	t.Parallel()

	kind, ok := NormalizeGMMoveKind(" additional_move ")
	if !ok || kind != GMMoveKindAdditionalMove {
		t.Fatalf("NormalizeGMMoveKind() = (%q, %t), want (%q, true)", kind, ok, GMMoveKindAdditionalMove)
	}
	if kind, ok := NormalizeGMMoveKind("mystery"); ok || kind != GMMoveKindUnspecified {
		t.Fatalf("NormalizeGMMoveKind(invalid) = (%q, %t), want unspecified false", kind, ok)
	}

	shape, ok := NormalizeGMMoveShape(" Shift_Environment ")
	if !ok || shape != GMMoveShapeShiftEnvironment {
		t.Fatalf("NormalizeGMMoveShape() = (%q, %t), want (%q, true)", shape, ok, GMMoveShapeShiftEnvironment)
	}
	if shape, ok := NormalizeGMMoveShape("mystery"); ok || shape != GMMoveShapeUnspecified {
		t.Fatalf("NormalizeGMMoveShape(invalid) = (%q, %t), want unspecified false", shape, ok)
	}

	targetType, ok := NormalizeGMMoveTargetType(" adversary_experience ")
	if !ok || targetType != GMMoveTargetTypeAdversaryExperience {
		t.Fatalf("NormalizeGMMoveTargetType() = (%q, %t), want (%q, true)", targetType, ok, GMMoveTargetTypeAdversaryExperience)
	}
	if targetType, ok := NormalizeGMMoveTargetType("mystery"); ok || targetType != GMMoveTargetTypeUnspecified {
		t.Fatalf("NormalizeGMMoveTargetType(invalid) = (%q, %t), want unspecified false", targetType, ok)
	}
}

func TestStatModifierHelpers(t *testing.T) {
	t.Parallel()

	if !ValidStatModifierTarget(" Agility ") {
		t.Fatal("expected agility to be a valid stat modifier target")
	}
	if ValidStatModifierTarget("luck") {
		t.Fatal("expected luck to be rejected")
	}

	raw := []StatModifierState{
		{
			ID:            "  mod-2  ",
			Target:        StatModifierTargetPresence,
			Delta:         1,
			Label:         "  Presence boost  ",
			Source:        "  spell  ",
			SourceID:      "  spell-2  ",
			ClearTriggers: []ConditionClearTrigger{ConditionClearTriggerSessionEnd, ConditionClearTriggerShortRest, ConditionClearTriggerShortRest},
		},
		{
			ID:       "mod-1",
			Target:   StatModifierTargetAgility,
			Delta:    2,
			Label:    "Quickstep",
			Source:   "feature",
			SourceID: "feature-1",
		},
		{
			ID:       "mod-1",
			Target:   StatModifierTargetAgility,
			Delta:    3,
			Label:    "duplicate ignored",
			Source:   "feature",
			SourceID: "feature-dup",
		},
	}
	normalized, err := NormalizeStatModifiers(raw)
	if err != nil {
		t.Fatalf("NormalizeStatModifiers() error = %v", err)
	}
	if len(normalized) != 2 {
		t.Fatalf("len(normalized) = %d, want 2", len(normalized))
	}
	if normalized[0].ID != "mod-1" || normalized[1].ID != "mod-2" {
		t.Fatalf("normalized ids = %#v, want mod-1 then mod-2", normalized)
	}
	if normalized[1].Label != "Presence boost" || normalized[1].Source != "spell" || normalized[1].SourceID != "spell-2" {
		t.Fatalf("normalized metadata = %#v, want trimmed values", normalized[1])
	}
	if len(normalized[1].ClearTriggers) != 2 {
		t.Fatalf("clear trigger len = %d, want 2", len(normalized[1].ClearTriggers))
	}

	same := append([]StatModifierState(nil), normalized...)
	if !StatModifiersEqual(normalized, same) {
		t.Fatal("expected normalized stat modifiers to compare equal")
	}
	if StatModifiersEqual(normalized, normalized[:1]) {
		t.Fatal("expected different-length stat modifier slices to differ")
	}

	added, removed := DiffStatModifiers(
		[]StatModifierState{normalized[0]},
		[]StatModifierState{normalized[0], normalized[1]},
	)
	if len(added) != 1 || added[0].ID != "mod-2" {
		t.Fatalf("added = %#v, want mod-2", added)
	}
	if len(removed) != 0 {
		t.Fatalf("removed = %#v, want empty", removed)
	}

	remaining, cleared := ClearStatModifiersByTrigger(normalized, ConditionClearTriggerShortRest)
	if len(remaining) != 1 || remaining[0].ID != "mod-1" {
		t.Fatalf("remaining = %#v, want mod-1 only", remaining)
	}
	if len(cleared) != 1 || cleared[0].ID != "mod-2" {
		t.Fatalf("cleared = %#v, want mod-2", cleared)
	}

	if _, err := NormalizeStatModifiers([]StatModifierState{{ID: "bad", Target: "luck"}}); err == nil {
		t.Fatal("expected unsupported target error")
	}
	if _, err := NormalizeStatModifiers([]StatModifierState{{Target: StatModifierTargetAgility}}); err == nil {
		t.Fatal("expected missing id error")
	}
}
