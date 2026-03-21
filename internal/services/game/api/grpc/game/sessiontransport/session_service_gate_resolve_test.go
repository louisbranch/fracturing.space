package sessiontransport

import (
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestResolveSessionGate_UsesDomainEngine(t *testing.T) {
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
			Type:        event.Type("session.gate_resolved"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			SessionID:   "s1",
			EntityType:  "session_gate",
			EntityID:    "g1",
			PayloadJSON: []byte(`{"gate_id":"g1","decision":"allow"}`),
		}),
	}}

	svc := newTestSessionService(
		Deps{
			Campaign:    campaignStore,
			Session:     sessionStore,
			SessionGate: gateStore,
			Participant: participantStore,
			Write:       testWritePath(domain),
		},
		gametest.FixedClock(now),
		nil,
	)

	_, err := svc.ResolveSessionGate(gametest.ContextWithParticipantID("manager-1"), &statev1.ResolveSessionGateRequest{
		CampaignId: "c1",
		SessionId:  "s1",
		GateId:     "g1",
		Decision:   "allow",
	})
	if err != nil {
		t.Fatalf("ResolveSessionGate returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if domain.lastCommand.Type != command.Type("session.gate_resolve") {
		t.Fatalf("command type = %s, want %s", domain.lastCommand.Type, "session.gate_resolve")
	}
}
