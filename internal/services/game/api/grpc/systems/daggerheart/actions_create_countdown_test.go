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
	"google.golang.org/grpc/codes"
)

// --- CreateCountdown tests ---

func TestCreateCountdown_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.CreateCountdown(context.Background(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestCreateCountdown_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.CreateCountdown(context.Background(), &pb.DaggerheartCreateCountdownRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCountdown_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.CreateCountdown(context.Background(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCountdown_MissingName(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.CreateCountdown(context.Background(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCountdown_InvalidMax(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.CreateCountdown(context.Background(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		Name:       "Test Countdown",
		Kind:       pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS,
		Direction:  pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Max:        0,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCountdown_CurrentOutOfRange(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.CreateCountdown(context.Background(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		Name:       "Test Countdown",
		Kind:       pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS,
		Direction:  pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Max:        4,
		Current:    5,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCountdown_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.CreateCountdown(context.Background(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		Name:       "Test Countdown",
		Kind:       pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS,
		Direction:  pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Max:        4,
		Current:    0,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestCreateCountdown_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	countdownPayload := daggerheart.CountdownCreatedPayload{
		CountdownID: "cd-1",
		Name:        "Test Countdown",
		Kind:        daggerheart.CountdownKindProgress,
		Current:     0,
		Max:         4,
		Direction:   daggerheart.CountdownDirectionIncrease,
		Looping:     false,
	}
	countdownPayloadJSON, err := json.Marshal(countdownPayload)
	if err != nil {
		t.Fatalf("encode countdown payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.countdown.create"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.countdown_created"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-countdown-success",
				EntityType:    "countdown",
				EntityID:      "cd-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   countdownPayloadJSON,
			}),
		},
	}}
	svc.stores.Domain = serviceDomain
	resp, err := svc.CreateCountdown(context.Background(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		Name:        "Test Countdown",
		Kind:        pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS,
		Direction:   pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Max:         4,
		Current:     0,
		CountdownId: "cd-1",
	})
	if err != nil {
		t.Fatalf("CreateCountdown returned error: %v", err)
	}
	if resp.Countdown == nil {
		t.Fatal("expected countdown in response")
	}
	if resp.Countdown.Name != "Test Countdown" {
		t.Fatalf("name = %q, want Test Countdown", resp.Countdown.Name)
	}
}

func TestCreateCountdown_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	countdownPayload := daggerheart.CountdownCreatedPayload{
		CountdownID: "cd-1",
		Name:        "Signal",
		Kind:        daggerheart.CountdownKindProgress,
		Current:     1,
		Max:         4,
		Direction:   daggerheart.CountdownDirectionIncrease,
		Looping:     true,
	}
	countdownPayloadJSON, err := json.Marshal(countdownPayload)
	if err != nil {
		t.Fatalf("encode countdown payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.countdown.create"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.countdown_created"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-countdown-create",
				EntityType:    "countdown",
				EntityID:      "cd-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   countdownPayloadJSON,
			}),
		},
	}}

	svc.stores.Domain = domain

	resp, err := svc.CreateCountdown(context.Background(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
		Name:        "Signal",
		Kind:        pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS,
		Direction:   pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Max:         4,
		Current:     1,
		Looping:     true,
	})
	if err != nil {
		t.Fatalf("CreateCountdown returned error: %v", err)
	}
	if resp.Countdown == nil {
		t.Fatal("expected countdown in response")
	}
	if resp.Countdown.CountdownId != "cd-1" {
		t.Fatalf("countdown_id = %q, want cd-1", resp.Countdown.CountdownId)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if domain.lastCommand.Type != command.Type("sys.daggerheart.countdown.create") {
		t.Fatalf("command type = %s, want %s", domain.lastCommand.Type, "sys.daggerheart.countdown.create")
	}
	if got := len(eventStore.Events["camp-1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.Events["camp-1"][0].Type != event.Type("sys.daggerheart.countdown_created") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["camp-1"][0].Type, event.Type("sys.daggerheart.countdown_created"))
	}
}
