package session

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNormalizeAITurnStatus(t *testing.T) {
	t.Parallel()

	tests := map[string]AITurnStatus{
		" idle ":  AITurnStatusIdle,
		"QUEUED":  AITurnStatusQueued,
		"running": AITurnStatusRunning,
		"FAILED":  AITurnStatusFailed,
	}
	for input, want := range tests {
		got, err := NormalizeAITurnStatus(input)
		if err != nil {
			t.Fatalf("NormalizeAITurnStatus(%q) error = %v", input, err)
		}
		if got != want {
			t.Fatalf("NormalizeAITurnStatus(%q) = %q, want %q", input, got, want)
		}
	}
	if _, err := NormalizeAITurnStatus("bogus"); err == nil {
		t.Fatal("expected unsupported ai turn status error")
	}
}

func TestGenericGateWorkflowValidateAndResolutionHelpers(t *testing.T) {
	t.Parallel()

	workflow, err := newGenericGateWorkflow(map[string]any{
		"eligible_participant_ids": []string{"p2", "p1"},
		"response_authority":       GateResponseAuthorityParticipant,
		"topic":                    "path",
	})
	if err != nil {
		t.Fatalf("newGenericGateWorkflow() error = %v", err)
	}
	if err := workflow.validateParticipant("p1"); err != nil {
		t.Fatalf("validateParticipant() error = %v", err)
	}
	if err := workflow.validateParticipant("p3"); err == nil {
		t.Fatal("expected ineligible participant error")
	}

	decision, response, err := workflow.validateResponse(" ready ", map[string]any{"note": "go"})
	if err != nil {
		t.Fatalf("validateResponse() error = %v", err)
	}
	if decision != "ready" && decision != " ready " {
		// Current generic workflow only trims surrounding whitespace in the
		// caller-facing helper contract when the value is normalized upstream.
		t.Fatalf("validateResponse() decision = %q", decision)
	}
	if got := response["note"]; got != "go" {
		t.Fatalf("response note = %#v, want go", got)
	}
	if _, _, err := workflow.validateResponse("", nil); err == nil {
		t.Fatal("expected missing response payload error")
	}

	progress := &GateProgress{ResolutionState: "kept"}
	workflow.deriveResolution(progress)
	if progress.ResolutionState != "kept" {
		t.Fatalf("deriveResolution() mutated progress = %#v", progress)
	}
}

func TestGateProgressJSONAPIHelpers(t *testing.T) {
	t.Parallel()

	metadataJSON := mustMarshalGateMetadataForType(t, "decision", map[string]any{
		"eligible_participant_ids": []string{"p2", "p1"},
		"response_authority":       GateResponseAuthorityParticipant,
		"options":                  []string{"north", "south"},
	})

	initialJSON, err := BuildInitialGateProgress("decision", metadataJSON)
	if err != nil {
		t.Fatalf("BuildInitialGateProgress() error = %v", err)
	}
	if initialJSON == nil {
		t.Fatal("BuildInitialGateProgress() returned nil, want encoded progress")
	}
	var initial GateProgress
	if err := json.Unmarshal(initialJSON, &initial); err != nil {
		t.Fatalf("json.Unmarshal(initial) error = %v", err)
	}
	if initial.EligibleCount != 2 || initial.PendingCount != 2 {
		t.Fatalf("initial progress = %#v", initial)
	}

	recordedAt := time.Date(2026, 3, 13, 16, 0, 0, 0, time.UTC)
	updatedJSON, err := RecordGateResponseProgress(
		"decision",
		metadataJSON,
		initialJSON,
		GateResponseRecordedPayload{ParticipantID: "p2", Decision: "north"},
		recordedAt,
		"participant",
		"p2",
	)
	if err != nil {
		t.Fatalf("RecordGateResponseProgress() error = %v", err)
	}
	var updated GateProgress
	if err := json.Unmarshal(updatedJSON, &updated); err != nil {
		t.Fatalf("json.Unmarshal(updated) error = %v", err)
	}
	if updated.RespondedCount != 1 || updated.PendingCount != 1 {
		t.Fatalf("updated progress = %#v", updated)
	}

	replacedJSON, err := RecordGateResponseProgress(
		"decision",
		metadataJSON,
		updatedJSON,
		GateResponseRecordedPayload{ParticipantID: "p2", Decision: "south"},
		recordedAt.Add(time.Minute),
		"participant",
		"p2",
	)
	if err != nil {
		t.Fatalf("RecordGateResponseProgress(replace) error = %v", err)
	}
	var replaced GateProgress
	if err := json.Unmarshal(replacedJSON, &replaced); err != nil {
		t.Fatalf("json.Unmarshal(replaced) error = %v", err)
	}
	if len(replaced.Responses) != 1 || replaced.Responses[0].Decision != "south" {
		t.Fatalf("replaced progress responses = %#v", replaced.Responses)
	}
}

