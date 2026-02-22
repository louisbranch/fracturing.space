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

func TestDeleteCountdown_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.DeleteCountdown(context.Background(), &pb.DaggerheartDeleteCountdownRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestDeleteCountdown_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.DeleteCountdown(context.Background(), &pb.DaggerheartDeleteCountdownRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDeleteCountdown_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.DeleteCountdown(context.Background(), &pb.DaggerheartDeleteCountdownRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDeleteCountdown_MissingCountdownId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.DeleteCountdown(context.Background(), &pb.DaggerheartDeleteCountdownRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDeleteCountdown_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.Countdowns["camp-1:cd-delete"] = storage.DaggerheartCountdown{
		CampaignID:  "camp-1",
		CountdownID: "cd-delete",
		Name:        "Delete Test",
		Kind:        daggerheart.CountdownKindConsequence,
		Current:     0,
		Max:         4,
		Direction:   daggerheart.CountdownDirectionIncrease,
		Looping:     false,
	}
	deletePayload := daggerheart.CountdownDeletedPayload{CountdownID: "cd-delete"}
	deletePayloadJSON, err := json.Marshal(deletePayload)
	if err != nil {
		t.Fatalf("encode countdown delete payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.countdown.delete"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.countdown_deleted"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-countdown-delete-success",
				EntityType:    "countdown",
				EntityID:      "cd-delete",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   deletePayloadJSON,
			}),
		},
	}}
	svc.stores.Domain = serviceDomain
	resp, err := svc.DeleteCountdown(context.Background(), &pb.DaggerheartDeleteCountdownRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CountdownId: "cd-delete",
	})
	if err != nil {
		t.Fatalf("DeleteCountdown returned error: %v", err)
	}
	if resp.CountdownId != "cd-delete" {
		t.Fatalf("countdown_id = %q, want %q", resp.CountdownId, "cd-delete")
	}
}

func TestDeleteCountdown_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)

	dhStore.Countdowns["camp-1:cd-1"] = storage.DaggerheartCountdown{
		CampaignID:  "camp-1",
		CountdownID: "cd-1",
		Name:        "Cleanup",
		Kind:        daggerheart.CountdownKindConsequence,
		Current:     0,
		Max:         4,
		Direction:   daggerheart.CountdownDirectionIncrease,
		Looping:     false,
	}
	deletePayload := daggerheart.CountdownDeletedPayload{CountdownID: "cd-1", Reason: "cleanup"}
	deletePayloadJSON, err := json.Marshal(deletePayload)
	if err != nil {
		t.Fatalf("encode countdown delete payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.countdown.delete"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.countdown_deleted"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-countdown-delete",
				EntityType:    "countdown",
				EntityID:      "cd-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   deletePayloadJSON,
			}),
		},
	}}

	svc.stores.Domain = domain

	resp, err := svc.DeleteCountdown(context.Background(), &pb.DaggerheartDeleteCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
		Reason:      "cleanup",
	})
	if err != nil {
		t.Fatalf("DeleteCountdown returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if domain.lastCommand.Type != command.Type("sys.daggerheart.countdown.delete") {
		t.Fatalf("command type = %s, want %s", domain.lastCommand.Type, "sys.daggerheart.countdown.delete")
	}
	if resp.CountdownId != "cd-1" {
		t.Fatalf("countdown_id = %q, want cd-1", resp.CountdownId)
	}
	if _, err := dhStore.GetDaggerheartCountdown(context.Background(), "camp-1", "cd-1"); err == nil {
		t.Fatal("expected countdown to be deleted")
	}
}
