package daggerheart

import (
	"strings"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func TestDecodeRollSystemMetadata(t *testing.T) {
	data := map[string]any{
		sdKeyCharacterID: " ch-1 ",
		sdKeyAdversaryID: " adv-1 ",
		"trait":          " agility ",
		sdKeyRollKind:    pb.RollKind_ROLL_KIND_ACTION.String(),
		sdKeyOutcome:     pb.Outcome_SUCCESS_WITH_HOPE.String(),
		sdKeyHopeFear:    true,
		sdKeyCrit:        false,
		sdKeyCritNegates: true,
		"gm_move":        false,
		sdKeyRoll:        float64(18),
		sdKeyModifier:    "2",
		sdKeyTotal:       int64(20),
		"modifiers": []any{
			map[string]any{"value": float64(2), "source": " experience "},
			map[string]any{"value": int64(-1), "source": " penalty "},
		},
	}

	metadata, err := decodeRollSystemMetadata(data)
	if err != nil {
		t.Fatalf("decodeRollSystemMetadata() error = %v", err)
	}

	if metadata.CharacterID != "ch-1" {
		t.Fatalf("CharacterID = %q, want ch-1", metadata.CharacterID)
	}
	if metadata.AdversaryID != "adv-1" {
		t.Fatalf("AdversaryID = %q, want adv-1", metadata.AdversaryID)
	}
	if metadata.Trait != "agility" {
		t.Fatalf("Trait = %q, want agility", metadata.Trait)
	}
	if metadata.rollKindOrDefault() != pb.RollKind_ROLL_KIND_ACTION {
		t.Fatalf("rollKindOrDefault() = %v, want action", metadata.rollKindOrDefault())
	}
	if metadata.outcomeOrFallback("fallback") != pb.Outcome_SUCCESS_WITH_HOPE.String() {
		t.Fatalf("outcomeOrFallback() = %q", metadata.outcomeOrFallback("fallback"))
	}

	if got, ok := intPointerValue(metadata.Roll); !ok || got != 18 {
		t.Fatalf("roll = (%d,%v), want (18,true)", got, ok)
	}
	if got, ok := intPointerValue(metadata.Modifier); !ok || got != 2 {
		t.Fatalf("modifier = (%d,%v), want (2,true)", got, ok)
	}
	if got, ok := intPointerValue(metadata.Total); !ok || got != 20 {
		t.Fatalf("total = (%d,%v), want (20,true)", got, ok)
	}

	if len(metadata.Modifiers) != 2 {
		t.Fatalf("len(modifiers) = %d, want 2", len(metadata.Modifiers))
	}
	if metadata.Modifiers[0].Value != 2 || metadata.Modifiers[0].Source != "experience" {
		t.Fatalf("modifier[0] = %+v", metadata.Modifiers[0])
	}
	if metadata.Modifiers[1].Value != -1 || metadata.Modifiers[1].Source != "penalty" {
		t.Fatalf("modifier[1] = %+v", metadata.Modifiers[1])
	}
}

func TestDecodeRollSystemMetadataInvalid(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]any
		wantErr string
	}{
		{
			name:    "character id wrong type",
			data:    map[string]any{sdKeyCharacterID: 42},
			wantErr: "system_data.character_id must be string",
		},
		{
			name:    "roll wrong type",
			data:    map[string]any{sdKeyRoll: "nope"},
			wantErr: "system_data.roll must be integer",
		},
		{
			name:    "roll non-integer float",
			data:    map[string]any{sdKeyRoll: 4.5},
			wantErr: "system_data.roll must be integer",
		},
		{
			name:    "modifier list wrong type",
			data:    map[string]any{"modifiers": "bad"},
			wantErr: "system_data.modifiers must be array",
		},
		{
			name:    "modifier missing value",
			data:    map[string]any{"modifiers": []any{map[string]any{"source": "xp"}}},
			wantErr: "system_data.modifiers[0].value is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodeRollSystemMetadata(tc.data)
			if err == nil {
				t.Fatalf("decodeRollSystemMetadata() error = nil, want contains %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("decodeRollSystemMetadata() error = %q, want contains %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestRollSystemDataHelpers(t *testing.T) {
	metadata := rollSystemMetadata{RollKind: "adversary_roll", Outcome: " ", HopeFear: boolPtr(false), Crit: nil}

	if metadata.rollKindCode() != "adversary_roll" {
		t.Fatalf("rollKindCode() = %q", metadata.rollKindCode())
	}
	if metadata.rollKindOrDefault() != pb.RollKind_ROLL_KIND_ACTION {
		t.Fatalf("rollKindOrDefault() = %v, want action", metadata.rollKindOrDefault())
	}
	if metadata.outcomeOrFallback("fallback") != "fallback" {
		t.Fatalf("outcomeOrFallback() = %q, want fallback", metadata.outcomeOrFallback("fallback"))
	}
	if got := boolPointerValue(metadata.HopeFear, true); got {
		t.Fatalf("boolPointerValue() = %v, want false", got)
	}
	if got := boolPointerValue(metadata.Crit, true); !got {
		t.Fatalf("boolPointerValue() = %v, want true", got)
	}
}
