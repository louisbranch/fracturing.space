package character

// State captures character facts derived from domain events.
type State struct {
	Created       bool
	Deleted       bool
	CharacterID   string
	Name          string
	Kind          string
	Notes         string
	ParticipantID string
	SystemProfile map[string]any
}
