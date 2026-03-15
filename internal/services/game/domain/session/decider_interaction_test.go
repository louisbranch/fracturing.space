package session

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func TestDecideSessionInteractionCommandsEmitExpectedEvents(t *testing.T) {
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
			name:      "active scene set",
			state:     State{Started: true},
			cmdType:   CommandTypeActiveSceneSet,
			payload:   ActiveSceneSetPayload{ActiveSceneID: "scene-1"},
			wantEvent: EventTypeActiveSceneSet,
		},
		{
			name:      "gm authority set",
			state:     State{Started: true},
			cmdType:   CommandTypeGMAuthoritySet,
			payload:   GMAuthoritySetPayload{ParticipantID: "gm-1"},
			wantEvent: EventTypeGMAuthoritySet,
		},
		{
			name:      "ooc pause",
			state:     State{Started: true},
			cmdType:   CommandTypeOOCPause,
			payload:   OOCPausedPayload{Reason: "rules"},
			wantEvent: EventTypeOOCPaused,
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
			cmdType:   CommandTypeOOCResume,
			payload:   OOCResumedPayload{Reason: "resume"},
			wantEvent: EventTypeOOCResumed,
		},
		{
			name:    "ai turn queue",
			state:   State{Started: true},
			cmdType: CommandTypeAITurnQueue,
			payload: AITurnQueuedPayload{
				TurnToken:          "turn-1",
				OwnerParticipantID: "gm-ai",
				SourceEventType:    "scene.player_phase_ended",
				SourceSceneID:      "scene-1",
				SourcePhaseID:      "phase-1",
			},
			wantEvent: EventTypeAITurnQueued,
		},
		{
			name:      "ai turn start",
			state:     State{Started: true, AITurnStatus: AITurnStatusQueued, AITurnToken: "turn-1"},
			cmdType:   CommandTypeAITurnStart,
			payload:   AITurnRunningPayload{TurnToken: "turn-1"},
			wantEvent: EventTypeAITurnRunning,
		},
		{
			name:      "ai turn fail",
			state:     State{Started: true, AITurnStatus: AITurnStatusRunning, AITurnToken: "turn-1"},
			cmdType:   CommandTypeAITurnFail,
			payload:   AITurnFailedPayload{TurnToken: "turn-1", LastError: "timeout"},
			wantEvent: EventTypeAITurnFailed,
		},
		{
			name:      "ai turn clear",
			state:     State{Started: true, AITurnStatus: AITurnStatusFailed, AITurnToken: "turn-1"},
			cmdType:   CommandTypeAITurnClear,
			payload:   AITurnClearedPayload{TurnToken: "turn-1", Reason: "retry"},
			wantEvent: EventTypeAITurnCleared,
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
		})
	}
}

