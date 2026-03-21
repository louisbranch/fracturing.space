package session

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func TestDecideSessionAuthorityCommandsEmitExpectedEvents(t *testing.T) {
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
			cmdType:   CommandTypeSceneActivate,
			payload:   SceneActivatedPayload{ActiveSceneID: "scene-1"},
			wantEvent: EventTypeSceneActivated,
		},
		{
			name:      "gm authority set",
			state:     State{Started: true},
			cmdType:   CommandTypeGMAuthoritySet,
			payload:   GMAuthoritySetPayload{ParticipantID: "gm-1"},
			wantEvent: EventTypeGMAuthoritySet,
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

func TestDecideSessionAuthorityCommandsRejectInvalidState(t *testing.T) {
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
			cmdType: CommandTypeSceneActivate,
			payload: SceneActivatedPayload{ActiveSceneID: "scene-1"},
			want:    rejectionCodeSessionActiveSceneUnchanged,
		},
		{
			name:    "gm authority unchanged",
			state:   State{GMAuthorityParticipantID: "gm-1"},
			cmdType: CommandTypeGMAuthoritySet,
			payload: GMAuthoritySetPayload{ParticipantID: "gm-1"},
			want:    rejectionCodeSessionGMAuthorityUnchanged,
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

func TestDecideSessionAuthorityCommandsRejectInvalidPayloadFields(t *testing.T) {
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
			cmdType: CommandTypeSceneActivate,
			payload: SceneActivatedPayload{},
			want:    rejectionCodeSessionActiveSceneRequired,
		},
		{
			name:    "gm authority requires participant",
			state:   State{Started: true},
			cmdType: CommandTypeGMAuthoritySet,
			payload: GMAuthoritySetPayload{},
			want:    rejectionCodeSessionGMAuthorityRequired,
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

func TestDecideSessionAuthorityCommandsRejectMalformedPayloadJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		cmdType command.Type
		state   State
	}{
		{cmdType: CommandTypeSceneActivate, state: State{Started: true}},
		{cmdType: CommandTypeGMAuthoritySet, state: State{Started: true}},
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
