package scene

// State captures the replayed scene context for a single narrative scope.
//
// Each scene is an independent sub-session boundary with its own character
// roster, gate, and spotlight. The command engine uses this to enforce
// scene-scoped gate blocking and spotlight routing without affecting
// other active scenes.
type State struct {
	// SceneID is the canonical identifier for this scene.
	SceneID string
	// Name is a human-facing label (e.g., "The Dark Cavern").
	Name string
	// Description is optional narrative setup text.
	Description string
	// Active indicates whether the scene is still running.
	Active bool
	// Characters tracks character IDs present in this scene (PCs and NPCs).
	// The same character may appear in multiple scenes simultaneously.
	Characters map[string]bool
	// GateOpen blocks scene-scoped commands while adjudication is paused.
	GateOpen bool
	// GateID identifies the active gate when GateOpen is true.
	GateID string
	// SpotlightType tracks which entity type currently holds initiative context.
	SpotlightType SpotlightType
	// SpotlightCharacterID tracks the focused character in spotlight workflows.
	SpotlightCharacterID string
}

// HasPC returns true if the scene contains at least one character whose ID
// is in the provided PC set. This is used for the "at least one PC" invariant.
func (s State) HasPC(pcs map[string]bool) bool {
	for charID := range s.Characters {
		if pcs[charID] {
			return true
		}
	}
	return false
}
