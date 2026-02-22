package daggerheart

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

// --- ApplyDowntimeMove tests ---

func TestApplyDowntimeMove_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyDowntimeMove(context.Background(), &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId: "c1", CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyDowntimeMove_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDowntimeMove(ctx, &pb.DaggerheartApplyDowntimeMoveRequest{
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDowntimeMove_MissingCharacterId(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDowntimeMove(ctx, &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDowntimeMove_CampaignNotFound(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDowntimeMove(ctx, &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId: "nonexistent", CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyDowntimeMove_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyDowntimeMove(context.Background(), &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDowntimeMove_MissingMove(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDowntimeMove(ctx, &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDowntimeMove_UnspecifiedMove(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDowntimeMove(ctx, &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move: &pb.DaggerheartDowntimeRequest{
			Move: pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_UNSPECIFIED,
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDowntimeMove_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDowntimeMove(ctx, &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move: &pb.DaggerheartDowntimeRequest{
			Move: pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_CLEAR_ALL_STRESS,
		},
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyDowntimeMove_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	current := dhStore.States["camp-1:char-1"]
	profile := dhStore.Profiles["camp-1:char-1"]
	state := daggerheart.NewCharacterState(daggerheart.CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          current.Hp,
		HPMax:       profile.HpMax,
		Hope:        current.Hope,
		HopeMax:     current.HopeMax,
		Stress:      current.Stress,
		StressMax:   profile.StressMax,
		Armor:       current.Armor,
		ArmorMax:    profile.ArmorMax,
		LifeState:   current.LifeState,
	})
	move, err := daggerheartDowntimeMoveFromProto(pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_CLEAR_ALL_STRESS)
	if err != nil {
		t.Fatalf("map downtime move: %v", err)
	}
	result := daggerheart.ApplyDowntimeMove(state, move, daggerheart.DowntimeOptions{})
	moveName := daggerheartDowntimeMoveToString(move)
	payload := daggerheart.DowntimeMoveAppliedPayload{
		CharacterID:  "char-1",
		Move:         moveName,
		HopeBefore:   &result.HopeBefore,
		HopeAfter:    &result.HopeAfter,
		StressBefore: &result.StressBefore,
		StressAfter:  &result.StressAfter,
		ArmorBefore:  &result.ArmorBefore,
		ArmorAfter:   &result.ArmorAfter,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode downtime move payload: %v", err)
	}

	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.downtime_move.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.downtime_move_applied"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-downtime-success",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = serviceDomain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-downtime-success")
	resp, err := svc.ApplyDowntimeMove(ctx, &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move: &pb.DaggerheartDowntimeRequest{
			Move: pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_CLEAR_ALL_STRESS,
		},
	})
	if err != nil {
		t.Fatalf("ApplyDowntimeMove returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character_id = %q, want char-1", resp.CharacterId)
	}
	if resp.State == nil {
		t.Fatal("expected state in response")
	}
	if resp.State.Stress != 0 {
		t.Fatalf("stress = %d, want 0", resp.State.Stress)
	}
}

func TestApplyDowntimeMove_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	current := dhStore.States["camp-1:char-1"]
	profile := dhStore.Profiles["camp-1:char-1"]
	state := daggerheart.NewCharacterState(daggerheart.CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          current.Hp,
		HPMax:       profile.HpMax,
		Hope:        current.Hope,
		HopeMax:     current.HopeMax,
		Stress:      current.Stress,
		StressMax:   profile.StressMax,
		Armor:       current.Armor,
		ArmorMax:    profile.ArmorMax,
		LifeState:   current.LifeState,
	})
	move, err := daggerheartDowntimeMoveFromProto(pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_CLEAR_ALL_STRESS)
	if err != nil {
		t.Fatalf("map downtime move: %v", err)
	}
	result := daggerheart.ApplyDowntimeMove(state, move, daggerheart.DowntimeOptions{})
	moveName := daggerheartDowntimeMoveToString(move)
	payload := daggerheart.DowntimeMoveAppliedPayload{
		CharacterID:  "char-1",
		Move:         moveName,
		HopeBefore:   &result.HopeBefore,
		HopeAfter:    &result.HopeAfter,
		StressBefore: &result.StressBefore,
		StressAfter:  &result.StressAfter,
		ArmorBefore:  &result.ArmorBefore,
		ArmorAfter:   &result.ArmorAfter,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode downtime move payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.downtime_move.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.downtime_move_applied"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-downtime-move",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-downtime-move")
	_, err = svc.ApplyDowntimeMove(ctx, &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move: &pb.DaggerheartDowntimeRequest{
			Move: pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_CLEAR_ALL_STRESS,
		},
	})
	if err != nil {
		t.Fatalf("ApplyDowntimeMove returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.downtime_move.apply") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.downtime_move.apply")
	}
	if domain.commands[0].SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", domain.commands[0].SystemID, daggerheart.SystemID)
	}
	if domain.commands[0].SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", domain.commands[0].SystemVersion, daggerheart.SystemVersion)
	}
	var got struct {
		CharacterID  string `json:"character_id"`
		Move         string `json:"move"`
		StressBefore *int   `json:"stress_before"`
		StressAfter  *int   `json:"stress_after"`
	}
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode downtime move command payload: %v", err)
	}
	if got.CharacterID != "char-1" {
		t.Fatalf("command character id = %s, want %s", got.CharacterID, "char-1")
	}
	if got.Move != moveName {
		t.Fatalf("command move = %s, want %s", got.Move, moveName)
	}
	if got.StressBefore == nil || *got.StressBefore != result.StressBefore {
		t.Fatalf("command stress before = %v, want %d", got.StressBefore, result.StressBefore)
	}
	if got.StressAfter == nil || *got.StressAfter != result.StressAfter {
		t.Fatalf("command stress after = %v, want %d", got.StressAfter, result.StressAfter)
	}
}

// --- ApplyTemporaryArmor tests ---

func TestApplyTemporaryArmor_MissingArmor(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyTemporaryArmor(ctx, &pb.DaggerheartApplyTemporaryArmorRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyTemporaryArmor_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	ctx := contextWithSessionID("sess-1")

	tempPayload := struct {
		CharacterID string `json:"character_id"`
		Source      string `json:"source"`
		Duration    string `json:"duration"`
		Amount      int    `json:"amount"`
		SourceID    string `json:"source_id"`
	}{
		CharacterID: "char-1",
		Source:      "ritual",
		Duration:    "short_rest",
		Amount:      2,
		SourceID:    "blessing:1",
	}
	tempPayloadJSON, err := json.Marshal(tempPayload)
	if err != nil {
		t.Fatalf("encode temporary armor payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_temporary_armor.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_temporary_armor_applied"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-temporary-armor",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   tempPayloadJSON,
			}),
		},
	}}
	svc.stores.Domain = serviceDomain

	ctx = grpcmeta.WithRequestID(ctx, "req-temporary-armor")
	resp, err := svc.ApplyTemporaryArmor(ctx, &pb.DaggerheartApplyTemporaryArmorRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Armor: &pb.DaggerheartTemporaryArmor{
			Source:   "ritual",
			Duration: "short_rest",
			Amount:   2,
			SourceId: "blessing:1",
		},
	})
	if err != nil {
		t.Fatalf("ApplyTemporaryArmor returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character_id = %q, want char-1", resp.CharacterId)
	}
	if resp.State == nil {
		t.Fatal("expected state in response")
	}
	if resp.State.Armor != 2 {
		t.Fatalf("armor = %d, want 2", resp.State.Armor)
	}
	if serviceDomain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", serviceDomain.calls)
	}
	if len(serviceDomain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(serviceDomain.commands))
	}
	if serviceDomain.commands[0].Type != command.Type("sys.daggerheart.character_temporary_armor.apply") {
		t.Fatalf("command type = %s, want %s", serviceDomain.commands[0].Type, "sys.daggerheart.character_temporary_armor.apply")
	}
	var got struct {
		CharacterID string `json:"character_id"`
		Source      string `json:"source"`
		Duration    string `json:"duration"`
		Amount      int    `json:"amount"`
		SourceID    string `json:"source_id"`
	}
	if err := json.Unmarshal(serviceDomain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode temporary armor command payload: %v", err)
	}
	if got.CharacterID != "char-1" {
		t.Fatalf("command character id = %s, want %s", got.CharacterID, "char-1")
	}
	if got.Source != "ritual" {
		t.Fatalf("command source = %s, want %s", got.Source, "ritual")
	}
	if got.Duration != "short_rest" {
		t.Fatalf("command duration = %s, want %s", got.Duration, "short_rest")
	}
	if got.Amount != 2 {
		t.Fatalf("command amount = %d, want %d", got.Amount, 2)
	}
	if got.SourceID != "blessing:1" {
		t.Fatalf("command source_id = %s, want %s", got.SourceID, "blessing:1")
	}
}

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

// --- ApplyDeathMove tests ---

func TestApplyDeathMove_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyDeathMove(context.Background(), &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId: "c1", CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyDeathMove_MissingSeedFunc(t *testing.T) {
	svc := &DaggerheartService{
		stores: Stores{
			Campaign:    newFakeCampaignStore(),
			Daggerheart: newFakeDaggerheartStore(),
			Event:       newFakeActionEventStore(),
		},
	}
	_, err := svc.ApplyDeathMove(context.Background(), &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId: "c1", CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyDeathMove_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDeathMove(ctx, &pb.DaggerheartApplyDeathMoveRequest{
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDeathMove_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyDeathMove(context.Background(), &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDeathMove_UnspecifiedMove(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDeathMove(ctx, &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_UNSPECIFIED,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDeathMove_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.States["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          0,
		Hope:        2,
		HopeMax:     daggerheart.HopeMax,
		Stress:      1,
		LifeState:   daggerheart.LifeStateAlive,
	}
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDeathMove(ctx, &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyDeathMove_HpClearOnNonRiskItAll(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	hpClear := int32(1)
	_, err := svc.ApplyDeathMove(ctx, &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH,
		HpClear:     &hpClear,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDeathMove_HpNotZero(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDeathMove(ctx, &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestApplyDeathMove_AlreadyDead(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	// Set up state with hp=0 and life_state=dead
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.States["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          0,
		LifeState:   daggerheart.LifeStateDead,
	}
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDeathMove(ctx, &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestApplyDeathMove_AvoidDeath_Success(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.States["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          0,
		Hope:        2,
		HopeMax:     daggerheart.HopeMax,
		Stress:      1,
		LifeState:   daggerheart.LifeStateAlive,
	}
	profile := dhStore.Profiles["camp-1:char-1"]
	move, err := daggerheartDeathMoveFromProto(pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH)
	if err != nil {
		t.Fatalf("map death move: %v", err)
	}
	hpMax := profile.HpMax
	if hpMax == 0 {
		hpMax = daggerheart.PCHpMax
	}
	stressMax := profile.StressMax
	if stressMax < 0 {
		stressMax = 0
	}
	hopeMax := daggerheart.HopeMax
	level := profile.Level
	if level == 0 {
		level = daggerheart.PCLevelDefault
	}
	result, err := daggerheart.ResolveDeathMove(daggerheart.DeathMoveInput{
		Move:      move,
		Level:     level,
		HP:        0,
		HPMax:     hpMax,
		Hope:      2,
		HopeMax:   hopeMax,
		Stress:    1,
		StressMax: stressMax,
		Seed:      42,
	})
	if err != nil {
		t.Fatalf("resolve death move: %v", err)
	}
	lifeStateBefore := daggerheart.LifeStateAlive
	payload := daggerheart.CharacterStatePatchedPayload{
		CharacterID:     "char-1",
		LifeStateBefore: &lifeStateBefore,
		LifeStateAfter:  &result.LifeState,
		HPBefore:        &result.HPBefore,
		HPAfter:         &result.HPAfter,
		HopeBefore:      &result.HopeBefore,
		HopeAfter:       &result.HopeAfter,
		HopeMaxBefore:   &result.HopeMaxBefore,
		HopeMaxAfter:    &result.HopeMaxAfter,
		StressBefore:    &result.StressBefore,
		StressAfter:     &result.StressAfter,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode death move payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: svc.stores.Event, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-death-success",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = serviceDomain
	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-death-success")
	resp, err := svc.ApplyDeathMove(ctx, &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH,
	})
	if err != nil {
		t.Fatalf("ApplyDeathMove returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character_id = %q, want char-1", resp.CharacterId)
	}
	if resp.Result == nil {
		t.Fatal("expected result in response")
	}
}

func TestApplyDeathMove_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	state := storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          0,
		Hope:        2,
		HopeMax:     daggerheart.HopeMax,
		Stress:      1,
		LifeState:   daggerheart.LifeStateAlive,
	}
	dhStore.States["camp-1:char-1"] = state
	profile := dhStore.Profiles["camp-1:char-1"]
	move, err := daggerheartDeathMoveFromProto(pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH)
	if err != nil {
		t.Fatalf("map death move: %v", err)
	}

	hpMax := profile.HpMax
	if hpMax == 0 {
		hpMax = daggerheart.PCHpMax
	}
	stressMax := profile.StressMax
	if stressMax < 0 {
		stressMax = 0
	}
	hopeMax := state.HopeMax
	if hopeMax == 0 {
		hopeMax = daggerheart.HopeMax
	}
	level := profile.Level
	if level == 0 {
		level = daggerheart.PCLevelDefault
	}

	result, err := daggerheart.ResolveDeathMove(daggerheart.DeathMoveInput{
		Move:      move,
		Level:     level,
		HP:        state.Hp,
		HPMax:     hpMax,
		Hope:      state.Hope,
		HopeMax:   hopeMax,
		Stress:    state.Stress,
		StressMax: stressMax,
		Seed:      42,
	})
	if err != nil {
		t.Fatalf("resolve death move: %v", err)
	}

	lifeStateBefore := state.LifeState
	if lifeStateBefore == "" {
		lifeStateBefore = daggerheart.LifeStateAlive
	}
	payload := daggerheart.CharacterStatePatchedPayload{
		CharacterID:     "char-1",
		LifeStateBefore: &lifeStateBefore,
		LifeStateAfter:  &result.LifeState,
		HPBefore:        &result.HPBefore,
		HPAfter:         &result.HPAfter,
		HopeBefore:      &result.HopeBefore,
		HopeAfter:       &result.HopeAfter,
		HopeMaxBefore:   &result.HopeMaxBefore,
		HopeMaxAfter:    &result.HopeMaxAfter,
		StressBefore:    &result.StressBefore,
		StressAfter:     &result.StressAfter,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode death move payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-death-move",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-death-move")
	_, err = svc.ApplyDeathMove(ctx, &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH,
	})
	if err != nil {
		t.Fatalf("ApplyDeathMove returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.character_state.patch") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.character_state.patch")
	}
	if domain.commands[0].SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", domain.commands[0].SystemID, daggerheart.SystemID)
	}
	if domain.commands[0].SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", domain.commands[0].SystemVersion, daggerheart.SystemVersion)
	}
	var got struct {
		CharacterID    string `json:"character_id"`
		LifeStateAfter string `json:"life_state_after"`
	}
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode death move command payload: %v", err)
	}
	if got.CharacterID != "char-1" {
		t.Fatalf("command character id = %s, want %s", got.CharacterID, "char-1")
	}
	if got.LifeStateAfter == "" {
		t.Fatal("expected life_state_after in command payload")
	}
}

// --- ResolveBlazeOfGlory tests ---

func TestResolveBlazeOfGlory_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ResolveBlazeOfGlory(context.Background(), &pb.DaggerheartResolveBlazeOfGloryRequest{
		CampaignId: "c1", CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestResolveBlazeOfGlory_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ResolveBlazeOfGlory(ctx, &pb.DaggerheartResolveBlazeOfGloryRequest{
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestResolveBlazeOfGlory_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ResolveBlazeOfGlory(context.Background(), &pb.DaggerheartResolveBlazeOfGloryRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestResolveBlazeOfGlory_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.States["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          0,
		LifeState:   daggerheart.LifeStateBlazeOfGlory,
	}
	charStore := svc.stores.Character.(*fakeCharacterStore)
	charStore.Characters["camp-1:char-1"] = storage.CharacterRecord{
		ID:         "char-1",
		CampaignID: "camp-1",
		Name:       "Hero",
		Kind:       character.KindPC,
	}
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ResolveBlazeOfGlory(ctx, &pb.DaggerheartResolveBlazeOfGloryRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestResolveBlazeOfGlory_CharacterAlreadyDead(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.States["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		LifeState:   daggerheart.LifeStateDead,
	}
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ResolveBlazeOfGlory(ctx, &pb.DaggerheartResolveBlazeOfGloryRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestResolveBlazeOfGlory_NotInBlazeState(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ResolveBlazeOfGlory(ctx, &pb.DaggerheartResolveBlazeOfGloryRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestResolveBlazeOfGlory_Success(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.States["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          0,
		LifeState:   daggerheart.LifeStateBlazeOfGlory,
	}
	charStore := svc.stores.Character.(*fakeCharacterStore)
	charStore.Characters["camp-1:char-1"] = storage.CharacterRecord{
		ID:         "char-1",
		CampaignID: "camp-1",
		Name:       "Hero",
		Kind:       character.KindPC,
	}
	payload := daggerheart.CharacterStatePatchedPayload{
		CharacterID: "char-1",
		LifeStateBefore: func() *string {
			l := daggerheart.LifeStateBlazeOfGlory
			return &l
		}(),
		LifeStateAfter: func() *string {
			l := daggerheart.LifeStateDead
			return &l
		}(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode blaze of glory payload: %v", err)
	}
	eventStore := svc.stores.Event.(*fakeEventStore)
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-blaze-success",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
		command.Type("character.delete"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("character.deleted"),
				Timestamp:   time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-blaze-success",
				EntityType:  "character",
				EntityID:    "char-1",
				PayloadJSON: []byte(`{"character_id":"char-1","reason":"blaze_of_glory"}`),
			}),
		},
	}}
	svc.stores.Domain = serviceDomain
	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-blaze-success")
	resp, err := svc.ResolveBlazeOfGlory(ctx, &pb.DaggerheartResolveBlazeOfGloryRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	if err != nil {
		t.Fatalf("ResolveBlazeOfGlory returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character_id = %q, want char-1", resp.CharacterId)
	}
	if resp.Result == nil {
		t.Fatal("expected result in response")
	}
	if resp.Result.LifeState != pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD {
		t.Fatalf("life_state = %v, want DEAD", resp.Result.LifeState)
	}
}

func TestResolveBlazeOfGlory_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.States["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          0,
		LifeState:   daggerheart.LifeStateBlazeOfGlory,
	}
	charStore := svc.stores.Character.(*fakeCharacterStore)
	charStore.Characters["camp-1:char-1"] = storage.CharacterRecord{
		ID:         "char-1",
		CampaignID: "camp-1",
		Name:       "Hero",
		Kind:       character.KindPC,
	}

	payload := daggerheart.CharacterStatePatchedPayload{
		CharacterID: "char-1",
		LifeStateBefore: func() *string {
			l := daggerheart.LifeStateBlazeOfGlory
			return &l
		}(),
		LifeStateAfter: func() *string {
			l := daggerheart.LifeStateDead
			return &l
		}(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode blaze of glory payload: %v", err)
	}

	eventStore := svc.stores.Event.(*fakeEventStore)
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-blaze",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
		command.Type("character.delete"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("character.deleted"),
				Timestamp:   time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-blaze",
				EntityType:  "character",
				EntityID:    "char-1",
				PayloadJSON: []byte(`{"character_id":"char-1","reason":"blaze_of_glory"}`),
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-blaze")
	_, err = svc.ResolveBlazeOfGlory(ctx, &pb.DaggerheartResolveBlazeOfGloryRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
	})
	if err != nil {
		t.Fatalf("ResolveBlazeOfGlory returned error: %v", err)
	}
	if domain.calls != 2 {
		t.Fatalf("expected domain to be called twice, got %d", domain.calls)
	}
	if len(domain.commands) != 2 {
		t.Fatalf("expected 2 domain commands, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.character_state.patch") {
		t.Fatalf("command[0] type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.character_state.patch")
	}
	if domain.commands[1].Type != command.Type("character.delete") {
		t.Fatalf("command[1] type = %s, want %s", domain.commands[1].Type, "character.delete")
	}
	if got := len(eventStore.Events["camp-1"]); got != 2 {
		t.Fatalf("expected 2 events, got %d", got)
	}
	if eventStore.Events["camp-1"][0].Type != event.Type("sys.daggerheart.character_state_patched") {
		t.Fatalf("event[0] type = %s, want %s", eventStore.Events["camp-1"][0].Type, event.Type("sys.daggerheart.character_state_patched"))
	}
	if eventStore.Events["camp-1"][1].Type != event.Type("character.deleted") {
		t.Fatalf("event[1] type = %s, want %s", eventStore.Events["camp-1"][1].Type, event.Type("character.deleted"))
	}
}

// --- ApplyDamage tests ---

func TestApplyDamage_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyDamage(context.Background(), &pb.DaggerheartApplyDamageRequest{
		CampaignId: "c1", CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyDamage_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     3,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyDamage_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{CharacterId: "ch1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDamage_MissingCharacterId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{CampaignId: "camp-1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDamage_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyDamage(context.Background(), &pb.DaggerheartApplyDamageRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDamage_MissingDamage(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDamage_NegativeAmount(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     -1,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDamage_UnspecifiedType(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     2,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED,
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDamage_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	profile := dhStore.Profiles["camp-1:char-1"]
	state := dhStore.States["camp-1:char-1"]
	damage := &pb.DaggerheartDamageRequest{
		Amount:     3,
		DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
	}
	result, mitigated, err := applyDaggerheartDamage(damage, profile, state)
	if err != nil {
		t.Fatalf("apply daggerheart damage: %v", err)
	}

	hpBefore := result.HPBefore
	hpAfter := result.HPAfter
	armorBefore := result.ArmorBefore
	armorAfter := result.ArmorAfter
	sourceCharacterIDs := normalizeTargets(damage.GetSourceCharacterIds())
	payload := daggerheart.DamageAppliedPayload{
		CharacterID:        "char-1",
		HpBefore:           &hpBefore,
		HpAfter:            &hpAfter,
		ArmorBefore:        &armorBefore,
		ArmorAfter:         &armorAfter,
		ArmorSpent:         result.ArmorSpent,
		Severity:           daggerheartSeverityToString(result.Result.Severity),
		Marks:              result.Result.Marks,
		DamageType:         daggerheartDamageTypeToString(damage.DamageType),
		RollSeq:            nil,
		ResistPhysical:     damage.ResistPhysical,
		ResistMagic:        damage.ResistMagic,
		ImmunePhysical:     damage.ImmunePhysical,
		ImmuneMagic:        damage.ImmuneMagic,
		Direct:             damage.Direct,
		MassiveDamage:      damage.MassiveDamage,
		Mitigated:          mitigated,
		Source:             damage.Source,
		SourceCharacterIDs: sourceCharacterIDs,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode damage payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.damage.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.damage_applied"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = domain
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Damage:      damage,
	})
	if err != nil {
		t.Fatalf("ApplyDamage returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character_id = %q, want char-1", resp.CharacterId)
	}
	if resp.State == nil {
		t.Fatal("expected state in response")
	}
	if resp.State.Hp >= 6 {
		t.Fatalf("hp = %d, expected less than 6 after 3 damage", resp.State.Hp)
	}
}

func TestApplyDamage_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	profile := dhStore.Profiles["camp-1:char-1"]
	state := dhStore.States["camp-1:char-1"]
	damage := &pb.DaggerheartDamageRequest{
		Amount:     3,
		DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
	}
	result, mitigated, err := applyDaggerheartDamage(damage, profile, state)
	if err != nil {
		t.Fatalf("apply daggerheart damage: %v", err)
	}

	hpBefore := result.HPBefore
	hpAfter := result.HPAfter
	armorBefore := result.ArmorBefore
	armorAfter := result.ArmorAfter
	sourceCharacterIDs := normalizeTargets(damage.GetSourceCharacterIds())
	payload := daggerheart.DamageAppliedPayload{
		CharacterID:        "char-1",
		HpBefore:           &hpBefore,
		HpAfter:            &hpAfter,
		ArmorBefore:        &armorBefore,
		ArmorAfter:         &armorAfter,
		ArmorSpent:         result.ArmorSpent,
		Severity:           daggerheartSeverityToString(result.Result.Severity),
		Marks:              result.Result.Marks,
		DamageType:         daggerheartDamageTypeToString(damage.DamageType),
		RollSeq:            nil,
		ResistPhysical:     damage.ResistPhysical,
		ResistMagic:        damage.ResistMagic,
		ImmunePhysical:     damage.ImmunePhysical,
		ImmuneMagic:        damage.ImmuneMagic,
		Direct:             damage.Direct,
		MassiveDamage:      damage.MassiveDamage,
		Mitigated:          mitigated,
		Source:             damage.Source,
		SourceCharacterIDs: sourceCharacterIDs,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode damage payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.damage.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.damage_applied"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-apply-damage",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-apply-damage")
	_, err = svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Damage:      damage,
	})
	if err != nil {
		t.Fatalf("ApplyDamage returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.damage.apply") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.damage.apply")
	}
	if domain.commands[0].SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", domain.commands[0].SystemID, daggerheart.SystemID)
	}
	if domain.commands[0].SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", domain.commands[0].SystemVersion, daggerheart.SystemVersion)
	}
	var got struct {
		CharacterID string `json:"character_id"`
		DamageType  string `json:"damage_type"`
		HpAfter     *int   `json:"hp_after"`
	}
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode damage command payload: %v", err)
	}
	if got.CharacterID != "char-1" {
		t.Fatalf("command character id = %s, want %s", got.CharacterID, "char-1")
	}
	if got.DamageType != "physical" {
		t.Fatalf("command damage type = %s, want %s", got.DamageType, "physical")
	}
	if got.HpAfter == nil || *got.HpAfter != hpAfter {
		t.Fatalf("command hp after = %v, want %d", got.HpAfter, hpAfter)
	}
}

func TestApplyDamage_WithArmorMitigation(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	profile := dhStore.Profiles["camp-1:char-1"]
	profile.MajorThreshold = 3
	profile.SevereThreshold = 6
	profile.ArmorMax = 1
	dhStore.Profiles["camp-1:char-1"] = profile

	state := dhStore.States["camp-1:char-1"]
	state.Hp = 6
	state.Armor = 1
	dhStore.States["camp-1:char-1"] = state

	damage := &pb.DaggerheartDamageRequest{
		Amount:     4,
		DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
	}
	result, mitigated, err := applyDaggerheartDamage(damage, profile, state)
	if err != nil {
		t.Fatalf("apply daggerheart damage: %v", err)
	}

	hpBefore := result.HPBefore
	hpAfter := result.HPAfter
	armorBefore := result.ArmorBefore
	armorAfter := result.ArmorAfter
	sourceCharacterIDs := normalizeTargets(damage.GetSourceCharacterIds())
	payload := daggerheart.DamageAppliedPayload{
		CharacterID:        "char-1",
		HpBefore:           &hpBefore,
		HpAfter:            &hpAfter,
		ArmorBefore:        &armorBefore,
		ArmorAfter:         &armorAfter,
		ArmorSpent:         result.ArmorSpent,
		Severity:           daggerheartSeverityToString(result.Result.Severity),
		Marks:              result.Result.Marks,
		DamageType:         daggerheartDamageTypeToString(damage.DamageType),
		RollSeq:            nil,
		ResistPhysical:     damage.ResistPhysical,
		ResistMagic:        damage.ResistMagic,
		ImmunePhysical:     damage.ImmunePhysical,
		ImmuneMagic:        damage.ImmuneMagic,
		Direct:             damage.Direct,
		MassiveDamage:      damage.MassiveDamage,
		Mitigated:          mitigated,
		Source:             damage.Source,
		SourceCharacterIDs: sourceCharacterIDs,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode damage payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.damage.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.damage_applied"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = domain

	_, err = svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Damage:      damage,
	})
	if err != nil {
		t.Fatalf("ApplyDamage returned error: %v", err)
	}

	events := eventStore.Events["camp-1"]
	if len(events) == 0 {
		t.Fatal("expected damage event")
	}
	last := events[len(events)-1]
	if last.Type != event.Type("sys.daggerheart.damage_applied") {
		t.Fatalf("last event type = %s, want %s", last.Type, event.Type("sys.daggerheart.damage_applied"))
	}
	var parsedPayload daggerheart.DamageAppliedPayload
	if err := json.Unmarshal(last.PayloadJSON, &parsedPayload); err != nil {
		t.Fatalf("decode damage payload: %v", err)
	}
	if parsedPayload.ArmorSpent != 1 {
		t.Fatalf("armor_spent = %d, want 1", parsedPayload.ArmorSpent)
	}
	if parsedPayload.Marks != 1 {
		t.Fatalf("marks = %d, want 1", parsedPayload.Marks)
	}
	if parsedPayload.Severity != "minor" {
		t.Fatalf("severity = %s, want minor", parsedPayload.Severity)
	}
}

func TestApplyDamage_RequireDamageRollWithoutSeq(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     3,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
		RequireDamageRoll: true,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyConditions_LifeStateOnly(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	before := daggerheart.LifeStateAlive
	after := daggerheart.LifeStateUnconscious
	patchPayload := daggerheart.CharacterStatePatchedPayload{
		CharacterID:     "char-1",
		LifeStateBefore: &before,
		LifeStateAfter:  &after,
	}
	patchJSON, err := json.Marshal(patchPayload)
	if err != nil {
		t.Fatalf("encode patch payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-conditions-life",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   patchJSON,
			}),
		},
	}}
	svc.stores.Domain = serviceDomain
	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-conditions-life")
	resp, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		LifeState:   pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS,
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if resp.State == nil {
		t.Fatal("expected state in response")
	}
	if resp.State.LifeState != pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS {
		t.Fatalf("life_state = %v, want UNCONSCIOUS", resp.State.LifeState)
	}

	events := eventStore.Events["camp-1"]
	if len(events) == 0 {
		t.Fatal("expected events")
	}
	last := events[len(events)-1]
	if last.Type != event.Type("sys.daggerheart.character_state_patched") {
		t.Fatalf("last event type = %s, want %s", last.Type, event.Type("sys.daggerheart.character_state_patched"))
	}
}

func TestApplyConditions_LifeStateNoChange(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		LifeState:   pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestApplyConditions_InvalidStoredLifeState(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	state := dhStore.States["camp-1:char-1"]
	state.LifeState = "not-a-life-state"
	dhStore.States["camp-1:char-1"] = state

	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		LifeState:   pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyConditions_NoConditionChanges(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	state := dhStore.States["camp-1:char-1"]
	state.Conditions = []string{"vulnerable"}
	dhStore.States["camp-1:char-1"] = state

	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestApplyConditions_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN},
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyConditions_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	conditionPayload := daggerheart.ConditionChangedPayload{
		CharacterID:      "char-1",
		ConditionsBefore: []string{},
		ConditionsAfter:  []string{daggerheart.ConditionHidden},
		Added:            []string{daggerheart.ConditionHidden},
	}
	conditionJSON, err := json.Marshal(conditionPayload)
	if err != nil {
		t.Fatalf("encode condition payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.condition.change"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.condition_changed"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-apply-conditions",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   conditionJSON,
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-apply-conditions")
	_, err = svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN},
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.condition.change") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.condition.change")
	}
	if domain.commands[0].SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", domain.commands[0].SystemID, daggerheart.SystemID)
	}
	if domain.commands[0].SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", domain.commands[0].SystemVersion, daggerheart.SystemVersion)
	}
	var got daggerheart.ConditionChangePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode condition command payload: %v", err)
	}
	if got.CharacterID != "char-1" {
		t.Fatalf("command character id = %s, want %s", got.CharacterID, "char-1")
	}
	if len(got.ConditionsAfter) != 1 || got.ConditionsAfter[0] != daggerheart.ConditionHidden {
		t.Fatalf("command conditions_after = %v, want %s", got.ConditionsAfter, daggerheart.ConditionHidden)
	}
	var foundConditionEvent bool
	for _, evt := range eventStore.Events["camp-1"] {
		if evt.Type == event.Type("sys.daggerheart.condition_changed") {
			foundConditionEvent = true
			break
		}
	}
	if !foundConditionEvent {
		t.Fatal("expected condition changed event")
	}
}

func TestApplyConditions_UsesDomainEngineForLifeState(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	before := daggerheart.LifeStateAlive
	after := daggerheart.LifeStateUnconscious
	patchPayload := daggerheart.CharacterStatePatchedPayload{
		CharacterID:     "char-1",
		LifeStateBefore: &before,
		LifeStateAfter:  &after,
	}
	patchJSON, err := json.Marshal(patchPayload)
	if err != nil {
		t.Fatalf("encode patch payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-apply-conditions",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   patchJSON,
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-apply-conditions")
	_, err = svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		LifeState:   pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS,
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.character_state.patch") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.character_state.patch")
	}
	if domain.commands[0].SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", domain.commands[0].SystemID, daggerheart.SystemID)
	}
	if domain.commands[0].SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", domain.commands[0].SystemVersion, daggerheart.SystemVersion)
	}
	var got daggerheart.CharacterStatePatchPayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode patch command payload: %v", err)
	}
	if got.CharacterID != "char-1" {
		t.Fatalf("command character id = %s, want %s", got.CharacterID, "char-1")
	}
	if got.LifeStateBefore == nil || *got.LifeStateBefore != before {
		t.Fatalf("command life_state_before = %v, want %s", got.LifeStateBefore, before)
	}
	if got.LifeStateAfter == nil || *got.LifeStateAfter != after {
		t.Fatalf("command life_state_after = %v, want %s", got.LifeStateAfter, after)
	}
	var foundStateEvent bool
	for _, evt := range eventStore.Events["camp-1"] {
		if evt.Type == event.Type("sys.daggerheart.character_state_patched") {
			foundStateEvent = true
			break
		}
	}
	if !foundStateEvent {
		t.Fatal("expected character state patched event")
	}
}

// --- ApplyRest tests ---

func TestApplyRest_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyRest(context.Background(), &pb.DaggerheartApplyRestRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyRest_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId:   "camp-1",
		CharacterIds: []string{"char-1"},
		Rest: &pb.DaggerheartRestRequest{
			RestType:  pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_SHORT,
			PartySize: 3,
		},
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyRest_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRest_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyRest(context.Background(), &pb.DaggerheartApplyRestRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRest_MissingRest(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRest_UnspecifiedRestType(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId: "camp-1",
		Rest: &pb.DaggerheartRestRequest{
			RestType: pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_UNSPECIFIED,
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRest_ShortRest_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	payloadJSON, err := json.Marshal(daggerheart.RestTakenPayload{
		RestType:         "short",
		Interrupted:      false,
		GMFearBefore:     0,
		GMFearAfter:      0,
		ShortRestsBefore: 0,
		ShortRestsAfter:  1,
		RefreshRest:      false,
		RefreshLongRest:  false,
	})
	if err != nil {
		t.Fatalf("encode rest payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.rest.take"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.rest_taken"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				EntityType:    "session",
				EntityID:      "camp-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = domain
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId:   "camp-1",
		CharacterIds: []string{"char-1"},
		Rest: &pb.DaggerheartRestRequest{
			RestType:  pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_SHORT,
			PartySize: 3,
		},
	})
	if err != nil {
		t.Fatalf("ApplyRest returned error: %v", err)
	}
	if resp.Snapshot == nil {
		t.Fatal("expected snapshot in response")
	}
}

func TestApplyRest_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	restPayload := struct {
		RestType    string `json:"rest_type"`
		Interrupted bool   `json:"interrupted"`
	}{
		RestType:    "short",
		Interrupted: false,
	}
	payloadJSON, err := json.Marshal(restPayload)
	if err != nil {
		t.Fatalf("encode rest payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.rest.take"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.rest_taken"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-rest-1",
				EntityType:    "session",
				EntityID:      "camp-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-rest-1")
	_, err = svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId:   "camp-1",
		CharacterIds: []string{"char-1"},
		Rest: &pb.DaggerheartRestRequest{
			RestType:  pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_SHORT,
			PartySize: 3,
		},
	})
	if err != nil {
		t.Fatalf("ApplyRest returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.rest.take") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.rest.take")
	}
	if domain.commands[0].SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", domain.commands[0].SystemID, daggerheart.SystemID)
	}
	if domain.commands[0].SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", domain.commands[0].SystemVersion, daggerheart.SystemVersion)
	}
	var got map[string]any
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode rest command payload: %v", err)
	}
	if got["rest_type"] != "short" {
		t.Fatalf("command rest_type = %v, want %s", got["rest_type"], "short")
	}
}

func TestApplyRest_LongRest_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	payloadJSON, err := json.Marshal(daggerheart.RestTakenPayload{
		RestType:         "long",
		Interrupted:      false,
		GMFearBefore:     0,
		GMFearAfter:      0,
		ShortRestsBefore: 1,
		ShortRestsAfter:  0,
		RefreshRest:      false,
		RefreshLongRest:  false,
	})
	if err != nil {
		t.Fatalf("encode rest payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.rest.take"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.rest_taken"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				EntityType:    "session",
				EntityID:      "camp-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = domain
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId:   "camp-1",
		CharacterIds: []string{"char-1"},
		Rest: &pb.DaggerheartRestRequest{
			RestType:  pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_LONG,
			PartySize: 3,
		},
	})
	if err != nil {
		t.Fatalf("ApplyRest returned error: %v", err)
	}
	if resp.Snapshot == nil {
		t.Fatal("expected snapshot in response")
	}
}

func TestApplyRest_LongRest_CountdownFailureDoesNotCommitRest(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	payloadJSON, err := json.Marshal(daggerheart.RestTakenPayload{
		RestType:         "long",
		Interrupted:      false,
		GMFearBefore:     0,
		GMFearAfter:      0,
		ShortRestsBefore: 1,
		ShortRestsAfter:  0,
		RefreshRest:      false,
		RefreshLongRest:  false,
	})
	if err != nil {
		t.Fatalf("encode rest payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.rest.take"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.rest_taken"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				EntityType:    "session",
				EntityID:      "camp-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := contextWithSessionID("sess-1")
	_, err = svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId:   "camp-1",
		CharacterIds: []string{"char-1"},
		Rest: &pb.DaggerheartRestRequest{
			RestType:            pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_LONG,
			PartySize:           3,
			LongTermCountdownId: "missing-countdown",
		},
	})
	assertStatusCode(t, err, codes.Internal)

	if len(eventStore.Events["camp-1"]) != 0 {
		t.Fatalf("expected no events committed on failed rest flow, got %d", len(eventStore.Events["camp-1"]))
	}
}

func TestApplyRest_LongRest_WithCountdown_UsesSingleDomainCommand(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.Countdowns["camp-1:cd-1"] = storage.DaggerheartCountdown{
		CampaignID:  "camp-1",
		CountdownID: "cd-1",
		Name:        "Long Term",
		Kind:        "progress",
		Current:     2,
		Max:         6,
		Direction:   "increase",
		Looping:     false,
	}
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	payloadJSON, err := json.Marshal(daggerheart.RestTakenPayload{
		RestType:         "long",
		Interrupted:      false,
		GMFearBefore:     0,
		GMFearAfter:      0,
		ShortRestsBefore: 1,
		ShortRestsAfter:  0,
		RefreshRest:      false,
		RefreshLongRest:  false,
	})
	if err != nil {
		t.Fatalf("encode rest payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.rest.take"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.rest_taken"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				EntityType:    "session",
				EntityID:      "camp-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := contextWithSessionID("sess-1")
	_, err = svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId:   "camp-1",
		CharacterIds: []string{"char-1"},
		Rest: &pb.DaggerheartRestRequest{
			RestType:            pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_LONG,
			PartySize:           3,
			LongTermCountdownId: "cd-1",
		},
	})
	if err != nil {
		t.Fatalf("ApplyRest returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected one domain command, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.rest.take") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.rest.take")
	}
	var got daggerheart.RestTakePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode rest command payload: %v", err)
	}
	if got.LongTermCountdown == nil {
		t.Fatal("expected long_term_countdown payload")
	}
	if got.LongTermCountdown.CountdownID != "cd-1" {
		t.Fatalf("countdown id = %s, want %s", got.LongTermCountdown.CountdownID, "cd-1")
	}
}

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

// --- ApplyAdversaryDamage tests ---

func newAdversaryDamageTestService() *DaggerheartService {
	svc := newAdversaryTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartAdversaryStore)
	dhStore.adversaries["camp-1:adv-1"] = storage.DaggerheartAdversary{
		AdversaryID: "adv-1",
		CampaignID:  "camp-1",
		SessionID:   "sess-1",
		Name:        "Goblin",
		HP:          8,
		HPMax:       8,
		Armor:       1,
		Major:       4,
		Severe:      7,
	}
	return svc
}

func TestApplyAdversaryDamage_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyAdversaryDamage(context.Background(), &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId: "c1", AdversaryId: "a1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyAdversaryDamage_RequiresDomainEngine(t *testing.T) {
	svc := newAdversaryDamageTestService()
	svc.stores.Domain = nil
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     5,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyAdversaryDamage_MissingCampaignId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryDamage_MissingAdversaryId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryDamage_MissingSessionId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.ApplyAdversaryDamage(context.Background(), &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId: "camp-1", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryDamage_MissingDamage(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId: "camp-1", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryDamage_NegativeAmount(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     -1,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryDamage_UnspecifiedType(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     2,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED,
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryDamage_Success(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartAdversaryStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	adversary := dhStore.adversaries["camp-1:adv-1"]
	damage := &pb.DaggerheartDamageRequest{
		Amount:     5,
		DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
	}
	result, mitigated, err := applyDaggerheartAdversaryDamage(damage, adversary)
	if err != nil {
		t.Fatalf("apply adversary damage: %v", err)
	}
	hpBefore := result.HPBefore
	hpAfter := result.HPAfter
	armorBefore := result.ArmorBefore
	armorAfter := result.ArmorAfter
	sourceCharacterIDs := normalizeTargets(damage.GetSourceCharacterIds())
	payload := daggerheart.AdversaryDamageAppliedPayload{
		AdversaryID:        "adv-1",
		HpBefore:           &hpBefore,
		HpAfter:            &hpAfter,
		ArmorBefore:        &armorBefore,
		ArmorAfter:         &armorAfter,
		ArmorSpent:         result.ArmorSpent,
		Severity:           daggerheartSeverityToString(result.Result.Severity),
		Marks:              result.Result.Marks,
		DamageType:         daggerheartDamageTypeToString(damage.DamageType),
		RollSeq:            nil,
		ResistPhysical:     damage.ResistPhysical,
		ResistMagic:        damage.ResistMagic,
		ImmunePhysical:     damage.ImmunePhysical,
		ImmuneMagic:        damage.ImmuneMagic,
		Direct:             damage.Direct,
		MassiveDamage:      damage.MassiveDamage,
		Mitigated:          mitigated,
		Source:             damage.Source,
		SourceCharacterIDs: sourceCharacterIDs,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode adversary damage payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.adversary_damage.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.adversary_damage_applied"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				EntityType:    "adversary",
				EntityID:      "adv-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = domain
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     5, // major damage (>=4, <7)
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryDamage returned error: %v", err)
	}
	if resp.AdversaryId != "adv-1" {
		t.Fatalf("adversary_id = %q, want adv-1", resp.AdversaryId)
	}
	if resp.Adversary == nil {
		t.Fatal("expected adversary in response")
	}
}

func TestApplyAdversaryDamage_UsesDomainEngine(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartAdversaryStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	adversary := dhStore.adversaries["camp-1:adv-1"]
	damage := &pb.DaggerheartDamageRequest{
		Amount:     5,
		DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
	}
	result, mitigated, err := applyDaggerheartAdversaryDamage(damage, adversary)
	if err != nil {
		t.Fatalf("apply adversary damage: %v", err)
	}
	hpBefore := result.HPBefore
	hpAfter := result.HPAfter
	armorBefore := result.ArmorBefore
	armorAfter := result.ArmorAfter
	sourceCharacterIDs := normalizeTargets(damage.GetSourceCharacterIds())
	payload := daggerheart.AdversaryDamageAppliedPayload{
		AdversaryID:        "adv-1",
		HpBefore:           &hpBefore,
		HpAfter:            &hpAfter,
		ArmorBefore:        &armorBefore,
		ArmorAfter:         &armorAfter,
		ArmorSpent:         result.ArmorSpent,
		Severity:           daggerheartSeverityToString(result.Result.Severity),
		Marks:              result.Result.Marks,
		DamageType:         daggerheartDamageTypeToString(damage.DamageType),
		RollSeq:            nil,
		ResistPhysical:     damage.ResistPhysical,
		ResistMagic:        damage.ResistMagic,
		ImmunePhysical:     damage.ImmunePhysical,
		ImmuneMagic:        damage.ImmuneMagic,
		Direct:             damage.Direct,
		MassiveDamage:      damage.MassiveDamage,
		Mitigated:          mitigated,
		Source:             damage.Source,
		SourceCharacterIDs: sourceCharacterIDs,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode adversary damage payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.adversary_damage.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.adversary_damage_applied"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-adversary-damage",
				EntityType:    "adversary",
				EntityID:      "adv-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-adversary-damage")
	_, err = svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Damage:      damage,
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryDamage returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.adversary_damage.apply") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.adversary_damage.apply")
	}
	if domain.commands[0].SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", domain.commands[0].SystemID, daggerheart.SystemID)
	}
	if domain.commands[0].SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", domain.commands[0].SystemVersion, daggerheart.SystemVersion)
	}
	var got struct {
		AdversaryID string `json:"adversary_id"`
		DamageType  string `json:"damage_type"`
		HpAfter     *int   `json:"hp_after"`
	}
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode adversary damage command payload: %v", err)
	}
	if got.AdversaryID != "adv-1" {
		t.Fatalf("command adversary id = %s, want %s", got.AdversaryID, "adv-1")
	}
	if got.DamageType != "physical" {
		t.Fatalf("command damage type = %s, want %s", got.DamageType, "physical")
	}
	if got.HpAfter == nil || *got.HpAfter != hpAfter {
		t.Fatalf("command hp after = %v, want %d", got.HpAfter, hpAfter)
	}
}

func TestApplyAdversaryDamage_DirectDamage(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartAdversaryStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	adversary := dhStore.adversaries["camp-1:adv-1"]
	damage := &pb.DaggerheartDamageRequest{
		Amount:     3,
		DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		Direct:     true,
	}
	result, mitigated, err := applyDaggerheartAdversaryDamage(damage, adversary)
	if err != nil {
		t.Fatalf("apply adversary damage: %v", err)
	}
	hpBefore := result.HPBefore
	hpAfter := result.HPAfter
	armorBefore := result.ArmorBefore
	armorAfter := result.ArmorAfter
	sourceCharacterIDs := normalizeTargets(damage.GetSourceCharacterIds())
	payload := daggerheart.AdversaryDamageAppliedPayload{
		AdversaryID:        "adv-1",
		HpBefore:           &hpBefore,
		HpAfter:            &hpAfter,
		ArmorBefore:        &armorBefore,
		ArmorAfter:         &armorAfter,
		ArmorSpent:         result.ArmorSpent,
		Severity:           daggerheartSeverityToString(result.Result.Severity),
		Marks:              result.Result.Marks,
		DamageType:         daggerheartDamageTypeToString(damage.DamageType),
		RollSeq:            nil,
		ResistPhysical:     damage.ResistPhysical,
		ResistMagic:        damage.ResistMagic,
		ImmunePhysical:     damage.ImmunePhysical,
		ImmuneMagic:        damage.ImmuneMagic,
		Direct:             damage.Direct,
		MassiveDamage:      damage.MassiveDamage,
		Mitigated:          mitigated,
		Source:             damage.Source,
		SourceCharacterIDs: sourceCharacterIDs,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode adversary damage payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.adversary_damage.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.adversary_damage_applied"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				EntityType:    "adversary",
				EntityID:      "adv-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = domain
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     3,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
			Direct:     true,
		},
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryDamage returned error: %v", err)
	}
	if resp.Adversary == nil {
		t.Fatal("expected adversary in response")
	}
}

// --- ApplyAdversaryConditions tests ---

func TestApplyAdversaryConditions_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyAdversaryConditions(context.Background(), &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId: "c1", AdversaryId: "a1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyAdversaryConditions_MissingCampaignId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		AdversaryId: "adv-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryConditions_MissingAdversaryId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId: "camp-1",
		Add:        []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryConditions_MissingSessionId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.ApplyAdversaryConditions(context.Background(), &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryConditions_RequiresDomainEngine(t *testing.T) {
	svc := newAdversaryDamageTestService()
	svc.stores.Domain = nil
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyAdversaryConditions_NoConditions(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryConditions_ConflictAddRemoveSame(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
		Remove:      []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryConditions_AddCondition_Success(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	payload := daggerheart.AdversaryConditionChangedPayload{
		AdversaryID:      "adv-1",
		ConditionsBefore: []string{},
		ConditionsAfter:  []string{daggerheart.ConditionVulnerable},
		Added:            []string{daggerheart.ConditionVulnerable},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode adversary condition payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.adversary_condition.change"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.adversary_condition_changed"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-adv-conditions-add",
				EntityType:    "adversary",
				EntityID:      "adv-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = serviceDomain
	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-adv-conditions-add")
	resp, err := svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryConditions returned error: %v", err)
	}
	if resp.AdversaryId != "adv-1" {
		t.Fatalf("adversary_id = %q, want adv-1", resp.AdversaryId)
	}
	if len(resp.Added) == 0 {
		t.Fatal("expected added conditions")
	}
}

func TestApplyAdversaryConditions_RemoveCondition_Success(t *testing.T) {
	svc := newAdversaryDamageTestService()
	// Pre-populate a condition on the adversary.
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartAdversaryStore)
	adv := dhStore.adversaries["camp-1:adv-1"]
	adv.Conditions = []string{"vulnerable"}
	dhStore.adversaries["camp-1:adv-1"] = adv
	eventStore := svc.stores.Event.(*fakeEventStore)
	payload := daggerheart.AdversaryConditionChangedPayload{
		AdversaryID:      "adv-1",
		ConditionsBefore: []string{daggerheart.ConditionVulnerable},
		ConditionsAfter:  []string{},
		Removed:          []string{daggerheart.ConditionVulnerable},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode adversary condition payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.adversary_condition.change"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.adversary_condition_changed"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-adv-conditions-remove",
				EntityType:    "adversary",
				EntityID:      "adv-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = serviceDomain
	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-adv-conditions-remove")
	resp, err := svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Remove:      []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryConditions returned error: %v", err)
	}
	if len(resp.Removed) == 0 {
		t.Fatal("expected removed conditions")
	}
}

func TestApplyAdversaryConditions_UsesDomainEngine(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	payload := daggerheart.AdversaryConditionChangedPayload{
		AdversaryID:      "adv-1",
		ConditionsBefore: []string{},
		ConditionsAfter:  []string{daggerheart.ConditionVulnerable},
		Added:            []string{daggerheart.ConditionVulnerable},
		Source:           "test",
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode adversary condition payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.adversary_condition.change"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.adversary_condition_changed"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-adversary-conditions",
				EntityType:    "adversary",
				EntityID:      "adv-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-adversary-conditions")
	_, err = svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
		Source:      "test",
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryConditions returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.adversary_condition.change") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.adversary_condition.change")
	}
	if domain.commands[0].SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", domain.commands[0].SystemID, daggerheart.SystemID)
	}
	if domain.commands[0].SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", domain.commands[0].SystemVersion, daggerheart.SystemVersion)
	}
	var got struct {
		AdversaryID     string   `json:"adversary_id"`
		ConditionsAfter []string `json:"conditions_after"`
	}
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode adversary condition command payload: %v", err)
	}
	if got.AdversaryID != "adv-1" {
		t.Fatalf("command adversary id = %s, want %s", got.AdversaryID, "adv-1")
	}
	if len(got.ConditionsAfter) != 1 || got.ConditionsAfter[0] != daggerheart.ConditionVulnerable {
		t.Fatalf("command conditions_after = %v, want [%s]", got.ConditionsAfter, daggerheart.ConditionVulnerable)
	}
}

func TestApplyAdversaryConditions_NoChanges(t *testing.T) {
	svc := newAdversaryDamageTestService()
	// Pre-populate a condition that we try to re-add.
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartAdversaryStore)
	adv := dhStore.adversaries["camp-1:adv-1"]
	adv.Conditions = []string{"vulnerable"}
	dhStore.adversaries["camp-1:adv-1"] = adv

	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

// --- ApplyConditions gap fills ---

func TestApplyConditions_AddCondition_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	conditionPayload := daggerheart.ConditionChangedPayload{
		CharacterID:      "char-1",
		ConditionsBefore: []string{},
		ConditionsAfter:  []string{daggerheart.ConditionHidden},
		Added:            []string{daggerheart.ConditionHidden},
	}
	conditionJSON, err := json.Marshal(conditionPayload)
	if err != nil {
		t.Fatalf("encode condition payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.condition.change"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.condition_changed"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-conditions-add",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   conditionJSON,
			}),
		},
	}}
	svc.stores.Domain = serviceDomain
	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-conditions-add")
	resp, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN},
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if len(resp.Added) == 0 {
		t.Fatal("expected added conditions")
	}
}

