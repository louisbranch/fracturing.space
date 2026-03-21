package session

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func TestDecideSessionAITurnCommandsEmitExpectedEvents(t *testing.T) {
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

func TestDecideSessionAITurnCommandsRejectInvalidState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		state   State
		cmdType command.Type
		payload any
		want    string
	}{
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

func TestDecideSessionAITurnCommandsRejectInvalidPayloadFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		state   State
		cmdType command.Type
		payload any
		want    string
	}{
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

func TestDecideSessionAITurnCommandsRejectMalformedPayloadJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		cmdType command.Type
		state   State
	}{
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
