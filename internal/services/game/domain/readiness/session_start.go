package readiness

import (
	"fmt"
	"sort"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	domainids "github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

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
	blockers := make([]Blocker, 0, 8)

	if options.IncludeSessionBoundary {
		if !campaignStatusAllowsSessionStart(state.Campaign.Status) {
			blockers = append(blockers, newBlocker(
				RejectionCodeSessionReadinessCampaignStatusDisallowsStart,
				fmt.Sprintf("campaign readiness requires campaign status draft or active before session start (status=%s)", normalizeCampaignStatus(state.Campaign.Status)),
				map[string]string{
					"status": string(normalizeCampaignStatus(state.Campaign.Status)),
				},
			))
		}
		if options.HasActiveSession {
			blockers = append(blockers, newBlocker(
				RejectionCodeSessionReadinessActiveSessionExists,
				"campaign readiness requires no active session",
				nil,
			))
		}
	}

	blockers = append(blockers, evaluateCoreSessionStartBlockers(state, options.SystemReadiness)...)
	return Report{Blockers: blockers}
}

func evaluateCoreSessionStartBlockers(state aggregate.State, systemReadiness CharacterSystemReadiness) []Blocker {
	blockers := make([]Blocker, 0, 6)

	if isAIGMMode(state.Campaign.GmMode) && strings.TrimSpace(state.Campaign.AIAgentID) == "" {
		blockers = append(blockers, newBlocker(
			RejectionCodeSessionReadinessAIAgentRequired,
			"campaign readiness requires ai agent binding for ai gm mode",
			nil,
		))
	}

	activeParticipants := activeParticipantsByID(state)
	if isAIGMMode(state.Campaign.GmMode) && len(activeParticipants.aiGMIDs) == 0 {
		blockers = append(blockers, newBlocker(
			RejectionCodeSessionReadinessAIGMParticipantRequired,
			"campaign readiness requires at least one ai-controlled gm participant for ai gm mode",
			nil,
		))
	}
	if len(activeParticipants.gmIDs) == 0 {
		blockers = append(blockers, newBlocker(
			RejectionCodeSessionReadinessGMRequired,
			"campaign readiness requires at least one gm participant",
			nil,
		))
	}
	if len(activeParticipants.playerIDs) == 0 {
		blockers = append(blockers, newBlocker(
			RejectionCodeSessionReadinessPlayerRequired,
			"campaign readiness requires at least one player participant",
			nil,
		))
	}

	activeCharacters := activeCharactersByID(state)
	playerCharacterCounts := make(map[string]int, len(activeParticipants.playerIDs))
	for _, playerID := range activeParticipants.playerIDs {
		playerCharacterCounts[playerID] = 0
	}

	for _, characterID := range activeCharacters.ids {
		characterState := activeCharacters.byID[characterID]
		controllerParticipantID := strings.TrimSpace(characterState.ParticipantID)
		if controllerParticipantID == "" {
			characterLabel, metadata := readinessCharacterLabelAndMetadata(characterID, characterState.Name)
			blockers = append(blockers, newBlocker(
				RejectionCodeSessionReadinessCharacterControllerRequired,
				fmt.Sprintf("campaign readiness requires character %s to have a controller", characterLabel),
				metadata,
			))
		}

		if _, ok := playerCharacterCounts[controllerParticipantID]; ok {
			playerCharacterCounts[controllerParticipantID]++
		}

		if systemReadiness == nil {
			continue
		}
		ready, reason := systemReadiness(characterID)
		if ready {
			continue
		}
		reason = strings.TrimSpace(reason)
		characterLabel, metadata := readinessCharacterLabelAndMetadata(characterID, characterState.Name)
		if reason != "" {
			metadata["reason"] = reason
		}
		blockers = append(blockers, newBlocker(
			RejectionCodeSessionReadinessCharacterSystemRequired,
			systemReadinessMessage(characterLabel, reason),
			metadata,
		))
	}

	for _, playerID := range activeParticipants.playerIDs {
		if playerCharacterCounts[playerID] > 0 {
			continue
		}
		playerState := activeParticipants.byID[playerID]
		playerName := strings.TrimSpace(playerState.Name)
		playerLabel := playerName
		if playerLabel == "" {
			playerLabel = playerID
		}
		metadata := map[string]string{
			"participant_id": playerID,
		}
		if playerName != "" {
			metadata["participant_name"] = playerName
		}
		blockers = append(blockers, newBlocker(
			RejectionCodeSessionReadinessPlayerCharacterRequired,
			fmt.Sprintf("campaign readiness requires player participant %s to control at least one character", playerLabel),
			metadata,
		))
	}

	return blockers
}

