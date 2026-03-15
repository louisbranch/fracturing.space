package workflowtransport

import "testing"

func TestRollSystemMetadataMapValue(t *testing.T) {
	metadata := RollSystemMetadata{
		CharacterID:       " char-1 ",
		AdversaryID:       " adv-1 ",
		Trait:             " agility ",
		RollKind:          " action ",
		Outcome:           " success ",
		Flavor:            " bold ",
		BreathCountdownID: " countdown-1 ",
		HopeFear:          BoolPtr(true),
		Crit:              BoolPtr(false),
		CritNegates:       BoolPtr(true),
		GMMove:            BoolPtr(false),
		Underwater:        BoolPtr(true),
		Critical:          BoolPtr(false),
		Roll:              IntPtr(12),
		Modifier:          IntPtr(3),
		Total:             IntPtr(15),
		BaseTotal:         IntPtr(14),
		CriticalBonus:     IntPtr(1),
		Advantage:         IntPtr(2),
		Disadvantage:      IntPtr(0),
		Modifiers:         []RollModifierMetadata{{Value: 2, Source: "xp"}},
	}

	data := metadata.MapValue()

	if data[KeyCharacterID] != "char-1" || data[KeyAdversaryID] != "adv-1" {
		t.Fatalf("identity fields mismatch: %+v", data)
	}
	if data["trait"] != "agility" || data[KeyRollKind] != "action" || data[KeyOutcome] != "success" {
		t.Fatalf("roll fields mismatch: %+v", data)
	}
	if data["flavor"] != "bold" || data["breath_countdown_id"] != "countdown-1" {
		t.Fatalf("string fields mismatch: %+v", data)
	}
	if data[KeyHopeFear] != true || data[KeyCrit] != false || data["critical"] != false {
		t.Fatalf("bool fields mismatch: %+v", data)
	}
	if data[KeyRoll] != 12 || data[KeyModifier] != 3 || data[KeyTotal] != 15 {
		t.Fatalf("int fields mismatch: %+v", data)
	}
	modifiers, ok := data["modifiers"].([]RollModifierMetadata)
	if !ok || len(modifiers) != 1 || modifiers[0].Source != "xp" {
		t.Fatalf("modifier list mismatch: %#v", data["modifiers"])
	}
}

func TestRollSystemDataHelpers(t *testing.T) {
	metadata := RollSystemMetadata{RollKind: "adversary_roll", Outcome: " ", HopeFear: BoolPtr(false), Crit: nil}

	if metadata.RollKindCode() != "adversary_roll" {
		t.Fatalf("RollKindCode() = %q", metadata.RollKindCode())
	}
	if metadata.OutcomeOrFallback("fallback") != "fallback" {
		t.Fatalf("OutcomeOrFallback() = %q, want fallback", metadata.OutcomeOrFallback("fallback"))
	}
	if got := BoolValue(metadata.HopeFear, true); got {
		t.Fatalf("BoolValue() = %v, want false", got)
	}
	if got := BoolValue(metadata.Crit, true); !got {
		t.Fatalf("BoolValue() = %v, want true", got)
	}
	if got, ok := IntValue(nil); ok || got != 0 {
		t.Fatalf("IntValue(nil) = (%d,%v), want (0,false)", got, ok)
	}
}
