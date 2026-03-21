package statetransport

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func TestCharacterStateToProto(t *testing.T) {
	state := projectionstore.DaggerheartCharacterState{
		Hp:      5,
		Hope:    2,
		HopeMax: 6,
		Stress:  3,
		Armor:   1,
		Conditions: standardProjectionConditions(
			rules.ConditionHidden,
			rules.ConditionVulnerable,
		),
		TemporaryArmor: []projectionstore.DaggerheartTemporaryArmor{
			{Source: "spell", Duration: "scene", SourceID: "src-1", Amount: 2},
		},
		LifeState: mechanics.LifeStateUnconscious,
	}

	got := CharacterStateToProto(state)
	if got.GetHp() != 5 || got.GetHope() != 2 || got.GetStress() != 3 || got.GetArmor() != 1 {
		t.Fatalf("unexpected numeric mapping: %+v", got)
	}
	if got.GetLifeState() != pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS {
		t.Fatalf("life state = %v, want unconscious", got.GetLifeState())
	}
	if len(got.GetConditionStates()) != 2 {
		t.Fatalf("conditions len = %d, want 2", len(got.GetConditionStates()))
	}
	if len(got.GetTemporaryArmorBuckets()) != 1 {
		t.Fatalf("temporary armor len = %d, want 1", len(got.GetTemporaryArmorBuckets()))
	}
	if got.GetTemporaryArmorBuckets()[0].GetAmount() != 2 {
		t.Fatalf("temporary armor amount = %d, want 2", got.GetTemporaryArmorBuckets()[0].GetAmount())
	}
}

func TestOptionalInt32(t *testing.T) {
	if OptionalInt32(nil) != nil {
		t.Fatal("OptionalInt32(nil) should be nil")
	}
	value := 7
	got := OptionalInt32(&value)
	if got == nil || *got != 7 {
		t.Fatalf("OptionalInt32(&7) = %v, want 7", got)
	}
}

func standardProjectionConditions(codes ...string) []projectionstore.DaggerheartConditionState {
	out := make([]projectionstore.DaggerheartConditionState, 0, len(codes))
	for _, code := range codes {
		out = append(out, projectionstore.DaggerheartConditionState{
			ID:       code,
			Class:    "standard",
			Standard: code,
			Code:     code,
			Label:    code,
		})
	}
	return out
}
