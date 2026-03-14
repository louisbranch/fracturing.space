package session

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func TestNormalizeGateWorkflowMetadataReadyCheckAddsDefaultOptions(t *testing.T) {
	t.Parallel()

	metadata, err := NormalizeGateWorkflowMetadata(GateTypeReadyCheck, nil)
	if err != nil {
		t.Fatalf("NormalizeGateWorkflowMetadata() error = %v", err)
	}
	options, ok := metadata["options"].([]string)
	if !ok {
		t.Fatalf("options type = %T, want []string", metadata["options"])
	}
	if len(options) != 2 || options[0] != "ready" || options[1] != "wait" {
		t.Fatalf("options = %#v", options)
	}
	if got := metadata["response_authority"]; got != GateResponseAuthorityParticipant {
		t.Fatalf("response_authority = %v, want %q", got, GateResponseAuthorityParticipant)
	}
}

func TestNormalizeGateWorkflowMetadataVoteDefaultsParticipantAuthority(t *testing.T) {
	t.Parallel()

	metadata, err := NormalizeGateWorkflowMetadata(GateTypeVote, nil)
	if err != nil {
		t.Fatalf("NormalizeGateWorkflowMetadata() error = %v", err)
	}
	if got := metadata["response_authority"]; got != GateResponseAuthorityParticipant {
		t.Fatalf("response_authority = %v, want %q", got, GateResponseAuthorityParticipant)
	}
}

