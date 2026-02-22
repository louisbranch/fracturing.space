package daggerheart

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
)

// --- SwapLoadout tests ---

func TestSwapLoadout_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SwapLoadout(context.Background(), &pb.DaggerheartSwapLoadoutRequest{
		CampaignId: "c1", CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSwapLoadout_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.SwapLoadout(ctx, &pb.DaggerheartSwapLoadoutRequest{
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSwapLoadout_MissingCharacterId(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.SwapLoadout(ctx, &pb.DaggerheartSwapLoadoutRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSwapLoadout_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SwapLoadout(context.Background(), &pb.DaggerheartSwapLoadoutRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSwapLoadout_MissingSwap(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.SwapLoadout(ctx, &pb.DaggerheartSwapLoadoutRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSwapLoadout_MissingCardId(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.SwapLoadout(ctx, &pb.DaggerheartSwapLoadoutRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Swap:        &pb.DaggerheartLoadoutSwapRequest{},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSwapLoadout_NegativeRecallCost(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.SwapLoadout(ctx, &pb.DaggerheartSwapLoadoutRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Swap: &pb.DaggerheartLoadoutSwapRequest{
			CardId:     "card-1",
			RecallCost: -1,
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSwapLoadout_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.SwapLoadout(ctx, &pb.DaggerheartSwapLoadoutRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Swap: &pb.DaggerheartLoadoutSwapRequest{
			CardId:     "card-1",
			RecallCost: 0,
		},
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSwapLoadout_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
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
	svc.stores.Domain = serviceDomain
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
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
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
	spendPayload := daggerheart.CharacterStatePatchedPayload{
		CharacterID:  "char-1",
		StressBefore: &stressBefore,
		StressAfter:  &stressAfter,
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
	svc.stores.Domain = serviceDomain
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

func TestSwapLoadout_UsesDomainEngineForLoadoutSwap(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	stressBefore := 3
	stressAfter := 3
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
		StressBefore: &stressBefore,
		StressAfter:  &stressAfter,
	}
	loadoutJSON, err := json.Marshal(loadoutPayload)
	if err != nil {
		t.Fatalf("encode loadout payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.loadout.swap"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.loadout_swapped"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-swap-1",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   loadoutJSON,
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-swap-1")
	_, err = svc.SwapLoadout(ctx, &pb.DaggerheartSwapLoadoutRequest{
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
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.loadout.swap") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.loadout.swap")
	}
	if domain.commands[0].SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", domain.commands[0].SystemID, daggerheart.SystemID)
	}
	if domain.commands[0].SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", domain.commands[0].SystemVersion, daggerheart.SystemVersion)
	}
	var got struct {
		CharacterID  string `json:"character_id"`
		CardID       string `json:"card_id"`
		From         string `json:"from"`
		To           string `json:"to"`
		RecallCost   int    `json:"recall_cost"`
		StressBefore *int   `json:"stress_before,omitempty"`
		StressAfter  *int   `json:"stress_after,omitempty"`
	}
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode loadout swap command payload: %v", err)
	}
	if got.CharacterID != "char-1" {
		t.Fatalf("command character id = %s, want %s", got.CharacterID, "char-1")
	}
	if got.CardID != "card-1" {
		t.Fatalf("command card id = %s, want %s", got.CardID, "card-1")
	}
	if got.From != "vault" {
		t.Fatalf("command from = %s, want %s", got.From, "vault")
	}
	if got.To != "active" {
		t.Fatalf("command to = %s, want %s", got.To, "active")
	}
	if got.RecallCost != 0 {
		t.Fatalf("command recall cost = %d, want %d", got.RecallCost, 0)
	}
	if got.StressBefore == nil || *got.StressBefore != stressBefore {
		t.Fatalf("command stress before = %v, want %d", got.StressBefore, stressBefore)
	}
	if got.StressAfter == nil || *got.StressAfter != stressAfter {
		t.Fatalf("command stress after = %v, want %d", got.StressAfter, stressAfter)
	}
}

func TestSwapLoadout_UsesDomainEngineForStressSpend(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

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

	spendPayload := daggerheart.CharacterStatePatchedPayload{
		CharacterID:  "char-1",
		StressBefore: &stressBefore,
		StressAfter:  &stressAfter,
	}
	spendJSON, err := json.Marshal(spendPayload)
	if err != nil {
		t.Fatalf("encode stress spend payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.loadout.swap"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.loadout_swapped"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-swap-1",
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
				RequestID:     "req-swap-1",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   spendJSON,
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-swap-1")
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
	if domain.calls != 2 {
		t.Fatalf("expected domain to be called twice, got %d", domain.calls)
	}
	if len(domain.commands) != 2 {
		t.Fatalf("expected 2 domain commands, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.loadout.swap") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.loadout.swap")
	}
	if domain.commands[1].Type != command.Type("sys.daggerheart.stress.spend") {
		t.Fatalf("command type = %s, want %s", domain.commands[1].Type, "sys.daggerheart.stress.spend")
	}
	if domain.commands[1].SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", domain.commands[1].SystemID, daggerheart.SystemID)
	}
	if domain.commands[1].SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", domain.commands[1].SystemVersion, daggerheart.SystemVersion)
	}
	var got struct {
		CharacterID string `json:"character_id"`
		Amount      int    `json:"amount"`
		Before      int    `json:"before"`
		After       int    `json:"after"`
		Source      string `json:"source"`
	}
	if err := json.Unmarshal(domain.commands[1].PayloadJSON, &got); err != nil {
		t.Fatalf("decode stress spend command payload: %v", err)
	}
	if got.CharacterID != "char-1" {
		t.Fatalf("command character id = %s, want %s", got.CharacterID, "char-1")
	}
	if got.Amount != 1 {
		t.Fatalf("command amount = %d, want %d", got.Amount, 1)
	}
	if got.Before != stressBefore {
		t.Fatalf("command before = %d, want %d", got.Before, stressBefore)
	}
	if got.After != stressAfter {
		t.Fatalf("command after = %d, want %d", got.After, stressAfter)
	}
	if got.Source != "loadout_swap" {
		t.Fatalf("command source = %s, want %s", got.Source, "loadout_swap")
	}
	if resp.State.Stress != int32(stressAfter) {
		t.Fatalf("response stress = %d, want %d", resp.State.Stress, stressAfter)
	}
}

func TestSwapLoadout_InRestSkipsRecallCost(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
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
	svc.stores.Domain = serviceDomain
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
