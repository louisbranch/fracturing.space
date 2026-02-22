package daggerheart

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

// --- ApplyGmMove tests ---

func TestApplyGmMove_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyGmMove_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		SessionId: "sess-1", Move: "test_move",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyGmMove_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1", Move: "test_move",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyGmMove_MissingMove(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyGmMove_NegativeFearSpent(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Move: "test_move", FearSpent: -1,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyGmMove_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.Snapshots["camp-1"] = storage.DaggerheartSnapshot{CampaignID: "camp-1", GMFear: 3}
	ctx := context.Background()
	_, err := svc.ApplyGmMove(ctx, &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Move: "change_environment", FearSpent: 1,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyGmMove_Success(t *testing.T) {
	svc := newActionTestService()
	domain := &fakeDomainEngine{}
	svc.stores.Domain = domain
	ctx := context.Background()
	resp, err := svc.ApplyGmMove(ctx, &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Move: "change_environment",
	})
	if err != nil {
		t.Fatalf("ApplyGmMove returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response")
	}
	if domain.calls != 0 {
		t.Fatalf("expected no domain calls, got %d", domain.calls)
	}
}

func TestApplyGmMove_WithFearSpent(t *testing.T) {
	svc := newActionTestService()
	// Pre-populate GM fear in snapshot
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.Snapshots["camp-1"] = storage.DaggerheartSnapshot{CampaignID: "camp-1", GMFear: 3}
	eventStore := svc.stores.Event.(*fakeEventStore)
	gmPayload := daggerheart.GMFearSetPayload{After: optionalInt(2), Reason: "gm_move"}
	gmPayloadJSON, err := json.Marshal(gmPayload)
	if err != nil {
		t.Fatalf("encode gm fear payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.gm_fear.set"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.gm_fear_changed"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-gm-move-fear",
				EntityType:    "campaign",
				EntityID:      "camp-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   gmPayloadJSON,
			}),
		},
	}}
	svc.stores.Domain = serviceDomain
	ctx := context.Background()
	resp, err := svc.ApplyGmMove(ctx, &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Move: "change_environment", FearSpent: 1,
	})
	if err != nil {
		t.Fatalf("ApplyGmMove returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.GetGmFearBefore() != 3 {
		t.Fatalf("expected gm fear before = 3, got %d", resp.GetGmFearBefore())
	}
	if resp.GetGmFearAfter() != 2 {
		t.Fatalf("expected gm fear after = 2, got %d", resp.GetGmFearAfter())
	}
	if len(eventStore.Events["camp-1"]) != 1 {
		t.Fatalf("expected 1 event, got %d", len(eventStore.Events["camp-1"]))
	}
	if eventStore.Events["camp-1"][0].Type != event.Type("sys.daggerheart.gm_fear_changed") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["camp-1"][0].Type, event.Type("sys.daggerheart.gm_fear_changed"))
	}
}

func TestApplyGmMove_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.Snapshots["camp-1"] = storage.DaggerheartSnapshot{CampaignID: "camp-1", GMFear: 2}
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.gm_fear.set"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.gm_fear_changed"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				EntityType:    "campaign",
				EntityID:      "camp-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   []byte(`{"before":2,"after":1,"reason":"gm_move"}`),
			}),
		},
	}}

	svc.stores.Domain = domain

	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		Move:       "change_environment",
		FearSpent:  1,
	})
	if err != nil {
		t.Fatalf("ApplyGmMove returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.gm_fear.set") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.gm_fear.set")
	}
	if domain.commands[0].SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", domain.commands[0].SystemID, daggerheart.SystemID)
	}
	if domain.commands[0].SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", domain.commands[0].SystemVersion, daggerheart.SystemVersion)
	}
	var got struct {
		After int `json:"after"`
	}
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode gm fear command payload: %v", err)
	}
	if got.After != 1 {
		t.Fatalf("command fear value = %d, want 1", got.After)
	}
	if got := len(eventStore.Events["camp-1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.Events["camp-1"][0].Type != event.Type("sys.daggerheart.gm_fear_changed") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["camp-1"][0].Type, event.Type("sys.daggerheart.gm_fear_changed"))
	}
}
