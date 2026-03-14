package session

import (
	"testing"
	"time"
)

func TestStoredGateMetadataAndProgressHelpers(t *testing.T) {
	readyMetadata, err := BuildStoredGateMetadata(GateTypeReadyCheck, map[string]any{
		"eligible_participant_ids": []string{"p2", "p1"},
		"note":                     "alpha",
	})
	if err != nil {
		t.Fatalf("build ready metadata: %v", err)
	}
	if readyMetadata.ResponseAuthority != GateResponseAuthorityParticipant {
		t.Fatalf("response authority = %q", readyMetadata.ResponseAuthority)
	}
	if len(readyMetadata.Options) != 2 || readyMetadata.Options[0] != "ready" || readyMetadata.Options[1] != "wait" {
		t.Fatalf("ready options = %#v", readyMetadata.Options)
	}

	voteMap, err := BuildGateMetadataMapFromStored(GateTypeVote, StoredGateMetadata{
		ResponseAuthority:      GateResponseAuthorityParticipant,
		EligibleParticipantIDs: []string{"p1", "p2"},
		Options:                []string{"north", "south"},
		Extra:                  map[string]any{"topic": "path"},
	})
	if err != nil {
		t.Fatalf("build vote metadata map: %v", err)
	}
	if got := voteMap["topic"]; got != "path" {
		t.Fatalf("topic = %#v", got)
	}

	progress, err := BuildGateProgressFromResponses(GateTypeVote, voteMap, []GateProgressResponse{
		{ParticipantID: "p1", Decision: "north"},
		{ParticipantID: "p2", Decision: "north"},
	})
	if err != nil {
		t.Fatalf("build gate progress: %v", err)
	}
	if progress.ResolutionState != GateResolutionStateReadyToResolve {
		t.Fatalf("resolution state = %q", progress.ResolutionState)
	}
	if progress.SuggestedDecision != "north" {
		t.Fatalf("suggested decision = %q", progress.SuggestedDecision)
	}
}

func TestStoredGateResolutionHelpers(t *testing.T) {
	stored, err := BuildStoredGateResolution("approve", map[string]any{"note": "ok"})
	if err != nil {
		t.Fatalf("build stored resolution: %v", err)
	}
	if stored.Decision != "approve" {
		t.Fatalf("decision = %q", stored.Decision)
	}
	if got := stored.Extra["note"]; got != "ok" {
		t.Fatalf("note = %#v", got)
	}

	resolution, err := BuildGateResolutionMapFromStored(stored.Decision, stored.Extra)
	if err != nil {
		t.Fatalf("build resolution map: %v", err)
	}
	if got := resolution["decision"]; got != "approve" {
		t.Fatalf("decision in map = %#v", got)
	}
}

func TestValidateGateResponseMetadata(t *testing.T) {
	decision, response, err := ValidateGateResponseMetadata(GateTypeReadyCheck, map[string]any{
		"eligible_participant_ids": []string{"p1"},
	}, "p1", "READY", map[string]any{"note": "go"})
	if err != nil {
		t.Fatalf("validate response metadata: %v", err)
	}
	if decision != "ready" {
		t.Fatalf("decision = %q", decision)
	}
	if got := response["note"]; got != "go" {
		t.Fatalf("note = %#v", got)
	}
}

