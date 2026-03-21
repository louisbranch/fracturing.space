package session

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func TestDecideSessionOOCCommandsEmitExpectedEvents(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 13, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		state     State
		cmdType   command.Type
		payload   any
		wantEvent event.Type
	}{
		{
			name:      "ooc pause",
			state:     State{Started: true},
			cmdType:   CommandTypeOOCOpen,
			payload:   OOCOpenedPayload{Reason: "rules"},
			wantEvent: EventTypeOOCOpened,
		},
		{
			name:      "ooc post",
			state:     State{Started: true, OOCPaused: true},
			cmdType:   CommandTypeOOCPost,
			payload:   OOCPostedPayload{PostID: "ooc-1", ParticipantID: "p1", Body: "Question"},
			wantEvent: EventTypeOOCPosted,
		},
		{
			name:      "ooc ready mark",
			state:     State{Started: true, OOCPaused: true},
			cmdType:   CommandTypeOOCReadyMark,
			payload:   OOCReadyMarkedPayload{ParticipantID: "p1"},
			wantEvent: EventTypeOOCReadyMarked,
		},
		{
			name:      "ooc ready clear",
			state:     State{Started: true, OOCPaused: true},
			cmdType:   CommandTypeOOCReadyClear,
			payload:   OOCReadyClearedPayload{ParticipantID: "p1"},
			wantEvent: EventTypeOOCReadyCleared,
		},
		{
			name:      "ooc resume",
			state:     State{Started: true, OOCPaused: true},
			cmdType:   CommandTypeOOCClose,
			payload:   OOCClosedPayload{Reason: "resume"},
			wantEvent: EventTypeOOCClosed,
		},
		{
			name:      "ooc interruption resolve",
			state:     State{Started: true, OOCInterruptedSceneID: "scene-1", OOCInterruptedPhaseID: "phase-1", OOCResolutionPending: true},
			cmdType:   CommandTypeOOCResolve,
			payload:   OOCResolvedPayload{Resolution: "  resume_original_phase  "},
			wantEvent: EventTypeOOCResolved,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			payloadJSON, err := json.Marshal(tc.payload)
			if err != nil {
				t.Fatalf("marshal payload: %v", err)
			}
			decision := Decide(tc.state, command.Command{
				CampaignID:  "camp-1",
				SessionID:   "sess-1",
				Type:        tc.cmdType,
				ActorType:   command.ActorTypeSystem,
				PayloadJSON: payloadJSON,
			}, func() time.Time { return now })

			if len(decision.Rejections) != 0 {
				t.Fatalf("unexpected rejections: %#v", decision.Rejections)
			}
			if len(decision.Events) != 1 {
				t.Fatalf("events = %#v, want 1 event", decision.Events)
			}
			if decision.Events[0].Type != tc.wantEvent {
				t.Fatalf("event type = %q, want %q", decision.Events[0].Type, tc.wantEvent)
			}
			if decision.Events[0].SessionID != ids.SessionID("sess-1") {
				t.Fatalf("session id = %q", decision.Events[0].SessionID)
			}
			if tc.cmdType == CommandTypeOOCResolve {
				var payload OOCResolvedPayload
				if err := json.Unmarshal(decision.Events[0].PayloadJSON, &payload); err != nil {
					t.Fatalf("decode interruption payload: %v", err)
				}
				if payload.SessionID != ids.SessionID("sess-1") {
					t.Fatalf("payload session id = %q, want %q", payload.SessionID, ids.SessionID("sess-1"))
				}
				if payload.Resolution != "resume_original_phase" {
					t.Fatalf("payload resolution = %q, want %q", payload.Resolution, "resume_original_phase")
				}
			}
		})
	}
}

func TestDecideSessionOOCCommandsRejectInvalidState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		state   State
		cmdType command.Type
		payload any
		want    string
	}{
		{
			name:    "ooc post when not paused",
			state:   State{},
			cmdType: CommandTypeOOCPost,
			payload: OOCPostedPayload{PostID: "ooc-1", ParticipantID: "p1", Body: "Question"},
			want:    rejectionCodeSessionOOCNotOpen,
		},
		{
			name:    "ooc interruption resolve when not pending",
			state:   State{Started: true},
			cmdType: CommandTypeOOCResolve,
			payload: OOCResolvedPayload{Resolution: "resume_original_phase"},
			want:    rejectionCodeSessionOOCResolutionNotPending,
		},
		{
			name:    "ooc interruption resolve while paused",
			state:   State{Started: true, OOCPaused: true, OOCInterruptedSceneID: "scene-1", OOCInterruptedPhaseID: "phase-1", OOCResolutionPending: true},
			cmdType: CommandTypeOOCResolve,
			payload: OOCResolvedPayload{Resolution: "resume_original_phase"},
			want:    rejectionCodeSessionOOCResolutionNotPending,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			payloadJSON, err := json.Marshal(tc.payload)
			if err != nil {
				t.Fatalf("marshal payload: %v", err)
			}
			decision := Decide(tc.state, command.Command{
				CampaignID:  "camp-1",
				SessionID:   "sess-1",
				Type:        tc.cmdType,
				ActorType:   command.ActorTypeSystem,
				PayloadJSON: payloadJSON,
			}, time.Now)

			if len(decision.Events) != 0 {
				t.Fatalf("unexpected events: %#v", decision.Events)
			}
			if len(decision.Rejections) != 1 {
				t.Fatalf("rejections = %#v, want 1 rejection", decision.Rejections)
			}
			if decision.Rejections[0].Code != tc.want {
				t.Fatalf("rejection code = %q, want %q", decision.Rejections[0].Code, tc.want)
			}
		})
	}
}

