package session

import (
	"encoding/json"
	"testing"
)

func TestSessionInteractionPayloadValidatorsAcceptAndRejectJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		validate func([]byte) error
		valid    string
	}{
		{name: "active scene set", validate: func(raw []byte) error { return validateActiveSceneSetPayload(json.RawMessage(raw)) }, valid: `{"active_scene_id":"scene-1"}`},
		{name: "gm authority set", validate: func(raw []byte) error { return validateGMAuthoritySetPayload(json.RawMessage(raw)) }, valid: `{"participant_id":"gm-1"}`},
		{name: "ooc paused", validate: func(raw []byte) error { return validateOOCPausedPayload(json.RawMessage(raw)) }, valid: `{"reason":"rules"}`},
		{name: "ooc posted", validate: func(raw []byte) error { return validateOOCPostedPayload(json.RawMessage(raw)) }, valid: `{"post_id":"ooc-1","participant_id":"p1","body":"question"}`},
		{name: "ooc ready marked", validate: func(raw []byte) error { return validateOOCReadyMarkedPayload(json.RawMessage(raw)) }, valid: `{"participant_id":"p1"}`},
		{name: "ooc ready cleared", validate: func(raw []byte) error { return validateOOCReadyClearedPayload(json.RawMessage(raw)) }, valid: `{"participant_id":"p1"}`},
		{name: "ooc resumed", validate: func(raw []byte) error { return validateOOCResumedPayload(json.RawMessage(raw)) }, valid: `{"reason":"resume"}`},
		{name: "ooc interruption resolved", validate: func(raw []byte) error { return validateOOCInterruptionResolvedPayload(json.RawMessage(raw)) }, valid: `{"resolution":"resume_original_phase"}`},
		{name: "ai turn queued", validate: func(raw []byte) error { return validateAITurnQueuedPayload(json.RawMessage(raw)) }, valid: `{"turn_token":"turn-1","owner_participant_id":"gm-ai"}`},
		{name: "ai turn running", validate: func(raw []byte) error { return validateAITurnRunningPayload(json.RawMessage(raw)) }, valid: `{"turn_token":"turn-1"}`},
		{name: "ai turn failed", validate: func(raw []byte) error { return validateAITurnFailedPayload(json.RawMessage(raw)) }, valid: `{"turn_token":"turn-1","last_error":"timeout"}`},
		{name: "ai turn cleared", validate: func(raw []byte) error { return validateAITurnClearedPayload(json.RawMessage(raw)) }, valid: `{"turn_token":"turn-1","reason":"retry"}`},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := tc.validate([]byte(tc.valid)); err != nil {
				t.Fatalf("valid payload error = %v", err)
			}
			if err := tc.validate([]byte(`{`)); err == nil {
				t.Fatal("invalid payload unexpectedly accepted")
			}
		})
	}
}