func TestGateProjectionJSONHelpers(t *testing.T) {
	metadata := map[string]any{
		"eligible_participant_ids": []string{"p1", "p2"},
	}
	initial, err := BuildInitialGateProgressState(GateTypeReadyCheck, metadata)
	if err != nil {
		t.Fatalf("build initial progress: %v", err)
	}
	if initial == nil || initial.EligibleCount != 2 {
		t.Fatalf("initial progress = %#v", initial)
	}

	progressJSON, err := MarshalGateProgressJSON(initial)
	if err != nil {
		t.Fatalf("marshal gate progress: %v", err)
	}
	progressMap, err := DecodeGateProgressMap(GateTypeReadyCheck, mustMarshalGateMetadata(t, metadata), progressJSON)
	if err != nil {
		t.Fatalf("decode gate progress map: %v", err)
	}
	if got := progressMap["eligible_count"]; got != float64(2) {
		t.Fatalf("eligible_count = %#v", got)
	}

	updated, err := RecordGateResponseProgressState(
		GateTypeReadyCheck,
		metadata,
		initial,
		GateResponseRecordedPayload{ParticipantID: "p1", Decision: "ready"},
		testGateTime,
		"participant",
		"p1",
	)
	if err != nil {
		t.Fatalf("record response progress: %v", err)
	}
	if updated.RespondedCount != 1 {
		t.Fatalf("responded count = %d", updated.RespondedCount)
	}

	resolutionJSON, err := MarshalGateResolutionMapJSON(map[string]any{"decision": "ready"})
	if err != nil {
		t.Fatalf("marshal resolution map: %v", err)
	}
	resolutionMap, err := DecodeGateResolutionMap(resolutionJSON)
	if err != nil {
		t.Fatalf("decode resolution map: %v", err)
	}
	if got := resolutionMap["decision"]; got != "ready" {
		t.Fatalf("resolution decision = %#v", got)
	}

	if _, err := JSONMapFromValue(map[string]any{"ok": true}); err != nil {
		t.Fatalf("json map from value: %v", err)
	}
}

func TestGateProjectionJSONErrorAndNilHelpers(t *testing.T) {
	if got, err := DecodeGateMetadataMap("gm_handoff", nil); err != nil || got != nil {
		t.Fatalf("decode nil metadata = %#v err=%v", got, err)
	}
	if _, err := DecodeGateMetadataMap(GateTypeReadyCheck, []byte("{")); err == nil {
		t.Fatal("expected invalid metadata json error")
	}
	if _, _, err := ValidateGateResponseMetadata(GateTypeReadyCheck, map[string]any{
		"eligible_participant_ids": []string{"p1"},
	}, "p2", "ready", nil); err == nil {
		t.Fatal("expected ineligible participant error")
	}
	if _, err := BuildInitialGateProgressState(GateTypeReadyCheck, map[string]any{
		"options": []string{"nope"},
	}); err == nil {
		t.Fatal("expected invalid ready check metadata error")
	}
	if got, err := DecodeGateProgress("gm_handoff", nil, nil); err != nil || got != nil {
		t.Fatalf("decode empty progress = %#v err=%v", got, err)
	}
	if got, err := DecodeGateProgressMap("gm_handoff", nil, nil); err != nil || got != nil {
		t.Fatalf("decode empty progress map = %#v err=%v", got, err)
	}
	if _, err := DecodeGateProgress(GateTypeVote, mustMarshalGateMetadataForType(t, GateTypeVote, map[string]any{
		"options": []string{"north", "south"},
	}), []byte("{")); err == nil {
		t.Fatal("expected invalid stored progress error")
	}
	if data, err := MarshalGateProgressJSON(nil); err != nil || data != nil {
		t.Fatalf("marshal nil progress = %q err=%v", data, err)
	}
	if _, err := MarshalGateProgressJSON(&GateProgress{
		Responses: []GateProgressResponse{{
			ParticipantID: "p1",
			Response:      map[string]any{"bad": func() {}},
		}},
	}); err == nil {
		t.Fatal("expected progress marshal error")
	}
	if _, err := RecordGateResponseProgressState(GateTypeReadyCheck, map[string]any{
		"options": []string{"bad"},
	}, nil, GateResponseRecordedPayload{ParticipantID: "p1", Decision: "ready"}, testGateTime, "participant", "p1"); err == nil {
		t.Fatal("expected invalid metadata error")
	}
	if _, err := RecordGateResponseProgressState(GateTypeReadyCheck, map[string]any{
		"eligible_participant_ids": []string{"p1"},
	}, &GateProgress{
		Responses: []GateProgressResponse{{
			ParticipantID: "p1",
			Response:      map[string]any{"bad": func() {}},
		}},
	}, GateResponseRecordedPayload{ParticipantID: "p1", Decision: "ready"}, testGateTime, "participant", "p1"); err == nil {
		t.Fatal("expected progress encode error")
	}
	if got, err := BuildGateResolutionMap("", nil); err != nil || got != nil {
		t.Fatalf("empty resolution map = %#v err=%v", got, err)
	}
	if _, err := DecodeGateResolutionMap([]byte("{")); err == nil {
		t.Fatal("expected invalid resolution json error")
	}
	if _, err := JSONMapFromValue(map[string]any{"bad": func() {}}); err == nil {
		t.Fatal("expected json map encode error")
	}
}

