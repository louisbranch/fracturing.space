package daggerheart

import (
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
)

func TestSwapLoadout_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := testTimestamp
	loadoutPayload := struct {
		CharacterID  string `json:"character_id"`
		CardID       string `json:"card_id"`
		From         string `json:"from"`
		To           string `json:"to"`
		RecallCost   int    `json:"recall_cost"`
		StressBefore *int   `json:"stress_before,omitempty"`
		StressAfter  *int   `json:"stress_after,omitempty"`
	}{
		CharacterID:  "char-1",
		CardID:       "card-1",
		From:         "vault",
		To:           "active",
		RecallCost:   0,
		StressBefore: optionalInt(3),
		StressAfter:  optionalInt(3),
	}
	loadoutJSON, err := json.Marshal(loadoutPayload)
	if err != nil {
		t.Fatalf("encode loadout payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.loadout.swap"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.loadout_swapped"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-swap-success",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   loadoutJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = serviceDomain
	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-swap-success")
	resp, err := svc.SwapLoadout(ctx, &pb.DaggerheartSwapLoadoutRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Swap: &pb.DaggerheartLoadoutSwapRequest{
			CardId:     "card-1",
			RecallCost: 0,
		},
	})
	if err != nil {
		t.Fatalf("SwapLoadout returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character_id = %q, want char-1", resp.CharacterId)
	}
	if resp.State == nil {
		t.Fatal("expected state in response")
	}
}

func TestSwapLoadout_WithRecallCost(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := testTimestamp
	stressBefore := 3
	stressAfter := 2
	loadoutPayload := struct {
		CharacterID  string `json:"character_id"`
		CardID       string `json:"card_id"`
		From         string `json:"from"`
		To           string `json:"to"`
		RecallCost   int    `json:"recall_cost"`
		StressBefore *int   `json:"stress_before,omitempty"`
		StressAfter  *int   `json:"stress_after,omitempty"`
	}{
		CharacterID:  "char-1",
		CardID:       "card-1",
		From:         "vault",
		To:           "active",
		RecallCost:   1,
		StressBefore: &stressBefore,
		StressAfter:  &stressAfter,
	}
	loadoutJSON, err := json.Marshal(loadoutPayload)
	if err != nil {
		t.Fatalf("encode loadout payload: %v", err)
	}
	spendPayload := daggerheartpayload.CharacterStatePatchedPayload{
		CharacterID: "char-1",
		Stress:      &stressAfter,
	}
	spendJSON, err := json.Marshal(spendPayload)
	if err != nil {
		t.Fatalf("encode stress spend payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.loadout.swap"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.loadout_swapped"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-swap-with-cost",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   loadoutJSON,
			}),
		},
		command.Type("sys.daggerheart.stress.spend"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-swap-with-cost",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   spendJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = serviceDomain
	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-swap-with-cost")
	resp, err := svc.SwapLoadout(ctx, &pb.DaggerheartSwapLoadoutRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Swap: &pb.DaggerheartLoadoutSwapRequest{
			CardId:     "card-1",
			RecallCost: 1,
		},
	})
	if err != nil {
		t.Fatalf("SwapLoadout returned error: %v", err)
	}
	if resp.State.Stress != 2 {
		t.Fatalf("stress = %d, want 2 (3 - 1 recall cost)", resp.State.Stress)
	}
}

func TestSwapLoadout_InRestSkipsRecallCost(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := testTimestamp
	loadoutPayload := struct {
		CharacterID  string `json:"character_id"`
		CardID       string `json:"card_id"`
		From         string `json:"from"`
		To           string `json:"to"`
		RecallCost   int    `json:"recall_cost"`
		StressBefore *int   `json:"stress_before,omitempty"`
		StressAfter  *int   `json:"stress_after,omitempty"`
	}{
		CharacterID:  "char-1",
		CardID:       "card-1",
		From:         "vault",
		To:           "active",
		RecallCost:   2,
		StressBefore: optionalInt(3),
		StressAfter:  optionalInt(3),
	}
	loadoutJSON, err := json.Marshal(loadoutPayload)
	if err != nil {
		t.Fatalf("encode loadout payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.loadout.swap"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.loadout_swapped"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-swap-rest",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   loadoutJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = serviceDomain
	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-swap-rest")
	resp, err := svc.SwapLoadout(ctx, &pb.DaggerheartSwapLoadoutRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Swap: &pb.DaggerheartLoadoutSwapRequest{
			CardId:     "card-1",
			RecallCost: 2,
			InRest:     true,
		},
	})
	if err != nil {
		t.Fatalf("SwapLoadout returned error: %v", err)
	}
	if resp.State.Stress != 3 {
		t.Fatalf("stress = %d, want 3 (in-rest should skip recall cost)", resp.State.Stress)
	}
}
