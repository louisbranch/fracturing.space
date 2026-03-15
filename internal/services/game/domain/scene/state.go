package scene

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"

type PlayerPhaseStatus string

const (
	PlayerPhaseStatusPlayers  PlayerPhaseStatus = "players"
	PlayerPhaseStatusGMReview PlayerPhaseStatus = "gm_review"
)

type PlayerPhaseSlotReviewStatus string

const (
	PlayerPhaseSlotReviewStatusOpen             PlayerPhaseSlotReviewStatus = "open"
	PlayerPhaseSlotReviewStatusUnderReview      PlayerPhaseSlotReviewStatus = "under_review"
	PlayerPhaseSlotReviewStatusAccepted         PlayerPhaseSlotReviewStatus = "accepted"
	PlayerPhaseSlotReviewStatusChangesRequested PlayerPhaseSlotReviewStatus = "changes_requested"
)

// State captures the replayed scene context for a single narrative scope.
//
// Each scene is an independent sub-session boundary with its own character
// roster, gate, and spotlight. The command engine uses this to enforce
// scene-scoped gate blocking and spotlight routing without affecting
// other active scenes.
type State struct {
	// SceneID is the canonical identifier for this scene.
	SceneID ids.SceneID
	// Name is a human-facing label (e.g., "The Dark Cavern").
	Name string
	// Description is optional narrative setup text.
	Description string
	// Active indicates whether the scene is still running.
	Active bool
	// Characters tracks character IDs present in this scene (PCs and NPCs).
	// The same character may appear in multiple scenes simultaneously.
	Characters map[ids.CharacterID]bool
	// GateOpen blocks scene-scoped commands while adjudication is paused.
	GateOpen bool
	// GateID identifies the active gate when GateOpen is true.
	GateID ids.GateID
	// SpotlightType tracks which entity type currently holds initiative context.
	SpotlightType SpotlightType
	// SpotlightCharacterID tracks the focused character in spotlight workflows.
	SpotlightCharacterID ids.CharacterID
	// PlayerPhaseID identifies the currently open player phase, when any.
	PlayerPhaseID string
	// PlayerPhaseFrameText stores the GM frame text for the open player phase.
	PlayerPhaseFrameText string
	// PlayerPhaseStatus stores whether the open phase is accepting player input
	// or waiting on GM review.
	PlayerPhaseStatus PlayerPhaseStatus
	// PlayerPhaseActingCharacters stores the selected acting characters for the open player phase.
	PlayerPhaseActingCharacters []ids.CharacterID
	// PlayerPhaseActingParticipants stores the participants currently allowed to act in the open player phase.
	PlayerPhaseActingParticipants map[ids.ParticipantID]bool
	// PlayerPhaseSlots stores the latest participant-owned slot state for the phase.
	PlayerPhaseSlots map[ids.ParticipantID]PlayerPhaseSlot
	// GMOutputText stores the latest committed GM narration for the scene.
	GMOutputText string
	// GMOutputParticipantID stores which participant committed the latest narration.
	GMOutputParticipantID ids.ParticipantID
}

// HasPC returns true if the scene contains at least one character whose ID
// is in the provided PC set. This is used for the "at least one PC" invariant.
func (s State) HasPC(pcs map[ids.CharacterID]bool) bool {
	for charID := range s.Characters {
		if pcs[charID] {
			return true
		}
	}
	return false
}

// PlayerPhaseSlot stores one participant-owned submission slot in the open player phase.
type PlayerPhaseSlot struct {
	ParticipantID      ids.ParticipantID
	CharacterIDs       []ids.CharacterID
	SummaryText        string
	Yielded            bool
	ReviewStatus       PlayerPhaseSlotReviewStatus
	ReviewReason       string
	ReviewCharacterIDs []ids.CharacterID
}
