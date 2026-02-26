package character

// State captures replayed character identity and profile state.
//
// Characters are mutable records tied to participants and are intentionally
// persisted as a projection-friendly state for fast lookups and authorization.
type State struct {
	// Created indicates the character exists in the campaign.
	Created bool
	// Deleted marks soft-delete semantics before full lifecycle cleanup.
	Deleted bool
	// CharacterID is the immutable identifier used in command payloads and events.
	CharacterID string
	// Name is the display label visible in gameplay and UIs.
	Name string
	// Kind captures whether this is PC/NPC (and future kinds).
	Kind string
	// Notes stores campaign-local free-form character metadata.
	Notes string
	// AvatarSetID identifies the avatar set bound to this character.
	AvatarSetID string
	// AvatarAssetID identifies the avatar image within AvatarSetID.
	AvatarAssetID string
	// Pronouns stores optional free-form character pronouns.
	Pronouns string
	// Aliases stores normalized ordered aliases.
	Aliases []string
	// OwnerParticipantID is the governance owner participant for mutation authority.
	OwnerParticipantID string
	// ParticipantID stores controller assignment for operational gameplay control.
	ParticipantID string
	// SystemProfile carries system-specific structured data for mechanics systems.
	SystemProfile map[string]any
}
