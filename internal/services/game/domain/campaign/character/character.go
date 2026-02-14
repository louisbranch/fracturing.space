// Package character provides character definitions and profile management.
package character

import (
	"fmt"
	"strings"
	"time"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
)

// CharacterKind describes the kind of character in a campaign.
type CharacterKind int

const (
	// CharacterKindUnspecified represents an invalid character kind value.
	CharacterKindUnspecified CharacterKind = iota
	// CharacterKindPC indicates a player character.
	CharacterKindPC
	// CharacterKindNPC indicates a non-player character.
	CharacterKindNPC
)

var (
	// ErrEmptyCampaignID indicates a missing campaign ID.
	ErrEmptyCampaignID = apperrors.New(apperrors.CodeCharacterEmptyCampaignID, "campaign id is required")
	// ErrEmptyCharacterName indicates a missing character name.
	ErrEmptyCharacterName = apperrors.New(apperrors.CodeCharacterEmptyName, "character name is required")
	// ErrInvalidCharacterKind indicates a missing or invalid character kind.
	ErrInvalidCharacterKind = apperrors.New(apperrors.CodeCharacterInvalidKind, "character kind is required")
)

// Character represents a character (PC or NPC) in a campaign.
type Character struct {
	ID         string
	CampaignID string
	// ParticipantID is the participant assigned to control the character.
	// Empty means unassigned.
	ParticipantID string
	Name          string
	Kind          CharacterKind
	Notes         string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CreateCharacterInput describes the metadata needed to create a character.
type CreateCharacterInput struct {
	CampaignID string
	Name       string
	Kind       CharacterKind
	Notes      string
}

// CreateCharacter creates a new character with a generated ID and timestamps.
func CreateCharacter(input CreateCharacterInput, now func() time.Time, idGenerator func() (string, error)) (Character, error) {
	if now == nil {
		now = time.Now
	}
	if idGenerator == nil {
		idGenerator = id.NewID
	}

	normalized, err := NormalizeCreateCharacterInput(input)
	if err != nil {
		return Character{}, err
	}

	characterID, err := idGenerator()
	if err != nil {
		return Character{}, fmt.Errorf("generate character id: %w", err)
	}

	createdAt := now().UTC()
	return Character{
		ID:            characterID,
		CampaignID:    normalized.CampaignID,
		ParticipantID: "",
		Name:          normalized.Name,
		Kind:          normalized.Kind,
		Notes:         normalized.Notes,
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}, nil
}

// CharacterKindFromLabel parses a string label into a CharacterKind.
// It trims whitespace and matches case-insensitively. Both short ("PC")
// and prefixed ("CHARACTER_KIND_PC") forms are accepted.
func CharacterKindFromLabel(value string) (CharacterKind, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return CharacterKindUnspecified, fmt.Errorf("character kind is required")
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "PC", "CHARACTER_KIND_PC":
		return CharacterKindPC, nil
	case "NPC", "CHARACTER_KIND_NPC":
		return CharacterKindNPC, nil
	default:
		return CharacterKindUnspecified, fmt.Errorf("unknown character kind: %s", trimmed)
	}
}

// NormalizeCreateCharacterInput trims and validates character input metadata.
func NormalizeCreateCharacterInput(input CreateCharacterInput) (CreateCharacterInput, error) {
	input.CampaignID = strings.TrimSpace(input.CampaignID)
	if input.CampaignID == "" {
		return CreateCharacterInput{}, ErrEmptyCampaignID
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return CreateCharacterInput{}, ErrEmptyCharacterName
	}
	if input.Kind == CharacterKindUnspecified {
		return CreateCharacterInput{}, ErrInvalidCharacterKind
	}
	input.Notes = strings.TrimSpace(input.Notes)
	return input, nil
}
