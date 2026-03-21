package gate

import (
	"testing"
	"time"
)

func TestGenericGateWorkflowHelpers(t *testing.T) {
	t.Parallel()

	workflow, err := NewGenericGateWorkflow(map[string]any{
		WorkflowEligibleParticipantIDsKey: []any{" p2 ", "p1", "p1"},
		WorkflowResponseAuthorityKey:      " participant ",
		"note":                            "danger",
	})
	if err != nil {
		t.Fatalf("NewGenericGateWorkflow error = %v", err)
	}

	progress := &GateProgress{}
	workflow.ApplyProgressMetadata(progress)
	if got := progress.ResponseAuthority; got != GateResponseAuthorityParticipant {
		t.Fatalf("response authority = %q", got)
	}
	if len(progress.EligibleParticipantIDs) != 2 || progress.EligibleParticipantIDs[0] != "p1" || progress.EligibleParticipantIDs[1] != "p2" {
		t.Fatalf("eligible ids = %#v", progress.EligibleParticipantIDs)
	}

	if err := workflow.ValidateParticipant("p1"); err != nil {
		t.Fatalf("ValidateParticipant eligible error = %v", err)
	}
	if err := workflow.ValidateParticipant("p9"); err == nil {
		t.Fatal("expected ineligible participant rejection")
	}

	decision, response, err := workflow.ValidateResponse(" yes ", map[string]any{"detail": "ok"})
	if err != nil {
		t.Fatalf("ValidateResponse error = %v", err)
	}
	if decision != "yes" || response["detail"] != "ok" {
		t.Fatalf("normalized response = %q %#v", decision, response)
	}
	if _, _, err := workflow.ValidateResponse("", nil); err == nil {
		t.Fatal("expected empty response rejection")
	}

	progress.SuggestedDecision = "unchanged"
	workflow.DeriveResolution(progress)
	if progress.SuggestedDecision != "unchanged" {
		t.Fatalf("DeriveResolution mutated progress = %#v", progress)
	}
}

func TestGateWorkflowParsingHelpers(t *testing.T) {
	t.Parallel()

	metadata, err := WorkflowMetadataFromJSON([]byte(`{"eligible_participant_ids":[" p2 ","p1"],"response_authority":"participant","extra":true}`))
	if err != nil {
		t.Fatalf("WorkflowMetadataFromJSON error = %v", err)
	}
	base, err := ParseGateWorkflowBase(metadata, "")
	if err != nil {
		t.Fatalf("ParseGateWorkflowBase error = %v", err)
	}
	if base.ResponseAuthority != GateResponseAuthorityParticipant {
		t.Fatalf("response authority = %q", base.ResponseAuthority)
	}
	if len(base.EligibleParticipantIDs) != 2 || base.EligibleParticipantIDs[0] != "p1" || base.EligibleParticipantIDs[1] != "p2" {
		t.Fatalf("eligible ids = %#v", base.EligibleParticipantIDs)
	}
	if base.ExtraMetadata["extra"] != true {
		t.Fatalf("extra metadata = %#v", base.ExtraMetadata)
	}

	if _, err := WorkflowResponseAuthority(7, ""); err == nil {
		t.Fatal("expected non-string response authority rejection")
	}
	if _, err := WorkflowStringSlice([]any{"ok", 7}, "eligible_participant_ids"); err == nil {
		t.Fatal("expected mixed-type slice rejection")
	}
	if _, err := WorkflowStringSlice("bad", "eligible_participant_ids"); err == nil {
		t.Fatal("expected non-slice rejection")
	}
	if values := WorkflowUniqueStrings([]string{" p2 ", "p1", "p1", ""}); len(values) != 2 || values[0] != "p1" || values[1] != "p2" {
		t.Fatalf("unique values = %#v", values)
	}
	if !WorkflowContains([]string{" p1 "}, "p1") {
		t.Fatal("expected contains match")
	}
}

func TestGateProjectionHelpers(t *testing.T) {
	t.Parallel()

	metadata := map[string]any{
		WorkflowEligibleParticipantIDsKey: []string{"p1", "p2"},
		WorkflowResponseAuthorityKey:      GateResponseAuthorityParticipant,
		"note":                            "danger",
	}

	metadataJSON, err := MarshalGateMetadataJSON("", metadata)
	if err != nil {
		t.Fatalf("MarshalGateMetadataJSON error = %v", err)
	}
	decodedMetadata, err := DecodeGateMetadataMap("", metadataJSON)
	if err != nil {
		t.Fatalf("DecodeGateMetadataMap error = %v", err)
	}
	if decodedMetadata["note"] != "danger" {
		t.Fatalf("decoded metadata = %#v", decodedMetadata)
	}
	if _, _, err := ValidateGateResponseMetadata("", metadata, "p1", "yes", map[string]any{"detail": "ok"}); err != nil {
		t.Fatalf("ValidateGateResponseMetadata error = %v", err)
	}

	storedMetadata, err := BuildStoredGateMetadata("", metadata)
	if err != nil {
		t.Fatalf("BuildStoredGateMetadata error = %v", err)
	}
	if _, err := BuildGateMetadataMapFromStored("", storedMetadata); err != nil {
		t.Fatalf("BuildGateMetadataMapFromStored error = %v", err)
	}

	progress, err := BuildInitialGateProgressState("", metadata)
	if err != nil {
		t.Fatalf("BuildInitialGateProgressState error = %v", err)
	}
	if progress.ResponseAuthority != GateResponseAuthorityParticipant {
		t.Fatalf("initial progress = %#v", progress)
	}
	payload := GateResponseRecordedPayload{ParticipantID: "p1", Decision: "yes", Response: map[string]any{"detail": "ok"}}
	progress, err = RecordGateResponseProgressState("", metadata, progress, payload, time.Date(2026, 3, 13, 12, 0, 0, 0, time.UTC), "participant", "p1")
	if err != nil {
		t.Fatalf("RecordGateResponseProgressState error = %v", err)
	}
	if len(progress.Responses) != 1 {
		t.Fatalf("responses = %#v", progress.Responses)
	}
	if _, err := DecodeGateProgressMap("", metadataJSON, []byte(`{`)); err == nil {
		t.Fatal("expected invalid progress json rejection")
	}
	if _, err := BuildGateProgressFromResponses("", metadata, progress.Responses); err != nil {
		t.Fatalf("BuildGateProgressFromResponses error = %v", err)
	}

	resolution, err := BuildGateResolutionMap(" accepted ", map[string]any{"detail": "ok"})
	if err != nil {
		t.Fatalf("BuildGateResolutionMap error = %v", err)
	}
	if resolution["decision"] != "accepted" {
		t.Fatalf("resolution = %#v", resolution)
	}
	if _, err := MarshalGateResolutionMapJSON(resolution); err != nil {
		t.Fatalf("MarshalGateResolutionMapJSON error = %v", err)
	}
	storedResolution, err := BuildStoredGateResolution("accepted", map[string]any{"detail": "ok"})
	if err != nil {
		t.Fatalf("BuildStoredGateResolution error = %v", err)
	}
	if storedResolution.Decision != "accepted" || storedResolution.Extra["detail"] != "ok" {
		t.Fatalf("stored resolution = %#v", storedResolution)
	}
	if _, err := BuildGateResolutionMapFromStored(storedResolution.Decision, storedResolution.Extra); err != nil {
		t.Fatalf("BuildGateResolutionMapFromStored error = %v", err)
	}
}
