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

// --- UpdateCountdown tests ---

func TestUpdateCountdown_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.UpdateCountdown(context.Background(), &pb.DaggerheartUpdateCountdownRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestUpdateCountdown_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.UpdateCountdown(context.Background(), &pb.DaggerheartUpdateCountdownRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateCountdown_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.UpdateCountdown(context.Background(), &pb.DaggerheartUpdateCountdownRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateCountdown_MissingCountdownId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.UpdateCountdown(context.Background(), &pb.DaggerheartUpdateCountdownRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateCountdown_NoDeltaOrCurrent(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.UpdateCountdown(context.Background(), &pb.DaggerheartUpdateCountdownRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CountdownId: "cd-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDeleteCountdown_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.DeleteCountdown(context.Background(), &pb.DaggerheartDeleteCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestUpdateCountdown_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.UpdateCountdown(context.Background(), &pb.DaggerheartUpdateCountdownRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CountdownId: "cd-1", Delta: 1,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestUpdateCountdown_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	createPayload := daggerheart.CountdownCreatedPayload{
		CountdownID: "cd-update",
		Name:        "Update Test",
		Kind:        daggerheart.CountdownKindProgress,
		Current:     0,
		Max:         4,
		Direction:   daggerheart.CountdownDirectionIncrease,
		Looping:     false,
	}
	createPayloadJSON, err := json.Marshal(createPayload)
	if err != nil {
		t.Fatalf("encode countdown create payload: %v", err)
	}
	update, err := daggerheart.ApplyCountdownUpdate(daggerheart.Countdown{
		CampaignID: "camp-1",
		ID:         "cd-update",
		Name:       "Update Test",
		Kind:       daggerheart.CountdownKindProgress,
		Current:    0,
		Max:        4,
		Direction:  daggerheart.CountdownDirectionIncrease,
		Looping:    false,
	}, 1, nil)
	if err != nil {
		t.Fatalf("apply countdown update: %v", err)
	}
	updatePayload := daggerheart.CountdownUpdatedPayload{
		CountdownID: "cd-update",
		Before:      update.Before,
		After:       update.After,
		Delta:       update.Delta,
		Looped:      update.Looped,
	}
	updatePayloadJSON, err := json.Marshal(updatePayload)
	if err != nil {
		t.Fatalf("encode countdown update payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.countdown.create"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.countdown_created"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-countdown-update-create",
				EntityType:    "countdown",
				EntityID:      "cd-update",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   createPayloadJSON,
			}),
		},
		command.Type("sys.daggerheart.countdown.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.countdown_updated"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-countdown-update",
				EntityType:    "countdown",
				EntityID:      "cd-update",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   updatePayloadJSON,
			}),
		},
	}}
	svc.stores.Domain = serviceDomain
	_, err = svc.CreateCountdown(context.Background(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		Name:        "Update Test",
		Kind:        pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS,
		Direction:   pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Max:         4,
		Current:     0,
		CountdownId: "cd-update",
	})
	if err != nil {
		t.Fatalf("CreateCountdown returned error: %v", err)
	}

	resp, err := svc.UpdateCountdown(context.Background(), &pb.DaggerheartUpdateCountdownRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CountdownId: "cd-update", Delta: 1,
	})
	if err != nil {
		t.Fatalf("UpdateCountdown returned error: %v", err)
	}
	if resp.Countdown == nil {
		t.Fatal("expected countdown in response")
	}
	if resp.After != 1 {
		t.Fatalf("after = %d, want 1", resp.After)
	}
}

func TestUpdateCountdown_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)

	dhStore.Countdowns["camp-1:cd-1"] = storage.DaggerheartCountdown{
		CampaignID:  "camp-1",
		CountdownID: "cd-1",
		Name:        "Update",
		Kind:        daggerheart.CountdownKindProgress,
		Current:     2,
		Max:         4,
		Direction:   daggerheart.CountdownDirectionIncrease,
		Looping:     false,
	}
	update, err := daggerheart.ApplyCountdownUpdate(daggerheart.Countdown{
		CampaignID: "camp-1",
		ID:         "cd-1",
		Name:       "Update",
		Kind:       daggerheart.CountdownKindProgress,
		Current:    2,
		Max:        4,
		Direction:  daggerheart.CountdownDirectionIncrease,
		Looping:    false,
	}, 1, nil)
	if err != nil {
		t.Fatalf("apply countdown update: %v", err)
	}
	updatePayload := daggerheart.CountdownUpdatedPayload{
		CountdownID: "cd-1",
		Before:      update.Before,
		After:       update.After,
		Delta:       update.Delta,
		Looped:      update.Looped,
		Reason:      "advance",
	}
	updatePayloadJSON, err := json.Marshal(updatePayload)
	if err != nil {
		t.Fatalf("encode countdown update payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.countdown.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.countdown_updated"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-countdown-update",
				EntityType:    "countdown",
				EntityID:      "cd-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   updatePayloadJSON,
			}),
		},
	}}
	svc.stores.Domain = domain

	resp, err := svc.UpdateCountdown(context.Background(), &pb.DaggerheartUpdateCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
		Delta:       1,
		Reason:      "advance",
	})
	if err != nil {
		t.Fatalf("UpdateCountdown returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if domain.lastCommand.Type != command.Type("sys.daggerheart.countdown.update") {
		t.Fatalf("command type = %s, want %s", domain.lastCommand.Type, "sys.daggerheart.countdown.update")
	}
	if resp.After != int32(update.After) {
		t.Fatalf("after = %d, want %d", resp.After, update.After)
	}
	if resp.Countdown == nil {
		t.Fatal("expected countdown in response")
	}
	if resp.Countdown.Current != int32(update.After) {
		t.Fatalf("current = %d, want %d", resp.Countdown.Current, update.After)
	}
}

// --- DeleteCountdown tests ---
