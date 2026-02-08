package character

import (
	"errors"
	"testing"
	"time"
)

func TestCreateCharacterNormalizesInput(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 10, 0, 0, 0, time.UTC)
	input := CreateCharacterInput{
		CampaignID: "camp-123",
		Name:       "  Alice  ",
		Kind:       CharacterKindPC,
		Notes:      "  A brave warrior  ",
	}

	character, err := CreateCharacter(input, func() time.Time { return fixedTime }, func() (string, error) {
		return "character-456", nil
	})
	if err != nil {
		t.Fatalf("create character: %v", err)
	}

	if character.ID != "character-456" {
		t.Fatalf("expected id character-456, got %q", character.ID)
	}
	if character.CampaignID != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", character.CampaignID)
	}
	if character.Name != "Alice" {
		t.Fatalf("expected trimmed name, got %q", character.Name)
	}
	if character.Kind != CharacterKindPC {
		t.Fatalf("expected kind PC, got %v", character.Kind)
	}
	if character.Notes != "A brave warrior" {
		t.Fatalf("expected trimmed notes, got %q", character.Notes)
	}
	if !character.CreatedAt.Equal(fixedTime) || !character.UpdatedAt.Equal(fixedTime) {
		t.Fatalf("expected timestamps to match fixed time")
	}
}

func TestNormalizeCreateCharacterInputValidation(t *testing.T) {
	tests := []struct {
		name  string
		input CreateCharacterInput
		err   error
	}{
		{
			name: "empty campaign id",
			input: CreateCharacterInput{
				CampaignID: "   ",
				Name:       "Alice",
				Kind:       CharacterKindPC,
			},
			err: ErrEmptyCampaignID,
		},
		{
			name: "empty name",
			input: CreateCharacterInput{
				CampaignID: "camp-123",
				Name:       "   ",
				Kind:       CharacterKindPC,
			},
			err: ErrEmptyCharacterName,
		},
		{
			name: "missing kind",
			input: CreateCharacterInput{
				CampaignID: "camp-123",
				Name:       "Alice",
				Kind:       CharacterKindUnspecified,
			},
			err: ErrInvalidCharacterKind,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeCreateCharacterInput(tt.input)
			if !errors.Is(err, tt.err) {
				t.Fatalf("expected error %v, got %v", tt.err, err)
			}
		})
	}
}