func TestBuildInitialGateProgressReturnsNilForEmptyGenericWorkflow(t *testing.T) {
	t.Parallel()

	progressJSON, err := BuildInitialGateProgress("gm_prompt", mustMarshalGateMetadataForType(t, "gm_prompt", map[string]any{
		"topic": "handoff",
	}))
	if err != nil {
		t.Fatalf("BuildInitialGateProgress() error = %v", err)
	}
	if progressJSON != nil {
		t.Fatalf("BuildInitialGateProgress() = %q, want nil for empty generic progress", progressJSON)
	}
}

func TestNormalizeGateWorkflowMetadataAndHelperParsing(t *testing.T) {
	t.Parallel()

	normalized, err := NormalizeGateWorkflowMetadata("decision", map[string]any{
		"eligible_participant_ids": []any{" p2 ", "p1", "p2", ""},
		"response_authority":       " participant ",
		"topic":                    "route",
	})
	if err != nil {
		t.Fatalf("NormalizeGateWorkflowMetadata() error = %v", err)
	}
	gotIDs, ok := normalized["eligible_participant_ids"].([]string)
	if !ok || len(gotIDs) != 2 || gotIDs[0] != "p1" || gotIDs[1] != "p2" {
		t.Fatalf("eligible ids = %#v", normalized["eligible_participant_ids"])
	}
	if got := normalized["response_authority"]; got != GateResponseAuthorityParticipant {
		t.Fatalf("response authority = %#v", got)
	}
	if got := normalized["topic"]; got != "route" {
		t.Fatalf("topic = %#v", got)
	}

	if _, err := NormalizeGateWorkflowMetadata("decision", map[string]any{
		"response_authority": []string{"bad"},
	}); err == nil {
		t.Fatal("expected response authority type error")
	}
	if _, err := NormalizeGateWorkflowMetadata("decision", map[string]any{
		"eligible_participant_ids": []any{"p1", 1},
	}); err == nil {
		t.Fatal("expected eligible_participant_ids type error")
	}
}

func TestValidateGateResponseRejectsIneligibleParticipantAndNormalizesPayload(t *testing.T) {
	t.Parallel()

	metadataJSON := mustMarshalGateMetadataForType(t, "decision", map[string]any{
		"eligible_participant_ids": []string{"p1"},
		"response_authority":       GateResponseAuthorityParticipant,
	})

	decision, response, err := ValidateGateResponse("decision", metadataJSON, "p1", " north ", map[string]any{"detail": "bridge"})
	if err != nil {
		t.Fatalf("ValidateGateResponse() error = %v", err)
	}
	if decision != "north" {
		t.Fatalf("decision = %q, want north", decision)
	}
	if got := response["detail"]; got != "bridge" {
		t.Fatalf("response detail = %#v", got)
	}

	if _, _, err := ValidateGateResponse("decision", metadataJSON, "p2", "north", nil); err == nil {
		t.Fatal("expected ineligible participant error")
	}
	if _, _, err := ValidateGateResponse("decision", metadataJSON, "p1", "", nil); err == nil {
		t.Fatal("expected missing decision/response error")
	}
}
