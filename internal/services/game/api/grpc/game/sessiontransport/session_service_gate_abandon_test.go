package sessiontransport

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestAbandonSessionGate_NilRequest(t *testing.T) {
	svc := NewSessionService(Deps{})
	_, err := svc.AbandonSessionGate(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAbandonSessionGate_MissingCampaignId(t *testing.T) {
	svc := NewSessionService(Deps{
		Campaign:    gametest.NewFakeCampaignStore(),
		Session:     gametest.NewFakeSessionStore(),
		SessionGate: gametest.NewFakeSessionGateStore(),
	})
	_, err := svc.AbandonSessionGate(context.Background(), &statev1.AbandonSessionGateRequest{
		SessionId: "s1", GateId: "g1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAbandonSessionGate_MissingSessionId(t *testing.T) {
	svc := NewSessionService(Deps{
		Campaign:    gametest.NewFakeCampaignStore(),
		Session:     gametest.NewFakeSessionStore(),
		SessionGate: gametest.NewFakeSessionGateStore(),
	})
	_, err := svc.AbandonSessionGate(context.Background(), &statev1.AbandonSessionGateRequest{
		CampaignId: "c1", GateId: "g1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAbandonSessionGate_MissingGateId(t *testing.T) {
	svc := NewSessionService(Deps{
		Campaign:    gametest.NewFakeCampaignStore(),
		Session:     gametest.NewFakeSessionStore(),
		SessionGate: gametest.NewFakeSessionGateStore(),
	})
	_, err := svc.AbandonSessionGate(context.Background(), &statev1.AbandonSessionGateRequest{
		CampaignId: "c1", SessionId: "s1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAbandonSessionGate_CampaignNotFound(t *testing.T) {
	svc := NewSessionService(Deps{
		Campaign:    gametest.NewFakeCampaignStore(),
		Session:     gametest.NewFakeSessionStore(),
		SessionGate: gametest.NewFakeSessionGateStore(),
	})
	_, err := svc.AbandonSessionGate(context.Background(), &statev1.AbandonSessionGateRequest{
		CampaignId: "c1", SessionId: "s1", GateId: "g1",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestAbandonSessionGate_DeniesMemberAccess(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	gateStore := gametest.NewFakeSessionGateStore()
	participantStore := gametest.NewFakeParticipantStore()
	now := time.Now().UTC()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	gateStore.Gates["c1:s1:g1"] = storage.SessionGate{
		CampaignID: "c1", SessionID: "s1", GateID: "g1",
		GateType: "decision", Status: session.GateStatusOpen,
		CreatedAt: now,
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
		SessionGate: gateStore,
		Participant: participantStore,
	})
	_, err := svc.AbandonSessionGate(gametest.ContextWithParticipantID("member-1"), &statev1.AbandonSessionGateRequest{
		CampaignId: "c1", SessionId: "s1", GateId: "g1",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestAbandonSessionGate_AlreadyAbandoned(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	gateStore := gametest.NewFakeSessionGateStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	gateStore.Gates["c1:s1:g1"] = storage.SessionGate{
		CampaignID: "c1", SessionID: "s1", GateID: "g1",
		GateType: "decision", Status: session.GateStatusAbandoned,
		CreatedAt: now,
	}

	svc := newTestSessionService(
		Deps{
			Campaign:    campaignStore,
			Session:     sessionStore,
			SessionGate: gateStore,
			Participant: participantStore,
		},
		gametest.FixedClock(now),
		nil,
	)

	resp, err := svc.AbandonSessionGate(gametest.ContextWithParticipantID("manager-1"), &statev1.AbandonSessionGateRequest{
		CampaignId: "c1", SessionId: "s1", GateId: "g1",
	})
	if err != nil {
		t.Fatalf("AbandonSessionGate returned error: %v", err)
	}
	if resp.GetGate() == nil {
		t.Fatal("expected gate in response")
	}
	if resp.GetGate().GetStatus() != statev1.SessionGateStatus_SESSION_GATE_ABANDONED {
		t.Fatalf("gate status = %v, want ABANDONED", resp.GetGate().GetStatus())
	}
	if len(eventStore.Events["c1"]) != 0 {
		t.Fatalf("expected 0 events for already-abandoned gate, got %d", len(eventStore.Events["c1"]))
	}
}

func TestAbandonSessionGate_Success(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	gateStore := gametest.NewFakeSessionGateStore()
	participantStore := gametest.NewFakeParticipantStore()
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	gateStore.Gates["c1:s1:g1"] = storage.SessionGate{
		CampaignID: "c1", SessionID: "s1", GateID: "g1",
		GateType: "decision", Status: session.GateStatusOpen,
		CreatedAt: now,
	}
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"part-1": {
			ID:             "part-1",
			CampaignID:     "c1",
			CampaignAccess: participant.CampaignAccessManager,
		},
	}
	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("session.gate_abandoned"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "part-1",
			SessionID:   "s1",
			EntityType:  "session_gate",
			EntityID:    "g1",
			PayloadJSON: []byte(`{"gate_id":"g1","reason":"timeout"}`),
		}),
	}}

	svc := newTestSessionService(
		Deps{
			Campaign:    campaignStore,
			Session:     sessionStore,
			SessionGate: gateStore,
			Participant: participantStore,
			Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(now),
		nil,
	)

	ctx := gametest.ContextWithParticipantID("part-1")
	resp, err := svc.AbandonSessionGate(ctx, &statev1.AbandonSessionGateRequest{
		CampaignId: "c1", SessionId: "s1", GateId: "g1", Reason: "timeout",
	})
	if err != nil {
		t.Fatalf("AbandonSessionGate returned error: %v", err)
	}
	if resp.GetGate() == nil {
		t.Fatal("expected gate in response")
	}
	if got := len(eventStore.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.Events["c1"][0].Type != event.Type("session.gate_abandoned") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["c1"][0].Type, event.Type("session.gate_abandoned"))
	}
}

func TestAbandonSessionGate_RequiresDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	gateStore := gametest.NewFakeSessionGateStore()
	participantStore := sessionManagerParticipantStore("c1")
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	gateStore.Gates["c1:s1:g1"] = storage.SessionGate{
		CampaignID: "c1", SessionID: "s1", GateID: "g1",
		GateType: "decision", Status: session.GateStatusOpen,
		CreatedAt: now,
	}

	svc := newTestSessionService(
		Deps{
			Campaign:    campaignStore,
			Session:     sessionStore,
			SessionGate: gateStore,
			Participant: participantStore,
		},
		nil,
		nil,
	)
	_, err := svc.AbandonSessionGate(gametest.ContextWithParticipantID("manager-1"), &statev1.AbandonSessionGateRequest{
		CampaignId: "c1", SessionId: "s1", GateId: "g1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestAbandonSessionGate_UsesDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	gateStore := gametest.NewFakeSessionGateStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	gateStore.Gates["c1:s1:g1"] = storage.SessionGate{
		CampaignID: "c1", SessionID: "s1", GateID: "g1",
		GateType: "decision", Status: session.GateStatusOpen,
		CreatedAt: now,
	}

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("session.gate_abandoned"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			SessionID:   "s1",
			EntityType:  "session_gate",
			EntityID:    "g1",
			PayloadJSON: []byte(`{"gate_id":"g1","reason":"timeout"}`),
		}),
	}}

	svc := newTestSessionService(
		Deps{
			Campaign:    campaignStore,
			Session:     sessionStore,
			SessionGate: gateStore,
			Participant: participantStore,
			Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(now),
		nil,
	)

	_, err := svc.AbandonSessionGate(gametest.ContextWithParticipantID("manager-1"), &statev1.AbandonSessionGateRequest{
		CampaignId: "c1",
		SessionId:  "s1",
		GateId:     "g1",
		Reason:     "timeout",
	})
	if err != nil {
		t.Fatalf("AbandonSessionGate returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if domain.lastCommand.Type != command.Type("session.gate_abandon") {
		t.Fatalf("command type = %s, want %s", domain.lastCommand.Type, "session.gate_abandon")
	}
}