func readinessCharacterLabelAndMetadata(characterID, characterName string) (string, map[string]string) {
	trimmedID := strings.TrimSpace(characterID)
	metadata := map[string]string{
		"character_id": trimmedID,
	}

	trimmedName := strings.TrimSpace(characterName)
	if trimmedName != "" {
		metadata["character_name"] = trimmedName
		return trimmedName, metadata
	}

	return trimmedID, metadata
}

func newBlocker(code, message string, metadata map[string]string) Blocker {
	cloned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return Blocker{
		Code:     code,
		Message:  message,
		Metadata: cloned,
	}
}

func campaignStatusAllowsSessionStart(status campaign.Status) bool {
	switch normalizeCampaignStatus(status) {
	case campaign.StatusDraft, campaign.StatusActive:
		return true
	default:
		return false
	}
}

func normalizeCampaignStatus(status campaign.Status) campaign.Status {
	trimmed := strings.TrimSpace(string(status))
	normalized, ok := campaign.NormalizeStatus(trimmed)
	if ok {
		return normalized
	}
	return campaign.Status(trimmed)
}

func systemReadinessMessage(characterLabel, reason string) string {
	message := fmt.Sprintf("campaign readiness requires character %s to satisfy system readiness", characterLabel)
	if reason == "" {
		return message
	}
	return message + ": " + reason
}

func isAIGMMode(mode campaign.GmMode) bool {
	switch mode {
	case campaign.GmModeAI, campaign.GmModeHybrid:
		return true
	default:
		return false
	}
}

type participantIndex struct {
	byID      map[string]participant.State
	gmIDs     []string
	aiGMIDs   []string
	playerIDs []string
}

func activeParticipantsByID(state aggregate.State) participantIndex {
	indexed := participantIndex{byID: make(map[string]participant.State)}
	if len(state.Participants) == 0 {
		return indexed
	}

	pids := make([]string, 0, len(state.Participants))
	for participantID := range state.Participants {
		pids = append(pids, string(participantID))
	}
	sort.Strings(pids)

	for _, participantID := range pids {
		participantState := state.Participants[domainids.ParticipantID(participantID)]
		if !participantState.Joined || participantState.Left {
			continue
		}
		indexed.byID[participantID] = participantState
		role, ok := participant.NormalizeRole(string(participantState.Role))
		if !ok {
			continue
		}
		switch role {
		case participant.RoleGM:
			indexed.gmIDs = append(indexed.gmIDs, participantID)
			controller, ok := participant.NormalizeController(string(participantState.Controller))
			if ok && controller == participant.ControllerAI {
				indexed.aiGMIDs = append(indexed.aiGMIDs, participantID)
			}
		case participant.RolePlayer:
			indexed.playerIDs = append(indexed.playerIDs, participantID)
		}
	}

	return indexed
}

type characterIndex struct {
	byID map[string]aggregateCharacterState
	ids  []string
}

type aggregateCharacterState struct {
	ParticipantID string
	Name          string
}

func activeCharactersByID(state aggregate.State) characterIndex {
	indexed := characterIndex{byID: make(map[string]aggregateCharacterState)}
	if len(state.Characters) == 0 {
		return indexed
	}

	cids := make([]string, 0, len(state.Characters))
	for characterID := range state.Characters {
		cids = append(cids, string(characterID))
	}
	sort.Strings(cids)

	for _, characterID := range cids {
		characterState := state.Characters[domainids.CharacterID(characterID)]
		if !characterState.Created || characterState.Deleted {
			continue
		}
		indexed.byID[characterID] = aggregateCharacterState{
			ParticipantID: string(characterState.ParticipantID),
			Name:          characterState.Name,
		}
		indexed.ids = append(indexed.ids, characterID)
	}

	return indexed
}