func TestDecideSessionOOCCommandsRejectInvalidPayloadFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		state   State
		cmdType command.Type
		payload any
		want    string
	}{
		{
			name:    "ooc pause rejects already open",
			state:   State{Started: true, OOCPaused: true},
			cmdType: CommandTypeOOCOpen,
			payload: OOCOpenedPayload{Reason: "rules"},
			want:    rejectionCodeSessionOOCAlreadyOpen,
		},
		{
			name:    "ooc post requires post id",
			state:   State{Started: true, OOCPaused: true},
			cmdType: CommandTypeOOCPost,
			payload: OOCPostedPayload{ParticipantID: "p1", Body: "Question"},
			want:    rejectionCodeSessionOOCPostIDRequired,
		},
		{
			name:    "ooc post requires participant",
			state:   State{Started: true, OOCPaused: true},
			cmdType: CommandTypeOOCPost,
			payload: OOCPostedPayload{PostID: "ooc-1", Body: "Question"},
			want:    rejectionCodeSessionOOCParticipantRequired,
		},
		{
			name:    "ooc post requires body",
			state:   State{Started: true, OOCPaused: true},
			cmdType: CommandTypeOOCPost,
			payload: OOCPostedPayload{PostID: "ooc-1", ParticipantID: "p1"},
			want:    rejectionCodeSessionOOCBodyRequired,
		},
		{
			name:    "ooc ready mark requires participant",
			state:   State{Started: true, OOCPaused: true},
			cmdType: CommandTypeOOCReadyMark,
			payload: OOCReadyMarkedPayload{},
			want:    rejectionCodeSessionOOCParticipantRequired,
		},
		{
			name:    "ooc ready clear requires participant",
			state:   State{Started: true, OOCPaused: true},
			cmdType: CommandTypeOOCReadyClear,
			payload: OOCReadyClearedPayload{},
			want:    rejectionCodeSessionOOCParticipantRequired,
		},
		{
			name:    "ooc resume rejects when closed",
			state:   State{Started: true},
			cmdType: CommandTypeOOCClose,
			payload: OOCClosedPayload{Reason: "resume"},
			want:    rejectionCodeSessionOOCNotOpen,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			payloadJSON, err := json.Marshal(tc.payload)
			if err != nil {
				t.Fatalf("marshal payload: %v", err)
			}
			decision := Decide(tc.state, command.Command{
				CampaignID:  "camp-1",
				SessionID:   "sess-1",
				Type:        tc.cmdType,
				ActorType:   command.ActorTypeSystem,
				PayloadJSON: payloadJSON,
			}, time.Now)

			if len(decision.Events) != 0 {
				t.Fatalf("unexpected events: %#v", decision.Events)
			}
			if len(decision.Rejections) != 1 {
				t.Fatalf("rejections = %#v, want 1 rejection", decision.Rejections)
			}
			if decision.Rejections[0].Code != tc.want {
				t.Fatalf("rejection code = %q, want %q", decision.Rejections[0].Code, tc.want)
			}
		})
	}
}

func TestDecideSessionOOCCommandsRejectMalformedPayloadJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		cmdType command.Type
		state   State
	}{
		{cmdType: CommandTypeOOCOpen, state: State{Started: true}},
		{cmdType: CommandTypeOOCPost, state: State{Started: true, OOCPaused: true}},
		{cmdType: CommandTypeOOCReadyMark, state: State{Started: true, OOCPaused: true}},
		{cmdType: CommandTypeOOCReadyClear, state: State{Started: true, OOCPaused: true}},
		{cmdType: CommandTypeOOCClose, state: State{Started: true, OOCPaused: true}},
		{cmdType: CommandTypeOOCResolve, state: State{Started: true, OOCInterruptedSceneID: "scene-1", OOCInterruptedPhaseID: "phase-1", OOCResolutionPending: true}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(string(tc.cmdType), func(t *testing.T) {
			t.Parallel()

			decision := Decide(tc.state, command.Command{
				CampaignID:  "camp-1",
				SessionID:   "sess-1",
				Type:        tc.cmdType,
				ActorType:   command.ActorTypeSystem,
				PayloadJSON: []byte(`{`),
			}, time.Now)

			if len(decision.Events) != 0 {
				t.Fatalf("unexpected events: %#v", decision.Events)
			}
			if len(decision.Rejections) != 1 {
				t.Fatalf("rejections = %#v, want 1 rejection", decision.Rejections)
			}
			if decision.Rejections[0].Code != command.RejectionCodePayloadDecodeFailed {
				t.Fatalf("rejection code = %q, want %q", decision.Rejections[0].Code, command.RejectionCodePayloadDecodeFailed)
			}
		})
	}
}
