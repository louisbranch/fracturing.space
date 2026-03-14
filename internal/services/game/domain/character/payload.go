package character

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"

// CreatePayload captures the payload for character.create commands and character.created events.
type CreatePayload struct {
	CharacterID        ids.CharacterID   `json:"character_id"`
	OwnerParticipantID ids.ParticipantID `json:"owner_participant_id,omitempty"`
	ParticipantID      ids.ParticipantID `json:"participant_id,omitempty"`
	Name               string            `json:"name"`
	Kind               string            `json:"kind"`
	Notes              string            `json:"notes,omitempty"`
	AvatarSetID        string            `json:"avatar_set_id,omitempty"`
	AvatarAssetID      string            `json:"avatar_asset_id,omitempty"`
	Pronouns           string            `json:"pronouns,omitempty"`
	Aliases            []string          `json:"aliases,omitempty"`
}

// UpdatePayload captures the payload for character.update commands and character.updated events.
type UpdatePayload struct {
	CharacterID ids.CharacterID   `json:"character_id"`
	Fields      map[string]string `json:"fields"`
}

// DeletePayload captures the payload for character.delete commands and character.deleted events.
type DeletePayload struct {
	CharacterID ids.CharacterID `json:"character_id"`
	Reason      string          `json:"reason,omitempty"`
}
