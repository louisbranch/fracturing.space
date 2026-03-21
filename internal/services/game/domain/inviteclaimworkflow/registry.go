package inviteclaimworkflow

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

const CommandTypeClaimBind command.Type = "invite.claim_bind"

const (
	rejectionCodeBindEventMissing  = "INVITE_CLAIM_WORKFLOW_BIND_EVENT_MISSING"
	rejectionCodeClaimEventMissing = "INVITE_CLAIM_WORKFLOW_CLAIM_EVENT_MISSING"
)

// RegisterCommands registers the invite claim workflow command with the shared
// registry.
func RegisterCommands(registry *command.Registry) error {
	if registry == nil {
		return errors.New("command registry is required")
	}
	return registry.Register(command.Definition{
		Type:            CommandTypeClaimBind,
		Owner:           command.OwnerCore,
		ValidatePayload: validateClaimBindPayload,
		ActiveSession:   command.BlockedDuringActiveSession(),
		Target:          command.TargetEntity("invite", "invite_id"),
	})
}

// RegisterEvents is a no-op because this workflow emits only participant and
// invite aggregate events, which stay registered by their owning domains.
func RegisterEvents(registry *event.Registry) error {
	if registry == nil {
		return errors.New("event registry is required")
	}
	return nil
}

// EmittableEventTypes returns nil because this workflow introduces no unique
// event types of its own.
func EmittableEventTypes() []event.Type { return nil }

// FoldHandledTypes returns nil because this workflow owns no fold handlers.
func FoldHandledTypes() []event.Type { return nil }

// DeciderHandledCommands returns the workflow command handled by this package.
func DeciderHandledCommands() []command.Type { return []command.Type{CommandTypeClaimBind} }

// ProjectionHandledTypes returns nil because this workflow owns no projection
// handlers.
func ProjectionHandledTypes() []event.Type { return nil }

// RejectionCodes returns the workflow-specific rejection codes introduced by
// this package.
func RejectionCodes() []string {
	return []string{rejectionCodeBindEventMissing, rejectionCodeClaimEventMissing}
}

func validateClaimBindPayload(raw json.RawMessage) error {
	var payload ClaimBindPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(string(payload.InviteID)) == "" {
		return errors.New("invite_id is required")
	}
	if strings.TrimSpace(string(payload.ParticipantID)) == "" {
		return errors.New("participant_id is required")
	}
	if strings.TrimSpace(string(payload.UserID)) == "" {
		return errors.New("user_id is required")
	}
	if strings.TrimSpace(payload.JWTID) == "" {
		return errors.New("jti is required")
	}
	return nil
}
