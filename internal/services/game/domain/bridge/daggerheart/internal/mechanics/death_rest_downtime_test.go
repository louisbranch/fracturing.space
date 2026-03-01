package mechanics

import (
	"errors"
	"testing"
)

func TestResolveDeathMove_BlazeOfGlory(t *testing.T) {
	outcome, err := ResolveDeathMove(DeathMoveInput{
		Move:    DeathMoveBlazeOfGlory,
		Level:   1,
		HPMax:   6,
		HopeMax: 6,
	})
	if err != nil {
		t.Fatalf("ResolveDeathMove: %v", err)
	}
	if outcome.LifeState != LifeStateBlazeOfGlory {
		t.Fatalf("LifeState = %q, want %q", outcome.LifeState, LifeStateBlazeOfGlory)
	}
}

func TestResolveRestOutcome_InvalidShortRestSequence(t *testing.T) {
	_, err := ResolveRestOutcome(RestState{ConsecutiveShortRests: 3}, RestTypeShort, false, 1, 4)
	if !errors.Is(err, ErrInvalidRestSequence) {
		t.Fatalf("expected ErrInvalidRestSequence, got %v", err)
	}
}

func TestApplyDowntimeMove_RepairAllArmorClearsShortRestArmor(t *testing.T) {
	state := NewCharacterState(CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Armor:       0,
		ArmorMax:    2,
	})
	state.ApplyTemporaryArmor(TemporaryArmorBucket{Source: "spell", Duration: "short_rest", Amount: 2})
	state.ApplyTemporaryArmor(TemporaryArmorBucket{Source: "ritual", Duration: "long_rest", Amount: 1})

	result := ApplyDowntimeMove(state, DowntimeRepairAllArmor, DowntimeOptions{})
	if result.ArmorAfter != state.ResourceCap(ResourceArmor) {
		t.Fatalf("ArmorAfter = %d, want cap %d", result.ArmorAfter, state.ResourceCap(ResourceArmor))
	}
	if state.TemporaryArmorAmount() != 1 {
		t.Fatalf("TemporaryArmorAmount = %d, want 1", state.TemporaryArmorAmount())
	}
}
