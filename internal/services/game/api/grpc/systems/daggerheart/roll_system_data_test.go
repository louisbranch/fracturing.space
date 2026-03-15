package daggerheart

import (
	"strings"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
)

func TestDecodeRollSystemMetadata(t *testing.T) {
	data := map[string]any{
		workflowtransport.KeyCharacterID: " ch-1 ",
		workflowtransport.KeyAdversaryID: " adv-1 ",
		"trait":                          " agility ",
		workflowtransport.KeyRollKind:    pb.RollKind_ROLL_KIND_ACTION.String(),
		workflowtransport.KeyOutcome:     pb.Outcome_SUCCESS_WITH_HOPE.String(),
		workflowtransport.KeyHopeFear:    true,
		workflowtransport.KeyCrit:        false,
		workflowtransport.KeyCritNegates: true,
		"gm_move":                        false,
		workflowtransport.KeyRoll:        float64(18),
		workflowtransport.KeyModifier:    "2",
		workflowtransport.KeyTotal:       int64(20),
		"modifiers": []any{
			map[string]any{"value": float64(2), "source": " experience "},
			map[string]any{"value": int64(-1), "source": " penalty "},
		},
	}

	metadata, err := workflowtransport.DecodeRollSystemMetadata(data)
	if err != nil {
		t.Fatalf("workflowtransport.DecodeRollSystemMetadata() error = %v", err)
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

	if got, ok := workflowtransport.IntValue(metadata.Roll); !ok || got != 18 {
		t.Fatalf("roll = (%d,%v), want (18,true)", got, ok)
	}
	if got, ok := workflowtransport.IntValue(metadata.Modifier); !ok || got != 2 {
		t.Fatalf("modifier = (%d,%v), want (2,true)", got, ok)
	}
	if got, ok := workflowtransport.IntValue(metadata.Total); !ok || got != 20 {
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
			data:    map[string]any{workflowtransport.KeyCharacterID: 42},
			wantErr: "system_data.character_id must be string",
		},
		{
			name:    "roll wrong type",
			data:    map[string]any{workflowtransport.KeyRoll: "nope"},
			wantErr: "system_data.roll must be integer",
		},
		{
			name:    "roll non-integer float",
			data:    map[string]any{workflowtransport.KeyRoll: 4.5},
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
			_, err := workflowtransport.DecodeRollSystemMetadata(tc.data)
			if err == nil {
				t.Fatalf("workflowtransport.DecodeRollSystemMetadata() error = nil, want contains %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("workflowtransport.DecodeRollSystemMetadata() error = %q, want contains %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestRollSystemDataHelpers(t *testing.T) {
	metadata := workflowtransport.RollSystemMetadata{RollKind: "adversary_roll", Outcome: " ", HopeFear: workflowtransport.BoolPtr(false), Crit: nil}

	if metadata.RollKindCode() != "adversary_roll" {
		t.Fatalf("RollKindCode() = %q", metadata.RollKindCode())
	}
	if metadata.RollKindOrDefault() != pb.RollKind_ROLL_KIND_ACTION {
		t.Fatalf("RollKindOrDefault() = %v, want action", metadata.RollKindOrDefault())
	}
	if metadata.OutcomeOrFallback("fallback") != "fallback" {
		t.Fatalf("OutcomeOrFallback() = %q, want fallback", metadata.OutcomeOrFallback("fallback"))
	}
	if got := workflowtransport.BoolValue(metadata.HopeFear, true); got {
		t.Fatalf("workflowtransport.BoolValue() = %v, want false", got)
	}
	if got := workflowtransport.BoolValue(metadata.Crit, true); !got {
		t.Fatalf("workflowtransport.BoolValue() = %v, want true", got)
	}
}
