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
	// ParticipantID links ownership/ownership intent.
	ParticipantID string
	// SystemProfile carries system-specific structured data for mechanics systems.
	SystemProfile map[string]any
}
