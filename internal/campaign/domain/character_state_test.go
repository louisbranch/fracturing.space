package domain

import (
	"errors"
	"testing"
)

func TestCreateCharacterStateSuccess(t *testing.T) {
	profile := CharacterProfile{HpMax: 6, StressMax: 4}
	input := CreateCharacterStateInput{
		CampaignID:  "camp-123",
		CharacterID: "char-456",
		Hope:        2,
		Stress:      1,
		Hp:          5,
	}

	state, err := CreateCharacterState(input, profile)
	if err != nil {
		t.Fatalf("create character state: %v", err)
	}
	if state.CampaignID != input.CampaignID {
		t.Fatalf("expected campaign id %q, got %q", input.CampaignID, state.CampaignID)
	}
	if state.Hope != 2 || state.Stress != 1 || state.Hp != 5 {
		t.Fatalf("expected state values preserved")
	}
}

func TestCreateCharacterStateValidationErrors(t *testing.T) {
	profile := CharacterProfile{HpMax: 6, StressMax: 4}
	tests := []struct {
		name  string
		input CreateCharacterStateInput
		err   error
	}{
		{
			name:  "invalid hope",
			input: CreateCharacterStateInput{Hope: 7, Stress: 0, Hp: 0},
			err:   ErrInvalidHope,
		},
		{
			name:  "stress too high",
			input: CreateCharacterStateInput{Hope: 0, Stress: 5, Hp: 0},
			err:   ErrInvalidStress,
		},
		{
			name:  "hp too high",
			input: CreateCharacterStateInput{Hope: 0, Stress: 0, Hp: 7},
			err:   ErrInvalidHp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CreateCharacterState(tt.input, profile)
			if !errors.Is(err, tt.err) {
				t.Fatalf("expected error %v, got %v", tt.err, err)
			}
		})
	}
}

func TestPatchCharacterStateUpdates(t *testing.T) {
	existing := CharacterState{Hope: 1, Stress: 2, Hp: 3}
	newHope := 4
	newHp := 5
	patch := PatchCharacterStateInput{Hope: &newHope, Hp: &newHp}
	profile := CharacterProfile{HpMax: 6, StressMax: 4}

	updated, err := PatchCharacterState(existing, patch, profile)
	if err != nil {
		t.Fatalf("patch character state: %v", err)
	}
	if updated.Hope != 4 || updated.Hp != 5 {
		t.Fatalf("expected hope/hp updated")
	}
	if updated.Stress != existing.Stress {
		t.Fatalf("expected stress to remain %d", existing.Stress)
	}
}

func TestPatchCharacterStateInvalid(t *testing.T) {
	existing := CharacterState{Hope: 1, Stress: 2, Hp: 3}
	invalidHope := 9
	patch := PatchCharacterStateInput{Hope: &invalidHope}
	profile := CharacterProfile{HpMax: 6, StressMax: 4}

	_, err := PatchCharacterState(existing, patch, profile)
	if !errors.Is(err, ErrInvalidHope) {
		t.Fatalf("expected error %v, got %v", ErrInvalidHope, err)
	}
}
