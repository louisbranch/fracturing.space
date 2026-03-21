package gate

import (
	"fmt"
	"strings"
)

// GateWorkflow captures typed behavior for one workflow family while keeping
// the transport-facing contract generic at the package boundary.
type GateWorkflow interface {
	NormalizedMetadata() map[string]any
	ApplyProgressMetadata(*GateProgress)
	ValidateParticipant(participantID string) error
	ValidateResponse(decision string, response map[string]any) (string, map[string]any, error)
	DeriveResolution(*GateProgress)
}

// GateWorkflowBase holds workflow metadata shared by gate types so generic
// gate handling does not repeatedly parse raw maps.
type GateWorkflowBase struct {
	ExtraMetadata          map[string]any
	EligibleParticipantIDs []string
	ResponseAuthority      string
}

func MetadataValue(metadata map[string]any, key string) any {
	if metadata == nil {
		return nil
	}
	return metadata[key]
}

func (w GateWorkflowBase) NormalizedMetadata() map[string]any {
	values := WorkflowCloneMap(w.ExtraMetadata)
	if len(w.EligibleParticipantIDs) > 0 {
		values[WorkflowEligibleParticipantIDsKey] = append([]string(nil), w.EligibleParticipantIDs...)
	}
	if strings.TrimSpace(w.ResponseAuthority) != "" {
		values[WorkflowResponseAuthorityKey] = w.ResponseAuthority
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func (w GateWorkflowBase) ApplyProgressMetadata(progress *GateProgress) {
	if progress == nil {
		return
	}
	progress.EligibleParticipantIDs = append([]string(nil), w.EligibleParticipantIDs...)
	progress.ResponseAuthority = w.ResponseAuthority
}

func (w GateWorkflowBase) ValidateParticipant(participantID string) error {
	participantID = strings.TrimSpace(participantID)
	if participantID == "" {
		return fmt.Errorf("participant id is required")
	}
	switch w.ResponseAuthority {
	case "", GateResponseAuthorityParticipant:
	default:
		return fmt.Errorf("unsupported gate response authority %q", w.ResponseAuthority)
	}
	if len(w.EligibleParticipantIDs) > 0 && !WorkflowContains(w.EligibleParticipantIDs, participantID) {
		return fmt.Errorf("participant %q is not eligible for this gate", participantID)
	}
	return nil
}

func NormalizeOptionalGateResponse(response map[string]any) map[string]any {
	if len(response) == 0 {
		return nil
	}
	return response
}
