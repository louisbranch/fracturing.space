package integrationoutbox

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	gameintegration "github.com/louisbranch/fracturing.space/internal/services/game/integration"
)

func TestIntegrationOutboxEventsForEventBuildsAIGMTurnRequests(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 12, 20, 0, 0, 0, time.UTC)
	tests := []struct {
		name            string
		evt             event.Event
		wantSourceType  string
		wantSourceScene string
		wantSourcePhase string
	}{
		{
			name: "gm authority set",
			evt: event.Event{
				CampaignID: ids.CampaignID("camp-1"),
				SessionID:  ids.SessionID("sess-1"),
				Type:       session.EventTypeGMAuthoritySet,
				Timestamp:  now,
				PayloadJSON: mustJSON(t, session.GMAuthoritySetPayload{
					SessionID:     ids.SessionID("sess-1"),
					ParticipantID: ids.ParticipantID("gm-ai"),
				}),
			},
			wantSourceType: string(session.EventTypeGMAuthoritySet),
		},
		{
			name: "ooc resumed",
			evt: event.Event{
				CampaignID: ids.CampaignID("camp-1"),
				SessionID:  ids.SessionID("sess-1"),
				Type:       session.EventTypeOOCResumed,
				Timestamp:  now,
				PayloadJSON: mustJSON(t, session.OOCResumedPayload{
					SessionID: ids.SessionID("sess-1"),
				}),
			},
			wantSourceType: string(session.EventTypeOOCResumed),
		},
		{
			name: "player phase review started",
			evt: event.Event{
				CampaignID: ids.CampaignID("camp-1"),
				SessionID:  ids.SessionID("sess-1"),
				Type:       scene.EventTypePlayerPhaseReviewStarted,
				Timestamp:  now,
				PayloadJSON: mustJSON(t, scene.PlayerPhaseReviewStartedPayload{
					SceneID: ids.SceneID("scene-1"),
					PhaseID: "phase-1",
				}),
			},
			wantSourceType:  string(scene.EventTypePlayerPhaseReviewStarted),
			wantSourceScene: "scene-1",
			wantSourcePhase: "phase-1",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			outboxEvents, err := integrationOutboxEventsForEvent(tc.evt)
			if err != nil {
				t.Fatalf("integrationOutboxEventsForEvent error = %v", err)
			}
			if len(outboxEvents) != 1 {
				t.Fatalf("outbox events = %d, want 1", len(outboxEvents))
			}
			outboxEvent := outboxEvents[0]
			if outboxEvent.EventType != gameintegration.AIGMTurnRequestedOutboxEventType {
				t.Fatalf("event type = %q, want %q", outboxEvent.EventType, gameintegration.AIGMTurnRequestedOutboxEventType)
			}
			if outboxEvent.DedupeKey != gameintegration.AIGMTurnRequestedDedupeKey(outboxEvent.ID) {
				t.Fatalf("dedupe key = %q, want id-derived key", outboxEvent.DedupeKey)
			}
			var payload gameintegration.AIGMTurnRequestedOutboxPayload
			if err := json.Unmarshal([]byte(outboxEvent.PayloadJSON), &payload); err != nil {
				t.Fatalf("unmarshal payload: %v", err)
			}
			if payload.CampaignID != "camp-1" || payload.SessionID != "sess-1" {
				t.Fatalf("payload ids = %#v", payload)
			}
			if payload.SourceEventType != tc.wantSourceType || payload.SourceSceneID != tc.wantSourceScene || payload.SourcePhaseID != tc.wantSourcePhase {
				t.Fatalf("payload source = %#v", payload)
			}
		})
	}
}

func TestIntegrationOutboxEventsForEventSkipsAIRequestWithoutCampaignOrSession(t *testing.T) {
	t.Parallel()

	outboxEvents, err := integrationOutboxEventsForEvent(event.Event{
		Type:      session.EventTypeOOCResumed,
		Timestamp: time.Date(2026, 3, 12, 20, 0, 0, 0, time.UTC),
		PayloadJSON: mustJSON(t, session.OOCResumedPayload{
			SessionID: ids.SessionID(""),
		}),
	})
	if err != nil {
		t.Fatalf("integrationOutboxEventsForEvent error = %v", err)
	}
	if len(outboxEvents) != 0 {
		t.Fatalf("outbox events = %d, want 0", len(outboxEvents))
	}
}

func TestIntegrationOutboxEventsForEventRejectsInvalidAIGMSourcePayload(t *testing.T) {
	t.Parallel()

	_, err := integrationOutboxEventsForEvent(event.Event{
		CampaignID:  ids.CampaignID("camp-1"),
		SessionID:   ids.SessionID("sess-1"),
		Type:        scene.EventTypePlayerPhaseReviewStarted,
		Timestamp:   time.Date(2026, 3, 12, 20, 0, 0, 0, time.UTC),
		PayloadJSON: []byte(`{"scene_id":1}`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIntegrationOutboxEventsForEventBuildsInviteOutcomeNotifications(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		evt           event.Event
		wantEventType string
		wantDedupeKey string
	}{
		{
			name: "claimed",
			evt: event.Event{
				CampaignID: ids.CampaignID("camp-1"),
				Type:       invite.EventTypeClaimed,
				Timestamp:  time.Date(2026, 3, 12, 20, 0, 0, 0, time.UTC),
				EntityID:   "invite-1",
				PayloadJSON: mustJSON(t, invite.ClaimPayload{
					InviteID: ids.InviteID("invite-1"),
					UserID:   ids.UserID("user-2"),
				}),
			},
			wantEventType: gameintegration.InviteNotificationClaimedOutboxEventType,
			wantDedupeKey: gameintegration.InviteAcceptedNotificationDedupeKey("invite-1"),
		},
		{
			name: "declined",
			evt: event.Event{
				CampaignID: ids.CampaignID("camp-1"),
				Type:       invite.EventTypeDeclined,
				Timestamp:  time.Date(2026, 3, 12, 20, 0, 0, 0, time.UTC),
				EntityID:   "invite-1",
				PayloadJSON: mustJSON(t, invite.DeclinePayload{
					InviteID: ids.InviteID("invite-1"),
					UserID:   ids.UserID("user-2"),
				}),
			},
			wantEventType: gameintegration.InviteNotificationDeclinedOutboxEventType,
			wantDedupeKey: gameintegration.InviteDeclinedNotificationDedupeKey("invite-1"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			outboxEvents, err := integrationOutboxEventsForEvent(tc.evt)
			if err != nil {
				t.Fatalf("integrationOutboxEventsForEvent error = %v", err)
			}
			if len(outboxEvents) != 1 {
				t.Fatalf("outbox events = %d, want 1", len(outboxEvents))
			}
			if outboxEvents[0].EventType != tc.wantEventType || outboxEvents[0].DedupeKey != tc.wantDedupeKey {
				t.Fatalf("outbox event = %#v", outboxEvents[0])
			}
		})
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return data
}
