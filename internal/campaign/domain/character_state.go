package domain

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidHope indicates hope is outside valid range 0..6.
	ErrInvalidHope = errors.New("hope must be in range 0..6")
	// ErrInvalidStress indicates stress exceeds profile maximum.
	ErrInvalidStress = errors.New("stress exceeds profile maximum")
	// ErrInvalidHp indicates hp exceeds profile maximum.
	ErrInvalidHp = errors.New("hp exceeds profile maximum")
)

// CharacterState represents the mutable state values for a character.
type CharacterState struct {
	CampaignID  string
	CharacterID string
	Hope        int
	Stress      int
	Hp          int
}

// CreateCharacterStateInput describes the input for creating a character state.
type CreateCharacterStateInput struct {
	CampaignID  string
	CharacterID string
	Hope        int
	Stress      int
	Hp          int
}

// CreateCharacterState creates a new character state with validation.
func CreateCharacterState(input CreateCharacterStateInput, profile CharacterProfile) (CharacterState, error) {
	if err := ValidateCharacterState(input.Hope, input.Stress, input.Hp, profile); err != nil {
		return CharacterState{}, err
	}

	return CharacterState{
		CampaignID:  input.CampaignID,
		CharacterID: input.CharacterID,
		Hope:        input.Hope,
		Stress:      input.Stress,
		Hp:          input.Hp,
	}, nil
}

// ValidateCharacterState validates state invariants against a profile.
func ValidateCharacterState(hope, stress, hp int, profile CharacterProfile) error {
	if hope < 0 || hope > 6 {
		return ErrInvalidHope
	}
	if stress < 0 || stress > profile.StressMax {
		return fmt.Errorf("%w: stress %d exceeds maximum %d", ErrInvalidStress, stress, profile.StressMax)
	}
	if hp < 0 || hp > profile.HpMax {
		return fmt.Errorf("%w: hp %d exceeds maximum %d", ErrInvalidHp, hp, profile.HpMax)
	}
	return nil
}

// PatchCharacterStateInput describes optional fields for patching a state.
type PatchCharacterStateInput struct {
	Hope   *int
	Stress *int
	Hp     *int
}

// PatchCharacterState applies a patch to an existing state, returning a new state.
func PatchCharacterState(existing CharacterState, patch PatchCharacterStateInput, profile CharacterProfile) (CharacterState, error) {
	result := existing

	if patch.Hope != nil {
		result.Hope = *patch.Hope
	}
	if patch.Stress != nil {
		result.Stress = *patch.Stress
	}
	if patch.Hp != nil {
		result.Hp = *patch.Hp
	}

	if err := ValidateCharacterState(result.Hope, result.Stress, result.Hp, profile); err != nil {
		return CharacterState{}, err
	}

	return result, nil
}