func TestApplyConditions_RemoveCondition_Success(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	state := dhStore.States["camp-1:char-1"]
	state.Conditions = []string{"hidden", "vulnerable"}
	dhStore.States["camp-1:char-1"] = state
	eventStore := svc.stores.Event.(*fakeEventStore)
	conditionPayload := daggerheart.ConditionChangedPayload{
		CharacterID:      "char-1",
		ConditionsBefore: []string{daggerheart.ConditionHidden, daggerheart.ConditionVulnerable},
		ConditionsAfter:  []string{daggerheart.ConditionVulnerable},
		Removed:          []string{daggerheart.ConditionHidden},
	}
	conditionJSON, err := json.Marshal(conditionPayload)
	if err != nil {
		t.Fatalf("encode condition payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.condition.change"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.condition_changed"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-conditions-remove",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   conditionJSON,
			}),
		},
	}}
	svc.stores.Domain = serviceDomain
	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-conditions-remove")
	resp, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Remove:      []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN},
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if len(resp.Removed) == 0 {
		t.Fatal("expected removed conditions")
	}
}

func TestApplyConditions_AddAndRemove(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	state := dhStore.States["camp-1:char-1"]
	state.Conditions = []string{"hidden"}
	dhStore.States["camp-1:char-1"] = state
	eventStore := svc.stores.Event.(*fakeEventStore)
	conditionPayload := daggerheart.ConditionChangedPayload{
		CharacterID:      "char-1",
		ConditionsBefore: []string{daggerheart.ConditionHidden},
		ConditionsAfter:  []string{daggerheart.ConditionVulnerable},
		Added:            []string{daggerheart.ConditionVulnerable},
		Removed:          []string{daggerheart.ConditionHidden},
	}
	conditionJSON, err := json.Marshal(conditionPayload)
	if err != nil {
		t.Fatalf("encode condition payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.condition.change"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.condition_changed"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-conditions-both",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   conditionJSON,
			}),
		},
	}}
	svc.stores.Domain = serviceDomain
	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-conditions-both")
	resp, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
		Remove:      []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN},
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if len(resp.Added) == 0 {
		t.Fatal("expected added conditions")
	}
	if len(resp.Removed) == 0 {
		t.Fatal("expected removed conditions")
	}
}

func TestApplyConditions_ConflictAddRemoveSame(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN},
		Remove:      []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

// --- ApplyGmMove gap fills ---

func TestApplyGmMove_FearSpentExceedsAvailable(t *testing.T) {
	svc := newActionTestService()
	// Snapshot has 0 fear.
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Move: "test_move", FearSpent: 10,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyGmMove_CampaignNotFound(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "nonexistent", SessionId: "sess-1", Move: "test",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyGmMove_SessionNotFound(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1", SessionId: "nonexistent", Move: "test",
	})
	assertStatusCode(t, err, codes.Internal)
}

// --- SessionActionRoll tests ---
