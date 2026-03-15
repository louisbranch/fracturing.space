package workflowtransport

import (
	"strings"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func TestDecodeRollSystemMetadata(t *testing.T) {
	data := map[string]any{
		KeyCharacterID: " ch-1 ",
		KeyAdversaryID: " adv-1 ",
		"trait":        " agility ",
		KeyRollKind:    pb.RollKind_ROLL_KIND_ACTION.String(),
		KeyOutcome:     pb.Outcome_SUCCESS_WITH_HOPE.String(),
		KeyHopeFear:    true,
		KeyCrit:        false,
		KeyCritNegates: true,
		"gm_move":      false,
		KeyRoll:        float64(18),
		KeyModifier:    "2",
		KeyTotal:       int64(20),
		"modifiers": []any{
			map[string]any{"value": float64(2), "source": " experience "},
			map[string]any{"value": int64(-1), "source": " penalty "},
		},
	}

	metadata, err := DecodeRollSystemMetadata(data)
	if err != nil {
		t.Fatalf("DecodeRollSystemMetadata() error = %v", err)
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
	if metadata.RollKindOrDefault() != pb.RollKind_ROLL_KIND_ACTION {
		t.Fatalf("RollKindOrDefault() = %v, want action", metadata.RollKindOrDefault())
	}
	if metadata.OutcomeOrFallback("fallback") != pb.Outcome_SUCCESS_WITH_HOPE.String() {
		t.Fatalf("OutcomeOrFallback() = %q", metadata.OutcomeOrFallback("fallback"))
	}

	if got, ok := IntValue(metadata.Roll); !ok || got != 18 {
		t.Fatalf("roll = (%d,%v), want (18,true)", got, ok)
	}
	if got, ok := IntValue(metadata.Modifier); !ok || got != 2 {
		t.Fatalf("modifier = (%d,%v), want (2,true)", got, ok)
	}
	if got, ok := IntValue(metadata.Total); !ok || got != 20 {
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
			data:    map[string]any{KeyCharacterID: 42},
			wantErr: "system_data.character_id must be string",
		},
		{
			name:    "roll wrong type",
			data:    map[string]any{KeyRoll: "nope"},
			wantErr: "system_data.roll must be integer",
		},
		{
			name:    "roll non-integer float",
			data:    map[string]any{KeyRoll: 4.5},
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
			_, err := DecodeRollSystemMetadata(tc.data)
			if err == nil {
				t.Fatalf("DecodeRollSystemMetadata() error = nil, want contains %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("DecodeRollSystemMetadata() error = %q, want contains %q", err.Error(), tc.wantErr)
			}
		})
	}
}