func TestDecideSessionInteractionCommandsRejectInvalidState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		state   State
		cmdType command.Type
		payload any
		want    string
	}{
		{
			name:    "active scene unchanged",
			state:   State{ActiveSceneID: "scene-1"},
			cmdType: CommandTypeActiveSceneSet,
			payload: ActiveSceneSetPayload{ActiveSceneID: "scene-1"},
			want:    rejectionCodeSessionActiveSceneUnchanged,
		},
		{
			name:    "gm authority unchanged",
			state:   State{GMAuthorityParticipantID: "gm-1"},
			cmdType: CommandTypeGMAuthoritySet,
			payload: GMAuthoritySetPayload{ParticipantID: "gm-1"},
			want:    rejectionCodeSessionGMAuthorityUnchanged,
		},
		{
			name:    "ooc post when not paused",
			state:   State{},
			cmdType: CommandTypeOOCPost,
			payload: OOCPostedPayload{PostID: "ooc-1", ParticipantID: "p1", Body: "Question"},
			want:    rejectionCodeSessionOOCNotOpen,
		},
		{
			name:    "ai turn start token mismatch",
			state:   State{AITurnStatus: AITurnStatusQueued, AITurnToken: "turn-1"},
			cmdType: CommandTypeAITurnStart,
			payload: AITurnRunningPayload{TurnToken: "turn-2"},
			want:    rejectionCodeSessionAITurnTokenMismatch,
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

func TestDecideSessionInteractionCommandsRejectInvalidPayloadFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		state   State
		cmdType command.Type
		payload any
		want    string
	}{
		{
			name:    "active scene requires id",
			state:   State{Started: true},
			cmdType: CommandTypeActiveSceneSet,
			payload: ActiveSceneSetPayload{},
			want:    rejectionCodeSessionActiveSceneRequired,
		},
		{
			name:    "gm authority requires participant",
			state:   State{Started: true},
			cmdType: CommandTypeGMAuthoritySet,
			payload: GMAuthoritySetPayload{},
			want:    rejectionCodeSessionGMAuthorityRequired,
		},
		{
			name:    "ooc pause rejects already open",
			state:   State{Started: true, OOCPaused: true},
			cmdType: CommandTypeOOCPause,
			payload: OOCPausedPayload{Reason: "rules"},
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
			cmdType: CommandTypeOOCResume,
			payload: OOCResumedPayload{Reason: "resume"},
			want:    rejectionCodeSessionOOCNotOpen,
		},
		{
			name:    "ai turn queue requires token",
			state:   State{Started: true},
			cmdType: CommandTypeAITurnQueue,
			payload: AITurnQueuedPayload{OwnerParticipantID: "gm-ai"},
			want:    rejectionCodeSessionAITurnTokenRequired,
		},
		{
			name:    "ai turn queue requires owner",
			state:   State{Started: true},
			cmdType: CommandTypeAITurnQueue,
			payload: AITurnQueuedPayload{TurnToken: "turn-1"},
			want:    rejectionCodeSessionAITurnOwnerRequired,
		},
		{
			name:    "ai turn start requires token",
			state:   State{Started: true, AITurnStatus: AITurnStatusQueued, AITurnToken: "turn-1"},
			cmdType: CommandTypeAITurnStart,
			payload: AITurnRunningPayload{},
			want:    rejectionCodeSessionAITurnTokenRequired,
		},
		{
			name:    "ai turn start rejects when not queued",
			state:   State{Started: true, AITurnStatus: AITurnStatusFailed, AITurnToken: "turn-1"},
			cmdType: CommandTypeAITurnStart,
			payload: AITurnRunningPayload{TurnToken: "turn-1"},
			want:    rejectionCodeSessionAITurnNotQueued,
		},
		{
			name:    "ai turn fail requires token",
			state:   State{Started: true, AITurnStatus: AITurnStatusRunning, AITurnToken: "turn-1"},
			cmdType: CommandTypeAITurnFail,
			payload: AITurnFailedPayload{LastError: "timeout"},
			want:    rejectionCodeSessionAITurnTokenRequired,
		},
		{
			name:    "ai turn fail rejects when not active",
			state:   State{Started: true, AITurnStatus: AITurnStatusFailed, AITurnToken: "turn-1"},
			cmdType: CommandTypeAITurnFail,
			payload: AITurnFailedPayload{TurnToken: "turn-1", LastError: "timeout"},
			want:    rejectionCodeSessionAITurnNotActive,
		},
		{
			name:    "ai turn clear rejects token mismatch",
			state:   State{Started: true, AITurnStatus: AITurnStatusFailed, AITurnToken: "turn-1"},
			cmdType: CommandTypeAITurnClear,
			payload: AITurnClearedPayload{TurnToken: "turn-2", Reason: "retry"},
			want:    rejectionCodeSessionAITurnTokenMismatch,
		},
		{
			name:    "ai turn clear rejects when not active",
			state:   State{Started: true},
			cmdType: CommandTypeAITurnClear,
			payload: AITurnClearedPayload{Reason: "retry"},
			want:    rejectionCodeSessionAITurnNotActive,
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

func TestDecideSessionInteractionCommandsRejectMalformedPayloadJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		cmdType command.Type
		state   State
	}{
		{cmdType: CommandTypeActiveSceneSet, state: State{Started: true}},
		{cmdType: CommandTypeGMAuthoritySet, state: State{Started: true}},
		{cmdType: CommandTypeOOCPause, state: State{Started: true}},
		{cmdType: CommandTypeOOCPost, state: State{Started: true, OOCPaused: true}},
		{cmdType: CommandTypeOOCReadyMark, state: State{Started: true, OOCPaused: true}},
		{cmdType: CommandTypeOOCReadyClear, state: State{Started: true, OOCPaused: true}},
		{cmdType: CommandTypeOOCResume, state: State{Started: true, OOCPaused: true}},
		{cmdType: CommandTypeAITurnQueue, state: State{Started: true}},
		{cmdType: CommandTypeAITurnStart, state: State{Started: true, AITurnStatus: AITurnStatusQueued, AITurnToken: "turn-1"}},
		{cmdType: CommandTypeAITurnFail, state: State{Started: true, AITurnStatus: AITurnStatusRunning, AITurnToken: "turn-1"}},
		{cmdType: CommandTypeAITurnClear, state: State{Started: true, AITurnStatus: AITurnStatusFailed, AITurnToken: "turn-1"}},
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