func TestValidateGateResponseVoteHonorsEligibleParticipantsAndOptions(t *testing.T) {
	t.Parallel()

	metadataJSON, err := json.Marshal(map[string]any{
		"eligible_participant_ids": []string{"p1", "p2"},
		"options":                  []string{"north", "south"},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	if _, _, err := ValidateGateResponse(GateTypeVote, metadataJSON, "p3", "north", nil); err == nil {
		t.Fatal("expected ineligible participant error")
	}
	if _, _, err := ValidateGateResponse(GateTypeVote, metadataJSON, "p1", "east", nil); err == nil {
		t.Fatal("expected invalid option error")
	}
	decision, response, err := ValidateGateResponse(GateTypeVote, metadataJSON, "p1", "north", map[string]any{"note": "bridge"})
	if err != nil {
		t.Fatalf("ValidateGateResponse() error = %v", err)
	}
	if decision != "north" {
		t.Fatalf("decision = %q, want %q", decision, "north")
	}
	if response["note"] != "bridge" {
		t.Fatalf("response = %#v", response)
	}
}

func TestNormalizeGateWorkflowMetadataRejectsInvalidWorkflowOptions(t *testing.T) {
	t.Parallel()

	if _, err := NormalizeGateWorkflowMetadata(GateTypeReadyCheck, map[string]any{
		"options": []string{"ready"},
	}); err == nil {
		t.Fatal("expected ready_check options validation error")
	}
	if _, err := NormalizeGateWorkflowMetadata(GateTypeVote, map[string]any{
		"options": []string{"north"},
	}); err == nil {
		t.Fatal("expected vote options validation error")
	}
}

func TestNormalizeGateWorkflowMetadataNormalizesEligibleIDsAndPreservesUnknownKeys(t *testing.T) {
	t.Parallel()

	metadata, err := NormalizeGateWorkflowMetadata(GateTypeVote, map[string]any{
		"eligible_participant_ids": []any{"p2", "p1", "p2", " "},
		"options":                  []any{"south", "north", "south"},
		"response_authority":       "participant",
		"audience":                 "table",
	})
	if err != nil {
		t.Fatalf("NormalizeGateWorkflowMetadata() error = %v", err)
	}
	if got := metadata["eligible_participant_ids"]; !gateWorkflowContains(got.([]string), "p1") || !gateWorkflowContains(got.([]string), "p2") {
		t.Fatalf("eligible_participant_ids = %#v", got)
	}
	if got := metadata["options"]; len(got.([]string)) != 2 || got.([]string)[0] != "north" || got.([]string)[1] != "south" {
		t.Fatalf("options = %#v", got)
	}
	if got := metadata["audience"]; got != "table" {
		t.Fatalf("audience = %#v, want table", got)
	}
}

func TestNormalizeGateWorkflowMetadataRejectsInvalidWorkflowMetadataShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		gateType string
		metadata map[string]any
	}{
		{
			name:     "eligible ids must be strings",
			gateType: GateTypeVote,
			metadata: map[string]any{"eligible_participant_ids": []any{"p1", 2}},
		},
		{
			name:     "response authority must be string",
			gateType: GateTypeVote,
			metadata: map[string]any{"response_authority": 1},
		},
		{
			name:     "response authority must be supported",
			gateType: GateTypeVote,
			metadata: map[string]any{"response_authority": "persona"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NormalizeGateWorkflowMetadata(tc.gateType, tc.metadata); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestRecordGateResponseProgressReplacesParticipantResponseAndRecomputesCounts(t *testing.T) {
	t.Parallel()

	metadataJSON, err := json.Marshal(map[string]any{
		"eligible_participant_ids": []string{"p1", "p2"},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	progressJSON, err := BuildInitialGateProgress(GateTypeReadyCheck, metadataJSON)
	if err != nil {
		t.Fatalf("BuildInitialGateProgress() error = %v", err)
	}

	progressJSON, err = RecordGateResponseProgress(
		GateTypeReadyCheck,
		metadataJSON,
		progressJSON,
		GateResponseRecordedPayload{
			GateID:        ids.GateID("gate-1"),
			ParticipantID: ids.ParticipantID("p1"),
			Decision:      "ready",
		},
		time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC),
		"participant",
		"p1",
	)
	if err != nil {
		t.Fatalf("RecordGateResponseProgress(first) error = %v", err)
	}
	progressJSON, err = RecordGateResponseProgress(
		GateTypeReadyCheck,
		metadataJSON,
		progressJSON,
		GateResponseRecordedPayload{
			GateID:        ids.GateID("gate-1"),
			ParticipantID: ids.ParticipantID("p1"),
			Decision:      "wait",
		},
		time.Date(2026, 3, 9, 12, 1, 0, 0, time.UTC),
		"participant",
		"p1",
	)
	if err != nil {
		t.Fatalf("RecordGateResponseProgress(second) error = %v", err)
	}

	var progress GateProgress
	if err := json.Unmarshal(progressJSON, &progress); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if progress.RespondedCount != 1 || progress.EligibleCount != 2 || progress.PendingCount != 1 || progress.AllResponded {
		t.Fatalf("progress counts = %#v", progress)
	}
	if progress.ResponseAuthority != GateResponseAuthorityParticipant {
		t.Fatalf("response authority = %q, want %q", progress.ResponseAuthority, GateResponseAuthorityParticipant)
	}
	if len(progress.PendingParticipantIDs) != 1 || progress.PendingParticipantIDs[0] != "p2" {
		t.Fatalf("pending participants = %#v", progress.PendingParticipantIDs)
	}
	if len(progress.Responses) != 1 || progress.Responses[0].Decision != "wait" {
		t.Fatalf("progress responses = %#v", progress.Responses)
	}
	if progress.DecisionCounts["wait"] != 1 {
		t.Fatalf("decision_counts = %#v", progress.DecisionCounts)
	}
	if progress.ReadyCount != 0 || progress.WaitCount != 1 || progress.AllReady {
		t.Fatalf("ready summary = %#v", progress)
	}
	if progress.ResolutionState != GateResolutionStateBlocked || progress.ResolutionReason != "wait_response_present" || progress.SuggestedDecision != "wait" {
		t.Fatalf("resolution summary = %#v", progress)
	}
}

func TestBuildInitialGateProgressVoteIncludesResponseAuthorityWithoutOptions(t *testing.T) {
	t.Parallel()

	progressJSON, err := BuildInitialGateProgress(GateTypeVote, nil)
	if err != nil {
		t.Fatalf("BuildInitialGateProgress() error = %v", err)
	}
	if len(progressJSON) == 0 {
		t.Fatal("expected vote progress JSON")
	}

	var progress GateProgress
	if err := json.Unmarshal(progressJSON, &progress); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if progress.WorkflowType != GateTypeVote {
		t.Fatalf("workflow type = %q, want %q", progress.WorkflowType, GateTypeVote)
	}
	if progress.ResponseAuthority != GateResponseAuthorityParticipant {
		t.Fatalf("response authority = %q, want %q", progress.ResponseAuthority, GateResponseAuthorityParticipant)
	}
	if progress.ResolutionState != GateResolutionStatePendingResponses || progress.ResolutionReason != "waiting_on_participants" {
		t.Fatalf("resolution summary = %#v", progress)
	}
}

func TestBuildInitialGateProgressUnknownWorkflowReturnsNil(t *testing.T) {
	t.Parallel()

	progressJSON, err := BuildInitialGateProgress("choice", nil)
	if err != nil {
		t.Fatalf("BuildInitialGateProgress() error = %v", err)
	}
	if len(progressJSON) != 0 {
		t.Fatalf("progress JSON = %s, want empty", string(progressJSON))
	}
}

func TestBuildInitialGateProgressRejectsInvalidMetadataJSON(t *testing.T) {
	t.Parallel()

	if _, err := BuildInitialGateProgress(GateTypeVote, []byte("{")); err == nil {
		t.Fatal("expected metadata decode error")
	}
}

func TestMarshalGateMetadataJSONEncodesNormalizedWorkflowMetadata(t *testing.T) {
	t.Parallel()

	metadataJSON, err := MarshalGateMetadataJSON(GateTypeVote, map[string]any{
		"eligible_participant_ids": []any{"p2", "p1", "p2"},
		"options":                  []any{"south", "north", "south"},
		"response_authority":       "",
		"audience":                 "table",
	})
	if err != nil {
		t.Fatalf("MarshalGateMetadataJSON() error = %v", err)
	}

	metadata, err := DecodeGateMetadataMap(GateTypeVote, metadataJSON)
	if err != nil {
		t.Fatalf("DecodeGateMetadataMap() error = %v", err)
	}
	if got := metadata["eligible_participant_ids"]; len(got.([]any)) != 2 || got.([]any)[0] != "p1" || got.([]any)[1] != "p2" {
		t.Fatalf("eligible_participant_ids = %#v", got)
	}
	if got := metadata["options"]; len(got.([]any)) != 2 || got.([]any)[0] != "north" || got.([]any)[1] != "south" {
		t.Fatalf("options = %#v", got)
	}
	if got := metadata["response_authority"]; got != GateResponseAuthorityParticipant {
		t.Fatalf("response_authority = %#v, want %q", got, GateResponseAuthorityParticipant)
	}
	if got := metadata["audience"]; got != "table" {
		t.Fatalf("audience = %#v, want table", got)
	}
}

func TestDecodeGateProgressMapRecomputesDerivedWorkflowFields(t *testing.T) {
	t.Parallel()

	metadataJSON, err := MarshalGateMetadataJSON(GateTypeReadyCheck, map[string]any{
		"eligible_participant_ids": []string{"p1", "p2"},
	})
	if err != nil {
		t.Fatalf("MarshalGateMetadataJSON() error = %v", err)
	}

	progressJSON, err := BuildInitialGateProgress(GateTypeReadyCheck, metadataJSON)
	if err != nil {
		t.Fatalf("BuildInitialGateProgress() error = %v", err)
	}
	progressJSON, err = RecordGateResponseProgress(
		GateTypeReadyCheck,
		metadataJSON,
		progressJSON,
		GateResponseRecordedPayload{
			GateID:        ids.GateID("gate-1"),
			ParticipantID: ids.ParticipantID("p1"),
			Decision:      "wait",
		},
		time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC),
		"participant",
		"p1",
	)
	if err != nil {
		t.Fatalf("RecordGateResponseProgress() error = %v", err)
	}

	progressMap, err := DecodeGateProgressMap(GateTypeReadyCheck, metadataJSON, progressJSON)
	if err != nil {
		t.Fatalf("DecodeGateProgressMap() error = %v", err)
	}
	if got := progressMap["workflow_type"]; got != GateTypeReadyCheck {
		t.Fatalf("workflow_type = %#v, want %q", got, GateTypeReadyCheck)
	}
	if got := progressMap["response_authority"]; got != GateResponseAuthorityParticipant {
		t.Fatalf("response_authority = %#v, want %q", got, GateResponseAuthorityParticipant)
	}
	if got := progressMap["responded_count"]; got != float64(1) {
		t.Fatalf("responded_count = %#v, want 1", got)
	}
	if got := progressMap["pending_count"]; got != float64(1) {
		t.Fatalf("pending_count = %#v, want 1", got)
	}
	if got := progressMap["wait_count"]; got != float64(1) {
		t.Fatalf("wait_count = %#v, want 1", got)
	}
	if got := progressMap["resolution_state"]; got != GateResolutionStateBlocked {
		t.Fatalf("resolution_state = %#v, want %q", got, GateResolutionStateBlocked)
	}
	if got := progressMap["suggested_decision"]; got != "wait" {
		t.Fatalf("suggested_decision = %#v, want wait", got)
	}
}

func TestMarshalAndDecodeGateResolutionJSON(t *testing.T) {
	t.Parallel()

	resolutionJSON, err := MarshalGateResolutionJSON("ready", map[string]any{
		"note":  "table agreed",
		"count": 2,
	})
	if err != nil {
		t.Fatalf("MarshalGateResolutionJSON() error = %v", err)
	}

	resolution, err := DecodeGateResolutionMap(resolutionJSON)
	if err != nil {
		t.Fatalf("DecodeGateResolutionMap() error = %v", err)
	}
	if got := resolution["decision"]; got != "ready" {
		t.Fatalf("decision = %#v, want ready", got)
	}
	if got := resolution["note"]; got != "table agreed" {
		t.Fatalf("note = %#v, want table agreed", got)
	}
	if got := resolution["count"]; got != float64(2) {
		t.Fatalf("count = %#v, want 2", got)
	}

	emptyResolutionJSON, err := MarshalGateResolutionJSON(" ", nil)
	if err != nil {
		t.Fatalf("MarshalGateResolutionJSON(empty) error = %v", err)
	}
	if len(emptyResolutionJSON) != 0 {
		t.Fatalf("empty resolution JSON = %s, want nil/empty", string(emptyResolutionJSON))
	}
}

func TestRecordGateResponseProgressVoteTracksLeadingOptionsAndTieState(t *testing.T) {
	t.Parallel()

	metadataJSON, err := json.Marshal(map[string]any{
		"eligible_participant_ids": []string{"p1", "p2", "p3"},
		"options":                  []string{"north", "south", "wait"},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	progressJSON, err := BuildInitialGateProgress(GateTypeVote, metadataJSON)
	if err != nil {
		t.Fatalf("BuildInitialGateProgress() error = %v", err)
	}

	record := func(participantID, decision string, minute int) {
		t.Helper()
		progressJSON, err = RecordGateResponseProgress(
			GateTypeVote,
			metadataJSON,
			progressJSON,
			GateResponseRecordedPayload{
				GateID:        ids.GateID("gate-1"),
				ParticipantID: ids.ParticipantID(participantID),
				Decision:      decision,
			},
			time.Date(2026, 3, 9, 12, minute, 0, 0, time.UTC),
			"participant",
			participantID,
		)
		if err != nil {
			t.Fatalf("RecordGateResponseProgress(%s) error = %v", participantID, err)
		}
	}

	record("p1", "north", 0)
	record("p2", "south", 1)

	progress := GateProgress{}
	if err := json.Unmarshal(progressJSON, &progress); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if progress.AllResponded || progress.PendingCount != 1 || len(progress.PendingParticipantIDs) != 1 || progress.PendingParticipantIDs[0] != "p3" {
		t.Fatalf("progress pending summary = %#v", progress)
	}
	if progress.LeadingOptionCount != 1 {
		t.Fatalf("leading option count during tie = %d, want 1", progress.LeadingOptionCount)
	}
	if len(progress.LeadingOptions) != 2 || !gateWorkflowContains(progress.LeadingOptions, "north") || !gateWorkflowContains(progress.LeadingOptions, "south") {
		t.Fatalf("leading options during tie = %#v", progress.LeadingOptions)
	}
	if !progress.LeadingTie {
		t.Fatalf("expected leading tie: %#v", progress)
	}
	if progress.ResolutionState != GateResolutionStatePendingResponses || progress.ResolutionReason != "waiting_on_participants" || progress.SuggestedDecision != "" {
		t.Fatalf("resolution summary during tie = %#v", progress)
	}

	record("p3", "north", 2)
	progress = GateProgress{}
	if err := json.Unmarshal(progressJSON, &progress); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !progress.AllResponded || progress.PendingCount != 0 || len(progress.PendingParticipantIDs) != 0 {
		t.Fatalf("progress pending summary after leader = %#v", progress)
	}
	if progress.LeadingOptionCount != 2 {
		t.Fatalf("leading option count = %d, want 2", progress.LeadingOptionCount)
	}
	if len(progress.LeadingOptions) != 1 || progress.LeadingOptions[0] != "north" {
		t.Fatalf("leading options = %#v", progress.LeadingOptions)
	}
	if progress.LeadingTie {
		t.Fatalf("expected no leading tie: %#v", progress)
	}
	if progress.ResolutionState != GateResolutionStateReadyToResolve || progress.ResolutionReason != "leader_selected" || progress.SuggestedDecision != "north" {
		t.Fatalf("resolution summary with leader = %#v", progress)
	}

	record("p3", "south", 3)
	progress = GateProgress{}
	if err := json.Unmarshal(progressJSON, &progress); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if progress.LeadingOptionCount != 2 {
		t.Fatalf("leading option count after update = %d, want 2", progress.LeadingOptionCount)
	}
	if len(progress.LeadingOptions) != 1 || progress.LeadingOptions[0] != "south" {
		t.Fatalf("leading options after update = %#v", progress.LeadingOptions)
	}
	if progress.LeadingTie {
		t.Fatalf("expected no leading tie after update: %#v", progress)
	}
	if progress.ResolutionState != GateResolutionStateReadyToResolve || progress.ResolutionReason != "leader_selected" || progress.SuggestedDecision != "south" {
		t.Fatalf("resolution summary after update = %#v", progress)
	}
}

func TestRecordGateResponseProgressReadyCheckAllReadyMarksReadyToResolve(t *testing.T) {
	t.Parallel()

	metadataJSON, err := json.Marshal(map[string]any{
		"eligible_participant_ids": []string{"p1", "p2"},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	progressJSON, err := BuildInitialGateProgress(GateTypeReadyCheck, metadataJSON)
	if err != nil {
		t.Fatalf("BuildInitialGateProgress() error = %v", err)
	}

	progressJSON, err = RecordGateResponseProgress(
		GateTypeReadyCheck,
		metadataJSON,
		progressJSON,
		GateResponseRecordedPayload{GateID: ids.GateID("gate-1"), ParticipantID: ids.ParticipantID("p1"), Decision: "ready"},
		time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC),
		"participant",
		"p1",
	)
	if err != nil {
		t.Fatalf("RecordGateResponseProgress(first) error = %v", err)
	}
	progressJSON, err = RecordGateResponseProgress(
		GateTypeReadyCheck,
		metadataJSON,
		progressJSON,
		GateResponseRecordedPayload{GateID: ids.GateID("gate-1"), ParticipantID: ids.ParticipantID("p2"), Decision: "ready"},
		time.Date(2026, 3, 9, 12, 1, 0, 0, time.UTC),
		"participant",
		"p2",
	)
	if err != nil {
		t.Fatalf("RecordGateResponseProgress(second) error = %v", err)
	}

	var progress GateProgress
	if err := json.Unmarshal(progressJSON, &progress); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !progress.AllReady || !progress.AllResponded || progress.ReadyCount != 2 || progress.WaitCount != 0 {
		t.Fatalf("ready summary = %#v", progress)
	}
	if progress.ResolutionState != GateResolutionStateReadyToResolve || progress.ResolutionReason != "all_ready" || progress.SuggestedDecision != "ready" {
		t.Fatalf("resolution summary = %#v", progress)
	}
}

func TestBuildInitialGateProgressVoteWithoutEligibleParticipantsRequiresManualReviewAfterFirstVote(t *testing.T) {
	t.Parallel()

	progressJSON, err := BuildInitialGateProgress(GateTypeVote, nil)
	if err != nil {
		t.Fatalf("BuildInitialGateProgress() error = %v", err)
	}
	progressJSON, err = RecordGateResponseProgress(
		GateTypeVote,
		nil,
		progressJSON,
		GateResponseRecordedPayload{GateID: ids.GateID("gate-1"), ParticipantID: ids.ParticipantID("p1"), Decision: "north"},
		time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC),
		"participant",
		"p1",
	)
	if err != nil {
		t.Fatalf("RecordGateResponseProgress() error = %v", err)
	}

	var progress GateProgress
	if err := json.Unmarshal(progressJSON, &progress); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if progress.ResolutionState != GateResolutionStateManualReview || progress.ResolutionReason != "open_ended_vote" || progress.SuggestedDecision != "north" {
		t.Fatalf("resolution summary = %#v", progress)
	}
}

func TestBuildInitialGateProgressVoteWithoutEligibleParticipantsMarksNoVotesAndTieStates(t *testing.T) {
	t.Parallel()

	progressJSON, err := BuildInitialGateProgress(GateTypeVote, nil)
	if err != nil {
		t.Fatalf("BuildInitialGateProgress() error = %v", err)
	}

	progressJSON, err = RecordGateResponseProgress(
		GateTypeVote,
		nil,
		progressJSON,
		GateResponseRecordedPayload{GateID: ids.GateID("gate-1"), ParticipantID: ids.ParticipantID("p1")},
		time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC),
		"participant",
		"p1",
	)
	if err != nil {
		t.Fatalf("RecordGateResponseProgress(no-vote) error = %v", err)
	}

	var progress GateProgress
	if err := json.Unmarshal(progressJSON, &progress); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if progress.ResolutionState != GateResolutionStateManualReview || progress.ResolutionReason != "no_votes_recorded" {
		t.Fatalf("resolution summary without decisions = %#v", progress)
	}

	progressJSON, err = BuildInitialGateProgress(GateTypeVote, nil)
	if err != nil {
		t.Fatalf("BuildInitialGateProgress() error = %v", err)
	}
	for i, decision := range []string{"north", "south"} {
		progressJSON, err = RecordGateResponseProgress(
			GateTypeVote,
			nil,
			progressJSON,
			GateResponseRecordedPayload{
				GateID:        ids.GateID("gate-1"),
				ParticipantID: ids.ParticipantID("p" + string(rune('1'+i))),
				Decision:      decision,
			},
			time.Date(2026, 3, 9, 12, i, 0, 0, time.UTC),
			"participant",
			"p1",
		)
		if err != nil {
			t.Fatalf("RecordGateResponseProgress(tie %d) error = %v", i, err)
		}
	}

	progress = GateProgress{}
	if err := json.Unmarshal(progressJSON, &progress); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if progress.ResolutionState != GateResolutionStateManualReview || progress.ResolutionReason != "vote_tied" || !progress.LeadingTie {
		t.Fatalf("resolution summary for open-ended tie = %#v", progress)
	}
}

func TestRecordGateResponseProgressVoteTieAfterAllResponsesRequiresManualReview(t *testing.T) {
	t.Parallel()

	metadataJSON, err := json.Marshal(map[string]any{
		"eligible_participant_ids": []string{"p1", "p2"},
		"options":                  []string{"north", "south"},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	progressJSON, err := BuildInitialGateProgress(GateTypeVote, metadataJSON)
	if err != nil {
		t.Fatalf("BuildInitialGateProgress() error = %v", err)
	}
	progressJSON, err = RecordGateResponseProgress(
		GateTypeVote,
		metadataJSON,
		progressJSON,
		GateResponseRecordedPayload{GateID: ids.GateID("gate-1"), ParticipantID: ids.ParticipantID("p1"), Decision: "north"},
		time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC),
		"participant",
		"p1",
	)
	if err != nil {
		t.Fatalf("RecordGateResponseProgress(first) error = %v", err)
	}
	progressJSON, err = RecordGateResponseProgress(
		GateTypeVote,
		metadataJSON,
		progressJSON,
		GateResponseRecordedPayload{GateID: ids.GateID("gate-1"), ParticipantID: ids.ParticipantID("p2"), Decision: "south"},
		time.Date(2026, 3, 9, 12, 1, 0, 0, time.UTC),
		"participant",
		"p2",
	)
	if err != nil {
		t.Fatalf("RecordGateResponseProgress(second) error = %v", err)
	}

	var progress GateProgress
	if err := json.Unmarshal(progressJSON, &progress); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !progress.AllResponded || !progress.LeadingTie {
		t.Fatalf("vote summary = %#v", progress)
	}
	if progress.ResolutionState != GateResolutionStateManualReview || progress.ResolutionReason != "vote_tied" || progress.SuggestedDecision != "" {
		t.Fatalf("resolution summary = %#v", progress)
	}
}

func TestValidateGateResponseCoversReadyCheckAndGenericErrorPaths(t *testing.T) {
	t.Parallel()

	if _, _, err := ValidateGateResponse(GateTypeReadyCheck, nil, "", "ready", nil); err == nil {
		t.Fatal("expected missing participant error")
	}
	if _, _, err := ValidateGateResponse(GateTypeReadyCheck, []byte("{"), "p1", "ready", nil); err == nil {
		t.Fatal("expected invalid metadata error")
	}
	if _, _, err := ValidateGateResponse(GateTypeReadyCheck, nil, "p1", "later", nil); err == nil {
		t.Fatal("expected invalid ready_check decision error")
	}
	decision, response, err := ValidateGateResponse(GateTypeReadyCheck, nil, "p1", "READY", nil)
	if err != nil {
		t.Fatalf("ValidateGateResponse(ready) error = %v", err)
	}
	if decision != "ready" || response != nil {
		t.Fatalf("ready decision = %q response = %#v", decision, response)
	}

	if _, _, err := ValidateGateResponse(GateTypeVote, nil, "p1", "", nil); err == nil {
		t.Fatal("expected missing vote decision error")
	}

	metadataJSON, err := json.Marshal(map[string]any{"response_authority": "persona"})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if _, _, err := ValidateGateResponse("choice", metadataJSON, "p1", "advance", nil); err == nil {
		t.Fatal("expected unsupported response authority error")
	}
	if _, _, err := ValidateGateResponse("choice", nil, "p1", "", nil); err == nil {
		t.Fatal("expected generic missing decision/payload error")
	}
	decision, response, err = ValidateGateResponse("choice", nil, "p1", "", map[string]any{"note": "hold"})
	if err != nil {
		t.Fatalf("ValidateGateResponse(choice payload) error = %v", err)
	}
	if decision != "" || response["note"] != "hold" {
		t.Fatalf("generic response = decision %q payload %#v", decision, response)
	}
}

func TestRecordGateResponseProgressRejectsInvalidStoredProgress(t *testing.T) {
	t.Parallel()

	if _, err := RecordGateResponseProgress(
		GateTypeVote,
		nil,
		[]byte("{"),
		GateResponseRecordedPayload{GateID: ids.GateID("gate-1"), ParticipantID: ids.ParticipantID("p1"), Decision: "north"},
		time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC),
		"participant",
		"p1",
	); err == nil {
		t.Fatal("expected invalid progress decode error")
	}
}
