package readiness

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"

const (
	// RejectionCodeSessionReadinessCampaignStatusDisallowsStart indicates campaign lifecycle state disallows session start.
	RejectionCodeSessionReadinessCampaignStatusDisallowsStart = "SESSION_READINESS_CAMPAIGN_STATUS_DISALLOWS_START"
	// RejectionCodeSessionReadinessActiveSessionExists indicates another session is already active.
	RejectionCodeSessionReadinessActiveSessionExists = "SESSION_READINESS_ACTIVE_SESSION_EXISTS"
	// RejectionCodeSessionReadinessAIAgentRequired indicates AI/HYBRID campaigns require a bound AI agent.
	RejectionCodeSessionReadinessAIAgentRequired = "SESSION_READINESS_AI_AGENT_REQUIRED"
	// RejectionCodeSessionReadinessAIGMParticipantRequired indicates AI/HYBRID campaigns require at least one AI-controlled GM participant.
	RejectionCodeSessionReadinessAIGMParticipantRequired = "SESSION_READINESS_AI_GM_PARTICIPANT_REQUIRED"
	// RejectionCodeSessionReadinessGMRequired indicates there is no active GM participant.
	RejectionCodeSessionReadinessGMRequired = "SESSION_READINESS_GM_REQUIRED"
	// RejectionCodeSessionReadinessPlayerRequired indicates there is no active player participant.
	RejectionCodeSessionReadinessPlayerRequired = "SESSION_READINESS_PLAYER_REQUIRED"
	// RejectionCodeSessionReadinessPlayerCharacterRequired indicates a player controls no active characters.
	RejectionCodeSessionReadinessPlayerCharacterRequired = "SESSION_READINESS_PLAYER_CHARACTER_REQUIRED"
	// RejectionCodeSessionReadinessCharacterControllerRequired indicates an active character is unassigned.
	RejectionCodeSessionReadinessCharacterControllerRequired = "SESSION_READINESS_CHARACTER_CONTROLLER_REQUIRED"
	// RejectionCodeSessionReadinessCharacterSystemRequired indicates system-specific character readiness failed.
	RejectionCodeSessionReadinessCharacterSystemRequired = "SESSION_READINESS_CHARACTER_SYSTEM_REQUIRED"
)

// Rejection describes why session-start readiness evaluation failed.
type Rejection struct {
	Code    string
	Message string
}

// Blocker describes one readiness invariant currently preventing session start.
type Blocker struct {
	Code     string
	Message  string
	Metadata map[string]string
	Action   Action
}

// Report captures all readiness blockers for deterministic participant/operator feedback.
type Report struct {
	Blockers []Blocker
}

// Ready reports whether the campaign has zero readiness blockers.
func (r Report) Ready() bool {
	return len(r.Blockers) == 0
}

// ReportOptions controls optional boundary checks and system-specific readiness evaluation.
type ReportOptions struct {
	SystemReadiness        CharacterSystemReadiness
	IncludeSessionBoundary bool
	HasActiveSession       bool
}

// CharacterSystemReadiness checks optional system-specific readiness for a
// character ID within the active game system.
//
// Returning false blocks session start. The reason should be concise and
// operator-readable because it is surfaced directly in domain rejection messages.
type CharacterSystemReadiness func(characterID string) (ready bool, reason string)

// EvaluateSessionStart evaluates campaign readiness invariants required before
// accepting session.start.
//
// The checks are deterministic and run only against aggregate replay state:
//  1. at least one active GM participant,
//  2. at least one active player participant,
//  3. each active player controls at least one active character,
//  4. every active character has a controller,
//  5. optional system-specific readiness for every active character.
func EvaluateSessionStart(state aggregate.State, systemReadiness CharacterSystemReadiness) *Rejection {
	report := EvaluateSessionStartReport(state, ReportOptions{SystemReadiness: systemReadiness})
	if report.Ready() {
		return nil
	}
	first := report.Blockers[0]
	return &Rejection{
		Code:    first.Code,
		Message: first.Message,
	}
}

// EvaluateSessionStartReport evaluates readiness invariants and returns every
// blocker in deterministic order so transports can render actionable checklists.
func EvaluateSessionStartReport(state aggregate.State, options ReportOptions) Report {
	blockers := evaluateSessionBoundaryBlockers(state, options)
	blockers = append(blockers, evaluateCoreSessionStartBlockers(state, options.SystemReadiness)...)
	return Report{Blockers: blockers}
}
