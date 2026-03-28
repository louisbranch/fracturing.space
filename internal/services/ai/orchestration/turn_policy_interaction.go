package orchestration

import (
	"encoding/json"
	"strings"
)

const playerPhaseStartToolName = "interaction_open_scene_player_phase"
const reviewResolveToolName = "interaction_resolve_scene_player_review"
const interruptResolutionToolName = "interaction_session_ooc_resolve"

// NewInteractionTurnPolicy returns the default turn policy for the current
// interaction-driven AI GM runtime.
func NewInteractionTurnPolicy() TurnPolicy {
	return interactionTurnPolicy{}
}

type interactionTurnPolicy struct{}

func (interactionTurnPolicy) Controller(commitToolName string) TurnController {
	return &interactionTurnController{commitToolName: strings.TrimSpace(commitToolName)}
}

type interactionTurnController struct {
	commitToolName          string
	committedOrResolved     bool
	readyForCompletion      bool
	lastCommitToolOrder     int
	lastPlayerHandoffOrder  int
	successfulToolCallOrder int
}

func (c *interactionTurnController) ObserveSuccessfulTool(name string, output string) {
	name = strings.TrimSpace(name)
	if toolCommitsOrResolvesInteraction(name, c.commitToolName) {
		c.committedOrResolved = true
	}
	c.successfulToolCallOrder++
	switch name {
	case c.commitToolName, playerPhaseStartToolName, reviewResolveToolName, interruptResolutionToolName:
		c.lastCommitToolOrder = c.successfulToolCallOrder
	}
	if ready, handoff, ok := toolResultControlState(output); ok {
		c.readyForCompletion = ready
		if handoff {
			c.lastPlayerHandoffOrder = c.successfulToolCallOrder
		}
		return
	}
	if toolHandsControlBackToPlayers(name) {
		c.lastPlayerHandoffOrder = c.successfulToolCallOrder
	}
}

func (c *interactionTurnController) HasCommittedOrResolvedInteraction() bool {
	return c.committedOrResolved
}

func (c *interactionTurnController) ReadyForCompletion() bool {
	return c.readyForCompletion
}

func (c *interactionTurnController) PlayerHandoffRegressed() bool {
	return c.lastCommitToolOrder > 0 &&
		c.lastPlayerHandoffOrder > 0 &&
		c.lastCommitToolOrder > c.lastPlayerHandoffOrder
}

func (c *interactionTurnController) BuildCommitReminder(text string) string {
	var b strings.Builder
	b.WriteString("You returned narration without making the authoritative interaction update for this turn.\n")
	b.WriteString("Use interaction_record_scene_gm_interaction for standalone narration, interaction_resolve_scene_player_review for GM review, or interaction_session_ooc_resolve for post-OOC resolution before returning final text.\n")
	b.WriteString("Use interaction_open_scene_player_phase when the interaction should immediately hand control to players.\n")
	b.WriteString("Commit one structured GM interaction made of ordered beats rather than separate narration and frame artifacts.\n")
	b.WriteString("Keep related prose in one beat unless the GM function or information context materially changes.\n")
	b.WriteString("If there is no active scene, create one with scene_create (which activates by default) or activate an existing scene.\n")
	if text != "" {
		b.WriteString("Use this draft narration as the commit text unless you need a small correction:\n")
		b.WriteString(text)
	}
	return b.String()
}

func (c *interactionTurnController) BuildTurnCompletionReminder(text string) string {
	var b strings.Builder
	b.WriteString("You already made an authoritative GM update, but the AI GM turn is not complete yet.\n")
	b.WriteString("Return final text only after the next player phase is open for players or the session is paused for OOC.\n")
	b.WriteString("Use interaction_open_scene_player_phase to hand control to players, interaction_resolve_scene_player_review when GM review is pending, or interaction_session_ooc_resolve when post-OOC interaction resolution is still blocking players.\n")
	if text != "" {
		b.WriteString("Keep this draft narration unless you need a small correction:\n")
		b.WriteString(text)
	}
	return b.String()
}

func (c *interactionTurnController) BuildPlayerPhaseStartReminder(text string) string {
	text = strings.TrimSpace(text)
	var b strings.Builder
	b.WriteString("You committed GM narration after opening a player phase, which leaves the interaction without an active player handoff.\n")
	b.WriteString("If players should act next, call interaction_open_scene_player_phase now with the acting character_ids and the structured interaction whose final prompt beat hands control to players.\n")
	b.WriteString("The beat-based interaction must be committed before the player phase is opened.\n")
	if text != "" {
		b.WriteString("Keep this final narration unless you need a small correction:\n")
		b.WriteString(text)
	}
	return b.String()
}

func toolCommitsOrResolvesInteraction(name, configuredCommitTool string) bool {
	switch strings.TrimSpace(name) {
	case strings.TrimSpace(configuredCommitTool), playerPhaseStartToolName, reviewResolveToolName, interruptResolutionToolName:
		return true
	default:
		return false
	}
}

func toolHandsControlBackToPlayers(name string) bool {
	switch strings.TrimSpace(name) {
	case playerPhaseStartToolName, reviewResolveToolName, interruptResolutionToolName:
		return true
	default:
		return false
	}
}

type toolResultControlHints struct {
	AITurnReadyForCompletion *bool `json:"ai_turn_ready_for_completion,omitempty"`
	PlayerPhase              struct {
		PhaseID              string   `json:"phase_id,omitempty"`
		Status               string   `json:"status,omitempty"`
		ActingParticipantIDs []string `json:"acting_participant_ids,omitempty"`
	} `json:"player_phase"`
	OOC struct {
		Open              bool `json:"open"`
		ResolutionPending bool `json:"resolution_pending"`
	} `json:"ooc"`
}

func toolResultControlState(output string) (ready bool, handoff bool, ok bool) {
	var hints toolResultControlHints
	if err := json.Unmarshal([]byte(output), &hints); err != nil {
		return false, false, false
	}
	if hints.AITurnReadyForCompletion != nil {
		ready = *hints.AITurnReadyForCompletion
	} else {
		ready = hints.OOC.Open && !hints.OOC.ResolutionPending
		if !ready {
			ready = strings.EqualFold(strings.TrimSpace(hints.PlayerPhase.Status), "players") &&
				strings.TrimSpace(hints.PlayerPhase.PhaseID) != "" &&
				len(hints.PlayerPhase.ActingParticipantIDs) > 0
		}
	}
	handoff = strings.EqualFold(strings.TrimSpace(hints.PlayerPhase.Status), "players") &&
		strings.TrimSpace(hints.PlayerPhase.PhaseID) != "" &&
		len(hints.PlayerPhase.ActingParticipantIDs) > 0
	return ready, handoff, true
}
