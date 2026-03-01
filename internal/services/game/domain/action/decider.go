package action

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

const (
	CommandTypeRollResolve   command.Type = "action.roll.resolve"
	CommandTypeOutcomeApply  command.Type = "action.outcome.apply"
	CommandTypeOutcomeReject command.Type = "action.outcome.reject"
	CommandTypeNoteAdd       command.Type = "story.note.add"

	EventTypeRollResolved    event.Type = "action.roll_resolved"
	EventTypeOutcomeApplied  event.Type = "action.outcome_applied"
	EventTypeOutcomeRejected event.Type = "action.outcome_rejected"
	EventTypeNoteAdded       event.Type = "story.note_added"

	rejectionCodeRequestIDRequired                 = "REQUEST_ID_REQUIRED"
	rejectionCodeRollSeqRequired                   = "ROLL_SEQ_REQUIRED"
	rejectionCodeOutcomeAlreadyApplied             = "OUTCOME_ALREADY_APPLIED"
	rejectionCodeOutcomeEffectSystemOwnedForbidden = "OUTCOME_EFFECT_SYSTEM_OWNED_FORBIDDEN"
	rejectionCodeOutcomeEffectTypeForbidden        = "OUTCOME_EFFECT_TYPE_FORBIDDEN"
)

var coreOutcomeEffectPolicy = newOutcomeEffectPolicy("session.gate_opened", "session.spotlight_set")

// Decide returns the decision for an action command against current state.
//
// The action aggregate is intentionally lightweight: each supported action command
// becomes a typed domain event, keeping roll outcome logic and note-taking in one
// replayable stream.
func Decide(state State, cmd command.Command, now func() time.Time) command.Decision {
	if now == nil {
		now = time.Now
	}

	switch cmd.Type {
	case CommandTypeRollResolve:
		return decideRollResolve(cmd, now)
	case CommandTypeOutcomeApply:
		return decideOutcomeApply(state, cmd, now)
	case CommandTypeOutcomeReject:
		return decideOutcomeReject(cmd, now)
	case CommandTypeNoteAdd:
		return decideNoteAdd(cmd, now)
	default:
		return command.Reject(command.Rejection{
			Code:    "COMMAND_TYPE_UNSUPPORTED",
			Message: "command type is not supported by action decider",
		})
	}
}

func decideRollResolve(cmd command.Command, now func() time.Time) command.Decision {
	var payload RollResolvePayload
	_ = json.Unmarshal(cmd.PayloadJSON, &payload)
	requestID := strings.TrimSpace(payload.RequestID)
	if requestID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeRequestIDRequired,
			Message: "request_id is required",
		})
	}
	if payload.RollSeq == 0 {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeRollSeqRequired,
			Message: "roll_seq must be greater than zero",
		})
	}
	return acceptActionEvent(cmd, now, EventTypeRollResolved, "roll", requestID, payload)
}

func decideOutcomeApply(state State, cmd command.Command, now func() time.Time) command.Decision {
	var payload OutcomeApplyPayload
	_ = json.Unmarshal(cmd.PayloadJSON, &payload)
	requestID := strings.TrimSpace(payload.RequestID)
	if requestID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeRequestIDRequired,
			Message: "request_id is required",
		})
	}
	if payload.RollSeq == 0 {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeRollSeqRequired,
			Message: "roll_seq must be greater than zero",
		})
	}
	if hasSystemOwnedOutcomeEffect(payload.PreEffects) || hasSystemOwnedOutcomeEffect(payload.PostEffects) {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeOutcomeEffectSystemOwnedForbidden,
			Message: "core action.outcome.apply cannot emit system-owned effects",
		})
	}
	if hasDisallowedCoreOutcomeEffect(payload.PreEffects) || hasDisallowedCoreOutcomeEffect(payload.PostEffects) {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeOutcomeEffectTypeForbidden,
			Message: "core action.outcome.apply effect type is not allowed",
		})
	}
	if _, alreadyApplied := state.AppliedOutcomes[payload.RollSeq]; alreadyApplied {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeOutcomeAlreadyApplied,
			Message: ErrOutcomeAlreadyApplied.Error(),
		})
	}

	events := make([]event.Event, 0, len(payload.PreEffects)+len(payload.PostEffects)+1)
	for _, effect := range payload.PreEffects {
		events = append(events, buildOutcomeEffectEvent(cmd, now, effect))
	}

	postEffects := payload.PostEffects
	payload.PreEffects = nil
	payload.PostEffects = nil
	events = append(events, acceptActionEvent(cmd, now, EventTypeOutcomeApplied, "outcome", requestID, payload).Events...)

	for _, effect := range postEffects {
		events = append(events, buildOutcomeEffectEvent(cmd, now, effect))
	}
	return command.Accept(events...)
}

