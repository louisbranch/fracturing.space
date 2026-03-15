package session

import (
	"fmt"
	"strings"
)

// gateWorkflow captures typed behavior for one workflow family while keeping
// the transport-facing contract generic at the package boundary.
type gateWorkflow interface {
	normalizedMetadata() map[string]any
	applyProgressMetadata(*GateProgress)
	validateParticipant(participantID string) error
	validateResponse(decision string, response map[string]any) (string, map[string]any, error)
	deriveResolution(*GateProgress)
}

// gateWorkflowBase holds workflow metadata shared by all gate types so
// ready-check, vote, and generic flows do not repeatedly parse raw maps.
type gateWorkflowBase struct {
	extraMetadata          map[string]any
	eligibleParticipantIDs []string
	responseAuthority      string
}

func metadataValue(metadata map[string]any, key string) any {
	if metadata == nil {
		return nil
	}
	return metadata[key]
}

func (w gateWorkflowBase) normalizedMetadata() map[string]any {
	values := gateWorkflowCloneMap(w.extraMetadata)
	if len(w.eligibleParticipantIDs) > 0 {
		values[gateWorkflowEligibleParticipantIDsKey] = append([]string(nil), w.eligibleParticipantIDs...)
	}
	if strings.TrimSpace(w.responseAuthority) != "" {
		values[gateWorkflowResponseAuthorityKey] = w.responseAuthority
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func (w gateWorkflowBase) applyProgressMetadata(progress *GateProgress) {
	if progress == nil {
		return
	}
	progress.EligibleParticipantIDs = append([]string(nil), w.eligibleParticipantIDs...)
	progress.ResponseAuthority = w.responseAuthority
}

func (w gateWorkflowBase) validateParticipant(participantID string) error {
	participantID = strings.TrimSpace(participantID)
	if participantID == "" {
		return fmt.Errorf("participant id is required")
	}
	switch w.responseAuthority {
	case "", GateResponseAuthorityParticipant:
	default:
		return fmt.Errorf("unsupported gate response authority %q", w.responseAuthority)
	}
	if len(w.eligibleParticipantIDs) > 0 && !gateWorkflowContains(w.eligibleParticipantIDs, participantID) {
		return fmt.Errorf("participant %q is not eligible for this gate", participantID)
	}
	return nil
}

func normalizeOptionalGateResponse(response map[string]any) map[string]any {
	if len(response) == 0 {
		return nil
	}
	return response
}