func TestStoredGateProjectionGenericWorkflowHelpers(t *testing.T) {
	stored, err := BuildStoredGateMetadata("gm_handoff", map[string]any{
		"eligible_participant_ids": []string{"p2", "p1"},
		"response_authority":       GateResponseAuthorityParticipant,
		"note":                     "handoff",
	})
	if err != nil {
		t.Fatalf("build generic stored metadata: %v", err)
	}
	if len(stored.Options) != 0 {
		t.Fatalf("generic options = %#v", stored.Options)
	}
	if got := stored.Extra["note"]; got != "handoff" {
		t.Fatalf("generic extra note = %#v", got)
	}

	metadataMap, err := BuildGateMetadataMapFromStored("gm_handoff", stored)
	if err != nil {
		t.Fatalf("build generic metadata map: %v", err)
	}
	if got := metadataMap["note"]; got != "handoff" {
		t.Fatalf("generic metadata note = %#v", got)
	}
	if got, err := BuildGateMetadataMapFromStored("gm_handoff", StoredGateMetadata{}); err != nil || got != nil {
		t.Fatalf("empty generic metadata map = %#v err=%v", got, err)
	}

	readyMetadataMap, err := BuildGateMetadataMapFromStored(GateTypeReadyCheck, StoredGateMetadata{
		Extra: map[string]any{"topic": "check-in"},
	})
	if err != nil {
		t.Fatalf("build ready metadata map: %v", err)
	}
	options, ok := readyMetadataMap["options"].([]any)
	if !ok || len(options) != 2 {
		t.Fatalf("ready metadata options = %#v", readyMetadataMap["options"])
	}

	progress, err := BuildGateProgressFromResponses("gm_handoff", nil, nil)
	if err != nil {
		t.Fatalf("build generic progress: %v", err)
	}
	if progress != nil {
		t.Fatalf("expected nil generic progress, got %#v", progress)
	}

	emptyResolution, err := BuildStoredGateResolution("", nil)
	if err != nil {
		t.Fatalf("build empty stored resolution: %v", err)
	}
	if emptyResolution.Decision != "" || emptyResolution.Extra != nil {
		t.Fatalf("empty stored resolution = %#v", emptyResolution)
	}

	extraResolution, err := BuildStoredGateResolution("", map[string]any{"note": "gm decides"})
	if err != nil {
		t.Fatalf("build extra stored resolution: %v", err)
	}
	if extraResolution.Decision != "" || extraResolution.Extra["note"] != "gm decides" {
		t.Fatalf("extra stored resolution = %#v", extraResolution)
	}
}

var testGateTime = mustParseGateTime("2026-03-09T12:00:00Z")

func mustParseGateTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return parsed
}

func mustMarshalGateMetadata(t *testing.T, metadata map[string]any) []byte {
	return mustMarshalGateMetadataForType(t, GateTypeReadyCheck, metadata)
}

func mustMarshalGateMetadataForType(t *testing.T, gateType string, metadata map[string]any) []byte {
	t.Helper()
	data, err := MarshalGateMetadataJSON(gateType, metadata)
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	return data
}