func decideOutcomeReject(cmd command.Command, now func() time.Time) command.Decision {
	var payload OutcomeRejectPayload
	_ = json.Unmarshal(cmd.PayloadJSON, &payload)
	requestID := strings.TrimSpace(payload.RequestID)
	if requestID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeRequestIDRequired,
			Message: "request_id is required",
		})
	}
	if payload.RollSeq == 0 {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeRollSeqRequired,
			Message: "roll_seq must be greater than zero",
		})
	}
	return acceptActionEvent(cmd, now, EventTypeOutcomeRejected, "outcome", requestID, payload)
}

func decideNoteAdd(cmd command.Command, now func() time.Time) command.Decision {
	var payload NoteAddPayload
	_ = json.Unmarshal(cmd.PayloadJSON, &payload)
	return acceptActionEvent(cmd, now, EventTypeNoteAdded, "note", cmd.EntityID, payload)
}

// acceptActionEvent creates the standard action event envelope for accepted commands.
//
// Centralizing this constructor keeps action event metadata consistent even when
// specific systems add new action shapes.
func acceptActionEvent(cmd command.Command, now func() time.Time, eventType event.Type, entityType, entityID string, payload any) command.Decision {
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, eventType, entityType, entityID, payloadJSON, now().UTC())

	return command.Accept(evt)
}

func buildOutcomeEffectEvent(cmd command.Command, now func() time.Time, effect OutcomeAppliedEffect) event.Event {
	payloadJSON := effect.PayloadJSON
	if len(payloadJSON) == 0 {
		payloadJSON = []byte("{}")
	}
	evt := command.NewEvent(
		cmd,
		event.Type(strings.TrimSpace(effect.Type)),
		strings.TrimSpace(effect.EntityType),
		strings.TrimSpace(effect.EntityID),
		payloadJSON,
		now().UTC(),
	)
	evt.SystemID = strings.TrimSpace(effect.SystemID)
	evt.SystemVersion = strings.TrimSpace(effect.SystemVersion)
	return evt
}

type outcomeEffectPolicy struct {
	allowed map[string]struct{}
}

func newOutcomeEffectPolicy(allowed ...string) outcomeEffectPolicy {
	policy := outcomeEffectPolicy{allowed: make(map[string]struct{}, len(allowed))}
	for _, effectType := range allowed {
		policy.allowed[strings.TrimSpace(effectType)] = struct{}{}
	}
	return policy
}

func (p outcomeEffectPolicy) hasSystemOwned(effects []OutcomeAppliedEffect) bool {
	for _, effect := range effects {
		if strings.HasPrefix(strings.TrimSpace(effect.Type), "sys.") {
			return true
		}
		if strings.TrimSpace(effect.SystemID) != "" || strings.TrimSpace(effect.SystemVersion) != "" {
			return true
		}
	}
	return false
}

func (p outcomeEffectPolicy) hasDisallowed(effects []OutcomeAppliedEffect) bool {
	for _, effect := range effects {
		if !p.isAllowed(strings.TrimSpace(effect.Type)) {
			return true
		}
	}
	return false
}

func (p outcomeEffectPolicy) isAllowed(effectType string) bool {
	_, ok := p.allowed[strings.TrimSpace(effectType)]
	return ok
}

func hasSystemOwnedOutcomeEffect(effects []OutcomeAppliedEffect) bool {
	return coreOutcomeEffectPolicy.hasSystemOwned(effects)
}

func hasDisallowedCoreOutcomeEffect(effects []OutcomeAppliedEffect) bool {
	return coreOutcomeEffectPolicy.hasDisallowed(effects)
}

func isAllowedCoreOutcomeEffectType(effectType string) bool {
	return coreOutcomeEffectPolicy.isAllowed(effectType)
}
