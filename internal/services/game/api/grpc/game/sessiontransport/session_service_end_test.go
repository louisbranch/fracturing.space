package sessiontransport

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestEndSession_NilRequest(t *testing.T) {
	svc := NewSessionService(Deps{})
	_, err := svc.EndSession(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestEndSession_MissingCampaignId(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore})
	_, err := svc.EndSession(context.Background(), &statev1.EndSessionRequest{SessionId: "s1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestEndSession_MissingSessionId(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore})
	_, err := svc.EndSession(context.Background(), &statev1.EndSessionRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestEndSession_CampaignNotFound(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore})
	_, err := svc.EndSession(context.Background(), &statev1.EndSessionRequest{CampaignId: "c1", SessionId: "s1"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestEndSession_SessionNotFound(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore, Participant: participantStore})
	_, err := svc.EndSession(gametest.ContextWithParticipantID("manager-1"), &statev1.EndSessionRequest{CampaignId: "c1", SessionId: "s1"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestEndSession_DeniesMemberAccess(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := gametest.NewFakeParticipantStore()
	now := time.Now().UTC()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now},
	}
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"member-1": {
			ID:             "member-1",
			CampaignID:     "c1",
			CampaignAccess: participant.CampaignAccessMember,
		},
	}

	svc := NewSessionService(Deps{
		Campaign:    campaignStore,
		Session:     sessionStore,
		Participant: participantStore,
	})
	_, err := svc.EndSession(gametest.ContextWithParticipantID("member-1"), &statev1.EndSessionRequest{
		CampaignId: "c1",
		SessionId:  "s1",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestEndSession_RequiresDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now},
	}
	sessionStore.ActiveSession["c1"] = "s1"

	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore, Participant: participantStore})
	_, err := svc.EndSession(gametest.ContextWithParticipantID("manager-1"), &statev1.EndSessionRequest{CampaignId: "c1", SessionId: "s1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestEndSession_Success(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now},
	}
	sessionStore.ActiveSession["c1"] = "s1"
	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("session.ended"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			SessionID:   "s1",
			EntityType:  "session",
			EntityID:    "s1",
			PayloadJSON: []byte(`{"session_id":"s1"}`),
		}),
	}}

	svc := newTestSessionService(
		Deps{
			Campaign:    campaignStore,
			Session:     sessionStore,
			Participant: participantStore,
			Write:       domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("session-123"),
	)

	resp, err := svc.EndSession(gametest.ContextWithParticipantID("manager-1"), &statev1.EndSessionRequest{CampaignId: "c1", SessionId: "s1"})
	if err != nil {
		t.Fatalf("EndSession returned error: %v", err)
	}
	if resp.Session.Status != statev1.SessionStatus_SESSION_ENDED {
		t.Errorf("Session Status = %v, want %v", resp.Session.Status, statev1.SessionStatus_SESSION_ENDED)
	}
	if resp.Session.EndedAt == nil {
		t.Error("Session EndedAt is nil")
	}
	if got := len(eventStore.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.Events["c1"][0].Type != event.Type("session.ended") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["c1"][0].Type, event.Type("session.ended"))
	}
}

func TestEndSession_UsesDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now},
	}
	sessionStore.ActiveSession["c1"] = "s1"

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("session.ended"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			SessionID:   "s1",
			EntityType:  "session",
			EntityID:    "s1",
			PayloadJSON: []byte(`{"session_id":"s1"}`),
		}),
	}}

	svc := newTestSessionService(
		Deps{
			Campaign:    campaignStore,
			Session:     sessionStore,
			Participant: participantStore,
			Write:       domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(now),
		nil,
	)

	_, err := svc.EndSession(gametest.ContextWithParticipantID("manager-1"), &statev1.EndSessionRequest{CampaignId: "c1", SessionId: "s1"})
	if err != nil {
		t.Fatalf("EndSession returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if domain.lastCommand.Type != command.Type("session.end") {
		t.Fatalf("command type = %s, want %s", domain.lastCommand.Type, "session.end")
	}
}

func TestEndSession_AlreadyEnded(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	endedAt := now.Add(-1 * time.Hour)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusEnded, StartedAt: now.Add(-2 * time.Hour), EndedAt: &endedAt},
	}

	svc := newTestSessionService(
		Deps{
			Campaign:    campaignStore,
			Session:     sessionStore,
			Participant: participantStore,
		},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("session-123"),
	)

	resp, err := svc.EndSession(gametest.ContextWithParticipantID("manager-1"), &statev1.EndSessionRequest{CampaignId: "c1", SessionId: "s1"})
	if err != nil {
		t.Fatalf("EndSession returned error: %v", err)
	}
	if resp.Session.Status != statev1.SessionStatus_SESSION_ENDED {
		t.Errorf("Session Status = %v, want %v", resp.Session.Status, statev1.SessionStatus_SESSION_ENDED)
	}
}
