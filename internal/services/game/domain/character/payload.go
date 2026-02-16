package character

// CreatePayload captures the payload for character.create commands and character.created events.
type CreatePayload struct {
	CharacterID string `json:"character_id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Notes       string `json:"notes,omitempty"`
}

// UpdatePayload captures the payload for character.update commands and character.updated events.
type UpdatePayload struct {
	CharacterID string            `json:"character_id"`
	Fields      map[string]string `json:"fields"`
}

// DeletePayload captures the payload for character.delete commands and character.deleted events.
type DeletePayload struct {
	CharacterID string `json:"character_id"`
	Reason      string `json:"reason,omitempty"`
}

// ProfileUpdatePayload captures the payload for character.profile_update commands and character.profile_updated events.
type ProfileUpdatePayload struct {
	CharacterID   string         `json:"character_id"`
	SystemProfile map[string]any `json:"system_profile,omitempty"`
}
