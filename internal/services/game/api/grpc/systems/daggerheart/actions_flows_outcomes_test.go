package daggerheart

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestSessionActionRoll_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionActionRoll(context.Background(), &pb.SessionActionRollRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionActionRoll_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionActionRoll(context.Background(), &pb.SessionActionRollRequest{
		SessionId: "sess-1", CharacterId: "char-1", Trait: "agility",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionActionRoll_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionActionRoll(context.Background(), &pb.SessionActionRollRequest{
		CampaignId: "camp-1", CharacterId: "char-1", Trait: "agility",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionActionRoll_MissingCharacterId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionActionRoll(context.Background(), &pb.SessionActionRollRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Trait: "agility",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionActionRoll_MissingTrait(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionActionRoll(context.Background(), &pb.SessionActionRollRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionActionRoll_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionActionRoll(context.Background(), &pb.SessionActionRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CharacterId: "char-1",
		Trait:       "agility",
		Difficulty:  10,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionActionRoll_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.roll_resolved"),
				Timestamp:   time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-roll-success",
				EntityType:  "roll",
				EntityID:    "req-roll-success",
				PayloadJSON: []byte(`{"request_id":"req-roll-success"}`),
			}),
		},
	}}
	svc.stores.Domain = serviceDomain
	ctx := grpcmeta.WithRequestID(context.Background(), "req-roll-success")
	resp, err := svc.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CharacterId: "char-1",
		Trait:       "agility",
		Difficulty:  10,
	})
	if err != nil {
		t.Fatalf("SessionActionRoll returned error: %v", err)
	}
	if resp.RollSeq == 0 {
		t.Fatal("expected non-zero roll seq")
	}
	if resp.Rng == nil {
		t.Fatal("expected rng in response")
	}
}

func TestSessionActionRoll_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.roll_resolved"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-roll-1",
				EntityType:  "roll",
				EntityID:    "req-roll-1",
				PayloadJSON: []byte(`{"request_id":"req-roll-1"}`),
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(context.Background(), "req-roll-1")
	resp, err := svc.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CharacterId: "char-1",
		Trait:       "agility",
		Difficulty:  10,
	})
	if err != nil {
		t.Fatalf("SessionActionRoll returned error: %v", err)
	}
	if resp.RollSeq == 0 {
		t.Fatal("expected non-zero roll seq")
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("action.roll.resolve") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "action.roll.resolve")
	}
	if got := len(eventStore.Events["camp-1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.Events["camp-1"][0].Type != event.Type("action.roll_resolved") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["camp-1"][0].Type, event.Type("action.roll_resolved"))
	}
}

func TestSessionActionRoll_UsesDomainEngineForHopeSpend(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	hopeBefore := 2
	hopeAfter := 1
	patchPayload := daggerheart.CharacterStatePatchedPayload{
		CharacterID: "char-1",
		HopeBefore:  &hopeBefore,
		HopeAfter:   &hopeAfter,
	}
	patchJSON, err := json.Marshal(patchPayload)
	if err != nil {
		t.Fatalf("encode patch payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.hope.spend"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-roll-1",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   patchJSON,
			}),
		},
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-roll-1",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   patchJSON,
			}),
		},
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.roll_resolved"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-roll-1",
				EntityType:  "roll",
				EntityID:    "req-roll-1",
				PayloadJSON: []byte(`{"request_id":"req-roll-1"}`),
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(context.Background(), "req-roll-1")
	_, err = svc.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CharacterId: "char-1",
		Trait:       "agility",
		Difficulty:  10,
		Modifiers: []*pb.ActionRollModifier{
			{Value: 1, Source: "experience"},
		},
	})
	if err != nil {
		t.Fatalf("SessionActionRoll returned error: %v", err)
	}
	if domain.calls != 2 {
		t.Fatalf("expected domain to be called two times, got %d", domain.calls)
	}
	if len(domain.commands) != 2 {
		t.Fatalf("expected 2 domain commands, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.hope.spend") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.hope.spend")
	}
	if domain.commands[1].Type != command.Type("action.roll.resolve") {
		t.Fatalf("command type = %s, want %s", domain.commands[1].Type, "action.roll.resolve")
	}
	if domain.commands[0].SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", domain.commands[0].SystemID, daggerheart.SystemID)
	}
	if domain.commands[0].SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", domain.commands[0].SystemVersion, daggerheart.SystemVersion)
	}
	var spend daggerheart.HopeSpendPayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &spend); err != nil {
		t.Fatalf("decode hope spend command payload: %v", err)
	}
	if spend.CharacterID != "char-1" {
		t.Fatalf("hope spend character id = %s, want %s", spend.CharacterID, "char-1")
	}
	if spend.Amount != 1 {
		t.Fatalf("hope spend amount = %d, want %d", spend.Amount, 1)
	}
	if spend.Before != hopeBefore {
		t.Fatalf("hope spend before = %d, want %d", spend.Before, hopeBefore)
	}
	if spend.After != hopeAfter {
		t.Fatalf("hope spend after = %d, want %d", spend.After, hopeAfter)
	}
	if spend.Source != "experience" {
		t.Fatalf("hope spend source = %s, want %s", spend.Source, "experience")
	}
	var foundPatchEvent bool
	for _, evt := range eventStore.Events["camp-1"] {
		if evt.Type == event.Type("sys.daggerheart.character_state_patched") {
			foundPatchEvent = true
			break
		}
	}
	if !foundPatchEvent {
		t.Fatal("expected character state patched event")
	}
	updated, err := svc.stores.Daggerheart.GetDaggerheartCharacterState(ctx, "camp-1", "char-1")
	if err != nil {
		t.Fatalf("load updated state: %v", err)
	}
	if updated.Hope != hopeAfter {
		t.Fatalf("state hope = %d, want %d", updated.Hope, hopeAfter)
	}
}

func TestSessionActionRoll_WithModifiers(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	hopeBefore := 2
	hopeAfter := 1
	patchPayload := daggerheart.CharacterStatePatchedPayload{
		CharacterID: "char-1",
		HopeBefore:  &hopeBefore,
		HopeAfter:   &hopeAfter,
	}
	patchJSON, err := json.Marshal(patchPayload)
	if err != nil {
		t.Fatalf("encode patch payload: %v", err)
	}
	rollPayloadJSON, err := json.Marshal(map[string]string{"request_id": "req-roll-modifiers"})
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.hope.spend"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-roll-modifiers",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   patchJSON,
			}),
		},
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-roll-modifiers",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   patchJSON,
			}),
		},
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.roll_resolved"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-roll-modifiers",
				EntityType:  "roll",
				EntityID:    "req-roll-modifiers",
				PayloadJSON: rollPayloadJSON,
			}),
		},
	}}
	svc.stores.Domain = serviceDomain
	ctx := grpcmeta.WithRequestID(context.Background(), "req-roll-modifiers")
	resp, err := svc.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CharacterId: "char-1",
		Trait:       "agility",
		Difficulty:  10,
		Modifiers: []*pb.ActionRollModifier{
			{Value: 2, Source: "experience"},
		},
	})
	if err != nil {
		t.Fatalf("SessionActionRoll returned error: %v", err)
	}
	if resp.RollSeq == 0 {
		t.Fatal("expected non-zero roll seq")
	}
	var foundPatchEvent bool
	for _, evt := range eventStore.Events["camp-1"] {
		if evt.Type == event.Type("sys.daggerheart.character_state_patched") {
			foundPatchEvent = true
			break
		}
	}
	if !foundPatchEvent {
		t.Fatal("expected character state patched event")
	}
	updated, err := svc.stores.Daggerheart.GetDaggerheartCharacterState(ctx, "camp-1", "char-1")
	if err != nil {
		t.Fatalf("load updated state: %v", err)
	}
	if updated.Hope != hopeAfter {
		t.Fatalf("state hope = %d, want %d", updated.Hope, hopeAfter)
	}
}

// --- SessionDamageRoll tests ---

func TestSessionDamageRoll_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionDamageRoll(context.Background(), &pb.SessionDamageRollRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionDamageRoll_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionDamageRoll(context.Background(), &pb.SessionDamageRollRequest{
		SessionId: "sess-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionDamageRoll_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionDamageRoll(context.Background(), &pb.SessionDamageRollRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionDamageRoll_MissingCharacterId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionDamageRoll(context.Background(), &pb.SessionDamageRollRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionDamageRoll_MissingDice(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionDamageRoll(context.Background(), &pb.SessionDamageRollRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionDamageRoll_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionDamageRoll(context.Background(), &pb.SessionDamageRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CharacterId: "char-1",
		Dice:        []*pb.DiceSpec{{Sides: 6, Count: 2}},
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionDamageRoll_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	rollPayload := action.RollResolvePayload{
		RequestID: "req-damage-roll-success",
		RollSeq:   1,
		Results: map[string]any{
			"rolls":          []int{3, 4},
			"base_total":     7,
			"modifier":       0,
			"critical_bonus": 0,
			"total":          7,
		},
		SystemData: map[string]any{
			"character_id":   "char-1",
			"roll_kind":      "damage_roll",
			"roll":           7,
			"base_total":     7,
			"modifier":       0,
			"critical":       false,
			"critical_bonus": 0,
			"total":          7,
		},
	}
	rollPayloadJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode damage roll payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.roll_resolved"),
				Timestamp:   time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-damage-roll-success",
				EntityType:  "roll",
				EntityID:    "req-damage-roll-success",
				PayloadJSON: rollPayloadJSON,
			}),
		},
	}}
	svc.stores.Domain = serviceDomain
	ctx := grpcmeta.WithRequestID(context.Background(), "req-damage-roll-success")
	resp, err := svc.SessionDamageRoll(ctx, &pb.SessionDamageRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CharacterId: "char-1",
		Dice:        []*pb.DiceSpec{{Sides: 6, Count: 2}},
	})
	if err != nil {
		t.Fatalf("SessionDamageRoll returned error: %v", err)
	}
	if resp.RollSeq == 0 {
		t.Fatal("expected non-zero roll seq")
	}
	if resp.Total == 0 {
		t.Fatal("expected non-zero total")
	}
}

func TestSessionDamageRoll_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)

	payload := action.RollResolvePayload{
		RequestID: "req-damage-roll-legacy",
		RollSeq:   1,
		Results: map[string]any{
			"rolls":          []int{3, 4},
			"base_total":     7,
			"modifier":       0,
			"critical_bonus": 0,
			"total":          7,
		},
		SystemData: map[string]any{
			"character_id":   "char-1",
			"roll_kind":      "damage_roll",
			"roll":           7,
			"base_total":     7,
			"modifier":       0,
			"critical":       false,
			"critical_bonus": 0,
			"total":          7,
		},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode damage roll payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.roll_resolved"),
				Timestamp:   time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-damage-roll-legacy",
				EntityType:  "roll",
				EntityID:    "req-damage-roll-legacy",
				PayloadJSON: payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(context.Background(), "req-damage-roll-legacy")
	_, err = svc.SessionDamageRoll(ctx, &pb.SessionDamageRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CharacterId: "char-1",
		Dice:        []*pb.DiceSpec{{Sides: 6, Count: 2}},
	})
	if err != nil {
		t.Fatalf("SessionDamageRoll returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("action.roll.resolve") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "action.roll.resolve")
	}
	var got struct {
		SystemData map[string]any `json:"system_data"`
	}
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode damage roll command payload: %v", err)
	}
	characterID, ok := got.SystemData["character_id"].(string)
	if !ok || characterID != "char-1" {
		t.Fatalf("command character id = %v, want %s", got.SystemData["character_id"], "char-1")
	}
	if gotRollSeq, ok := got.SystemData["roll_seq"]; ok {
		if gotRollSeq != nil {
			if _, ok := gotRollSeq.(float64); !ok {
				t.Fatalf("command roll seq = %v, expected number", gotRollSeq)
			}
		}
	}
	var gotPayload struct {
		RollSeq uint64 `json:"roll_seq"`
	}
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &gotPayload); err != nil {
		t.Fatalf("decode damage roll command payload: %v", err)
	}
	if gotPayload.RollSeq == 0 {
		t.Fatal("expected non-zero roll seq in command payload")
	}
}

// --- SessionAttackFlow tests ---

func TestSessionAttackFlow_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionAttackFlow_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAttackFlow_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAttackFlow_MissingCharacterId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAttackFlow_MissingTrait(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAttackFlow_MissingTargetId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CharacterId: "char-1", Trait: "agility",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAttackFlow_MissingDamage(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CharacterId: "char-1", Trait: "agility", TargetId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAttackFlow_MissingDamageType(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CharacterId: "char-1", Trait: "agility", TargetId: "adv-1",
		Damage: &pb.DaggerheartAttackDamageSpec{},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

// --- SessionReactionFlow tests ---

func TestSessionReactionFlow_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionReactionFlow(context.Background(), &pb.SessionReactionFlowRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionReactionFlow_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionReactionFlow(context.Background(), &pb.SessionReactionFlowRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionReactionFlow_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionReactionFlow(context.Background(), &pb.SessionReactionFlowRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionReactionFlow_MissingCharacterId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionReactionFlow(context.Background(), &pb.SessionReactionFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionReactionFlow_MissingTrait(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionReactionFlow(context.Background(), &pb.SessionReactionFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionReactionFlow_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-reaction-1",
		RollSeq:   1,
		Results:   map[string]any{"d20": 12},
		Outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_REACTION.String(),
			"hope_fear":    false,
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}

	outcomeJSON, err := json.Marshal(action.OutcomeApplyPayload{
		RequestID: "req-reaction-1",
		RollSeq:   1,
		Targets:   []string{"char-1"},
	})
	if err != nil {
		t.Fatalf("encode outcome payload: %v", err)
	}

	svc.stores.Domain = &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.roll_resolved"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-reaction-1",
				EntityType:  "roll",
				EntityID:    "req-reaction-1",
				PayloadJSON: rollJSON,
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_applied"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-reaction-1",
				EntityType:  "outcome",
				EntityID:    "req-reaction-1",
				PayloadJSON: outcomeJSON,
			}),
		},
	}}
	ctx := grpcmeta.WithRequestID(context.Background(), "req-reaction-1")
	resp, err := svc.SessionReactionFlow(ctx, &pb.SessionReactionFlowRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CharacterId: "char-1",
		Trait:       "agility",
		Difficulty:  10,
	})
	if err != nil {
		t.Fatalf("SessionReactionFlow returned error: %v", err)
	}
	if resp.ActionRoll == nil {
		t.Fatal("expected action roll in response")
	}
	if resp.RollOutcome == nil {
		t.Fatal("expected roll outcome in response")
	}
	if resp.ReactionOutcome == nil {
		t.Fatal("expected reaction outcome in response")
	}
}

func TestSessionReactionFlow_ForwardsAdvantageDisadvantage(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-reaction-forward-adv",
		RollSeq:   1,
		Results: map[string]any{
			"d20": 16,
		},
		Outcome: pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_REACTION.String(),
			"hope_fear":    false,
			"advantage":    0,
			"disadvantage": 0,
			"outcome":      pb.Outcome_SUCCESS_WITH_HOPE.String(),
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}

	outcomeJSON, err := json.Marshal(action.OutcomeApplyPayload{
		RequestID: "req-reaction-forward-adv",
		RollSeq:   1,
		Targets:   []string{"char-1"},
	})
	if err != nil {
		t.Fatalf("encode outcome payload: %v", err)
	}

	svc.stores.Domain = &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.roll_resolved"),
				Timestamp:   time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-reaction-forward-adv",
				EntityType:  "roll",
				EntityID:    "req-reaction-forward-adv",
				PayloadJSON: rollJSON,
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_applied"),
				Timestamp:   time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-reaction-forward-adv",
				EntityType:  "outcome",
				EntityID:    "req-reaction-forward-adv",
				PayloadJSON: outcomeJSON,
			}),
		},
	}}

	ctx := grpcmeta.WithRequestID(context.Background(), "req-reaction-forward-adv")
	reactionSeed := uint64(11)
	_, err = svc.SessionReactionFlow(ctx, &pb.SessionReactionFlowRequest{
		CampaignId:   "camp-1",
		SessionId:    "sess-1",
		CharacterId:  "char-1",
		Trait:        "agility",
		Difficulty:   10,
		Advantage:    2,
		Disadvantage: 1,
		ReactionRng: &commonv1.RngRequest{
			Seed:     &reactionSeed,
			RollMode: commonv1.RollMode_REPLAY,
		},
	})
	if err != nil {
		t.Fatalf("SessionReactionFlow returned error: %v", err)
	}

	if len(svc.stores.Domain.(*fakeDomainEngine).commands) == 0 {
		t.Fatal("expected domain commands")
	}

	var commandPayload action.RollResolvePayload
	rollCommandPayload := svc.stores.Domain.(*fakeDomainEngine).commands[0].PayloadJSON
	if err := json.Unmarshal(rollCommandPayload, &commandPayload); err != nil {
		t.Fatalf("decode action roll command payload: %v", err)
	}

	advRaw, ok := commandPayload.SystemData["advantage"]
	if !ok {
		t.Fatal("expected advantage in system_data")
	}
	disRaw, ok := commandPayload.SystemData["disadvantage"]
	if !ok {
		t.Fatal("expected disadvantage in system_data")
	}
	advantage, ok := advRaw.(float64)
	if !ok || int(advantage) != 2 {
		t.Fatalf("advantage in command payload = %v, want 2", advRaw)
	}
	disadvantage, ok := disRaw.(float64)
	if !ok || int(disadvantage) != 1 {
		t.Fatalf("disadvantage in command payload = %v, want 1", disRaw)
	}
}

// --- SessionAdversaryAttackRoll tests ---

func TestSessionAdversaryAttackRoll_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionAdversaryAttackRoll(context.Background(), &pb.SessionAdversaryAttackRollRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionAdversaryAttackRoll_MissingCampaignId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackRoll(context.Background(), &pb.SessionAdversaryAttackRollRequest{
		SessionId: "sess-1", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackRoll_MissingSessionId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackRoll(context.Background(), &pb.SessionAdversaryAttackRollRequest{
		CampaignId: "camp-1", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackRoll_MissingAdversaryId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackRoll(context.Background(), &pb.SessionAdversaryAttackRollRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackRoll_RequiresDomainEngine(t *testing.T) {
	svc := newAdversaryDamageTestService()
	svc.stores.Domain = nil
	_, err := svc.SessionAdversaryAttackRoll(context.Background(), &pb.SessionAdversaryAttackRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionAdversaryAttackRoll_Success(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)

	payload := action.RollResolvePayload{
		RequestID: "req-adv-roll-success",
		RollSeq:   1,
		Results: map[string]any{
			"rolls":        []int{7},
			"roll":         7,
			"modifier":     0,
			"total":        7,
			"advantage":    0,
			"disadvantage": 0,
		},
		SystemData: map[string]any{
			"character_id": "adv-1",
			"adversary_id": "adv-1",
			"roll_kind":    "adversary_roll",
			"roll":         7,
			"modifier":     0,
			"total":        7,
			"advantage":    0,
			"disadvantage": 0,
		},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode adversary roll payload: %v", err)
	}

	svc.stores.Domain = &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("action.roll_resolved"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-adv-roll-success",
				EntityType:    "adversary",
				EntityID:      "adv-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}

	ctx := grpcmeta.WithRequestID(context.Background(), "req-adv-roll-success")
	resp, err := svc.SessionAdversaryAttackRoll(ctx, &pb.SessionAdversaryAttackRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
	})
	if err != nil {
		t.Fatalf("SessionAdversaryAttackRoll returned error: %v", err)
	}
	if resp.RollSeq == 0 {
		t.Fatal("expected non-zero roll seq")
	}
}

func TestSessionAdversaryAttackRoll_UsesDomainEngine(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)

	payload := action.RollResolvePayload{
		RequestID: "req-adv-roll",
		RollSeq:   1,
		Results: map[string]any{
			"rolls":        []int{12, 18},
			"roll":         18,
			"modifier":     2,
			"total":        20,
			"advantage":    1,
			"disadvantage": 0,
		},
		SystemData: map[string]any{
			"character_id": "adv-1",
			"adversary_id": "adv-1",
			"roll_kind":    "adversary_roll",
			"roll":         18,
			"modifier":     2,
			"total":        20,
			"advantage":    1,
			"disadvantage": 0,
		},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode adversary roll payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("action.roll_resolved"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-adv-roll",
				EntityType:    "adversary",
				EntityID:      "adv-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}

	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(context.Background(), "req-adv-roll")
	resp, err := svc.SessionAdversaryAttackRoll(ctx, &pb.SessionAdversaryAttackRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
	})
	if err != nil {
		t.Fatalf("SessionAdversaryAttackRoll returned error: %v", err)
	}
	if resp.RollSeq == 0 {
		t.Fatal("expected non-zero roll seq")
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if domain.lastCommand.Type != command.Type("action.roll.resolve") {
		t.Fatalf("command type = %s, want %s", domain.lastCommand.Type, "action.roll.resolve")
	}
	if got := len(eventStore.Events["camp-1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.Events["camp-1"][0].Type != event.Type("action.roll_resolved") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["camp-1"][0].Type, event.Type("action.roll_resolved"))
	}
}

// --- SessionAdversaryActionCheck tests ---

func TestSessionAdversaryActionCheck_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionAdversaryActionCheck(context.Background(), &pb.SessionAdversaryActionCheckRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionAdversaryActionCheck_MissingCampaignId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryActionCheck(context.Background(), &pb.SessionAdversaryActionCheckRequest{
		SessionId: "sess-1", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryActionCheck_MissingSessionId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryActionCheck(context.Background(), &pb.SessionAdversaryActionCheckRequest{
		CampaignId: "camp-1", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryActionCheck_MissingAdversaryId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryActionCheck(context.Background(), &pb.SessionAdversaryActionCheckRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryActionCheck_RequiresDomainEngine(t *testing.T) {
	svc := newAdversaryDamageTestService()
	svc.stores.Domain = nil
	_, err := svc.SessionAdversaryActionCheck(context.Background(), &pb.SessionAdversaryActionCheckRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
		Difficulty:  10,
	})
	if err != nil {
		t.Fatalf("SessionAdversaryActionCheck returned error: %v", err)
	}
}

func TestSessionAdversaryActionCheck_Success(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := grpcmeta.WithRequestID(context.Background(), "req-adv-action-success")
	resp, err := svc.SessionAdversaryActionCheck(ctx, &pb.SessionAdversaryActionCheckRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
		Difficulty:  10,
	})
	if err != nil {
		t.Fatalf("SessionAdversaryActionCheck returned error: %v", err)
	}
	if resp.RollSeq == 0 {
		t.Fatal("expected non-zero roll seq")
	}
}

func TestSessionAdversaryActionCheck_DoesNotRequireDomainEngine(t *testing.T) {
	svc := newAdversaryDamageTestService()
	resp, err := svc.SessionAdversaryActionCheck(context.Background(), &pb.SessionAdversaryActionCheckRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
		Difficulty:  10,
	})
	if err != nil {
		t.Fatalf("SessionAdversaryActionCheck returned error: %v", err)
	}
	if resp.RollSeq == 0 {
		t.Fatal("expected non-zero roll seq")
	}
}

// --- SessionAdversaryAttackFlow tests ---

func TestSessionAdversaryAttackFlow_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionAdversaryAttackFlow_MissingCampaignId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackFlow_MissingSessionId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackFlow_MissingAdversaryId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackFlow_MissingTargetId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackFlow_MissingDamage(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", AdversaryId: "adv-1", TargetId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackFlow_MissingDamageType(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", AdversaryId: "adv-1", TargetId: "char-1",
		Damage: &pb.DaggerheartAttackDamageSpec{},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

// --- SessionGroupActionFlow tests ---

func TestSessionGroupActionFlow_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionGroupActionFlow_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionGroupActionFlow_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionGroupActionFlow_MissingLeader(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionGroupActionFlow_MissingLeaderTrait(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", LeaderCharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionGroupActionFlow_MissingDifficulty(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", LeaderCharacterId: "char-1", LeaderTrait: "agility",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionGroupActionFlow_MissingSupporters(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", LeaderCharacterId: "char-1", LeaderTrait: "agility", Difficulty: 10,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

// --- SessionTagTeamFlow tests ---

func TestSessionTagTeamFlow_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionTagTeamFlow_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionTagTeamFlow_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionTagTeamFlow_MissingDifficulty(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionTagTeamFlow_MissingFirst(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Difficulty: 10,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionTagTeamFlow_MissingSecond(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Difficulty: 10,
		First: &pb.TagTeamParticipant{CharacterId: "char-1", Trait: "agility"},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionTagTeamFlow_SameParticipant(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Difficulty: 10,
		First:               &pb.TagTeamParticipant{CharacterId: "char-1", Trait: "agility"},
		Second:              &pb.TagTeamParticipant{CharacterId: "char-1", Trait: "strength"},
		SelectedCharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

// --- ApplyRollOutcome tests ---

func TestApplyRollOutcome_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyRollOutcome(context.Background(), &pb.ApplyRollOutcomeRequest{
		SessionId: "s1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyRollOutcome_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyRollOutcome(context.Background(), &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRollOutcome_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := withCampaignSessionMetadata(context.Background(), "camp-1", "")
	_, err := svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		RollSeq: 1,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRollOutcome_MissingRollSeq(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRollOutcome_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	svc.stores.Domain = nil

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-outcome-required",
		RollSeq:   1,
		Results:   map[string]any{"d20": 20},
		Outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_ACTION.String(),
			"hope_fear":    true,
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-roll-outcome-required",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-outcome-required",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	ctx := grpcmeta.WithRequestID(withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"), "req-roll-outcome-required")
	_, err = svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyRollOutcome_IdempotentWhenAlreadyAppliedEvenWithOpenGate(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.Snapshots["camp-1"] = storage.DaggerheartSnapshot{
		CampaignID: "camp-1",
		GMFear:     3,
	}

	svc.stores.SessionGate = &fakeOpenSessionGateStore{
		gate: storage.SessionGate{
			CampaignID: "camp-1",
			SessionID:  "sess-1",
			GateID:     "gate-open",
			GateType:   "gm_consequence",
			Reason:     "gm_consequence",
			Status:     session.GateStatusOpen,
			CreatedAt:  now,
		},
	}
	svc.stores.SessionSpotlight = &fakeSessionSpotlightStateStore{
		exists: true,
		spotlight: storage.SessionSpotlight{
			CampaignID:    "camp-1",
			SessionID:     "sess-1",
			SpotlightType: session.SpotlightTypeGM,
		},
	}

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-duplicate",
		RollSeq:   1,
		Results:   map[string]any{"d20": 1},
		Outcome:   pb.Outcome_FAILURE_WITH_FEAR.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_ACTION.String(),
			"hope_fear":    true,
			"gm_move":      true,
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-roll-duplicate",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-duplicate",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	_, err = eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now.Add(time.Second),
		Type:        event.Type("action.outcome_applied"),
		SessionID:   "sess-1",
		RequestID:   "req-roll-duplicate",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "outcome",
		EntityID:    "req-roll-duplicate",
		PayloadJSON: []byte(`{"request_id":"req-roll-duplicate","roll_seq":1}`),
	})
	if err != nil {
		t.Fatalf("append outcome event: %v", err)
	}

	domain := svc.stores.Domain.(*fakeDomainEngine)
	ctx := grpcmeta.WithRequestID(withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"), "req-roll-duplicate")
	resp, err := svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if !resp.RequiresComplication {
		t.Fatal("expected requires complication to be true")
	}
	if resp.Updated == nil || resp.Updated.GmFear == nil {
		t.Fatal("expected gm fear in idempotent response")
	}
	if got, want := int(resp.Updated.GetGmFear()), 3; got != want {
		t.Fatalf("gm fear = %d, want %d", got, want)
	}
	if domain.calls != 0 {
		t.Fatalf("expected no new domain commands for duplicate outcome, got %d", domain.calls)
	}
}

func TestApplyRollOutcome_AlreadyAppliedStillEnsuresComplicationGate(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	dhStore.Snapshots["camp-1"] = storage.DaggerheartSnapshot{
		CampaignID: "camp-1",
		GMFear:     2,
	}

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-gate-retry",
		RollSeq:   1,
		Results:   map[string]any{"d20": 1},
		Outcome:   pb.Outcome_FAILURE_WITH_FEAR.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_ACTION.String(),
			"hope_fear":    true,
			"gm_move":      true,
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-roll-gate-retry",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-gate-retry",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	_, err = eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now.Add(time.Second),
		Type:        event.Type("action.outcome_applied"),
		SessionID:   "sess-1",
		RequestID:   "req-roll-gate-retry",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "outcome",
		EntityID:    "req-roll-gate-retry",
		PayloadJSON: []byte(`{"request_id":"req-roll-gate-retry","roll_seq":1}`),
	})
	if err != nil {
		t.Fatalf("append outcome event: %v", err)
	}

	gatePayload := session.GateOpenedPayload{
		GateID:   "gate-1",
		GateType: "gm_consequence",
		Reason:   "gm_consequence",
		Metadata: map[string]any{"roll_seq": uint64(rollEvent.Seq), "request_id": "req-roll-gate-retry"},
	}
	gateJSON, err := json.Marshal(gatePayload)
	if err != nil {
		t.Fatalf("encode gate payload: %v", err)
	}

	spotlightPayload := session.SpotlightSetPayload{SpotlightType: string(session.SpotlightTypeGM)}
	spotlightJSON, err := json.Marshal(spotlightPayload)
	if err != nil {
		t.Fatalf("encode spotlight payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("session.gate_open"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("session.gate_opened"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-roll-gate-retry",
				EntityType:  "session_gate",
				EntityID:    "gate-1",
				PayloadJSON: gateJSON,
			}),
		},
		command.Type("session.spotlight_set"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("session.spotlight_set"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-roll-gate-retry",
				EntityType:  "session_spotlight",
				EntityID:    "sess-1",
				PayloadJSON: spotlightJSON,
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"), "req-roll-gate-retry")
	resp, err := svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if !resp.RequiresComplication {
		t.Fatal("expected requires complication to be true")
	}
	if domain.calls != 2 {
		t.Fatalf("expected domain to be called twice for gate recovery, got %d", domain.calls)
	}
	var foundGate bool
	var foundSpotlight bool
	for _, cmd := range domain.commands {
		switch cmd.Type {
		case command.Type("session.gate_open"):
			foundGate = true
		case command.Type("session.spotlight_set"):
			foundSpotlight = true
		}
	}
	if !foundGate {
		t.Fatal("expected session gate open command")
	}
	if !foundSpotlight {
		t.Fatal("expected session spotlight set command")
	}
}

func TestApplyRollOutcome_PartialRetrySkipsRepeatedGMFearSet(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	dhStore.Snapshots["camp-1"] = storage.DaggerheartSnapshot{
		CampaignID: "camp-1",
		GMFear:     1,
	}

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-partial-retry",
		RollSeq:   1,
		Results:   map[string]any{"d20": 1},
		Outcome:   pb.Outcome_FAILURE_WITH_FEAR.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_ACTION.String(),
			"hope_fear":    true,
			"gm_move":      true,
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-roll-partial-retry",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-partial-retry",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	_, err = eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:    "camp-1",
		Timestamp:     now.Add(time.Second),
		Type:          event.Type("sys.daggerheart.gm_fear_changed"),
		SessionID:     "sess-1",
		RequestID:     "req-roll-partial-retry",
		ActorType:     event.ActorTypeSystem,
		EntityType:    "campaign",
		EntityID:      "camp-1",
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   []byte(`{"before":0,"after":1}`),
	})
	if err != nil {
		t.Fatalf("append gm fear event: %v", err)
	}

	gatePayload := session.GateOpenedPayload{
		GateID:   "gate-1",
		GateType: "gm_consequence",
		Reason:   "gm_consequence",
		Metadata: map[string]any{"roll_seq": uint64(rollEvent.Seq), "request_id": "req-roll-partial-retry"},
	}
	gateJSON, err := json.Marshal(gatePayload)
	if err != nil {
		t.Fatalf("encode gate payload: %v", err)
	}

	spotlightPayload := session.SpotlightSetPayload{SpotlightType: string(session.SpotlightTypeGM)}
	spotlightJSON, err := json.Marshal(spotlightPayload)
	if err != nil {
		t.Fatalf("encode spotlight payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(
				event.Event{
					CampaignID:  "camp-1",
					Type:        event.Type("action.outcome_applied"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					SessionID:   "sess-1",
					RequestID:   "req-roll-partial-retry",
					EntityType:  "outcome",
					EntityID:    "req-roll-partial-retry",
					PayloadJSON: []byte(`{"request_id":"req-roll-partial-retry","roll_seq":1}`),
				},
				event.Event{
					CampaignID:  "camp-1",
					Type:        event.Type("session.gate_opened"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					SessionID:   "sess-1",
					RequestID:   "req-roll-partial-retry",
					EntityType:  "session_gate",
					EntityID:    "gate-1",
					PayloadJSON: gateJSON,
				},
				event.Event{
					CampaignID:  "camp-1",
					Type:        event.Type("session.spotlight_set"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					SessionID:   "sess-1",
					RequestID:   "req-roll-partial-retry",
					EntityType:  "session_spotlight",
					EntityID:    "sess-1",
					PayloadJSON: spotlightJSON,
				},
			),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"), "req-roll-partial-retry")
	resp, err := svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if !resp.RequiresComplication {
		t.Fatal("expected requires complication to be true")
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	var payload action.OutcomeApplyPayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode outcome command payload: %v", err)
	}
	for _, effect := range payload.PreEffects {
		if effect.Type == "sys.daggerheart.gm_fear_changed" {
			t.Fatal("did not expect gm fear pre_effect on partial retry")
		}
	}
	if snap := dhStore.Snapshots["camp-1"]; snap.GMFear != 1 {
		t.Fatalf("gm fear = %d, want %d", snap.GMFear, 1)
	}
}

func TestApplyRollOutcome_PartialRetrySkipsRepeatedCharacterPatch(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-patch-retry",
		RollSeq:   1,
		Results:   map[string]any{"d20": 20},
		Outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_ACTION.String(),
			"hope_fear":    true,
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-roll-patch-retry",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-patch-retry",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	_, err = eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:    "camp-1",
		Timestamp:     now.Add(time.Second),
		Type:          event.Type("sys.daggerheart.character_state_patched"),
		SessionID:     "sess-1",
		RequestID:     "req-roll-patch-retry",
		ActorType:     event.ActorTypeSystem,
		EntityType:    "character",
		EntityID:      "char-1",
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","hope_before":2,"hope_after":3,"stress_before":3,"stress_after":3}`),
	})
	if err != nil {
		t.Fatalf("append patch event: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_applied"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-roll-patch-retry",
				EntityType:  "outcome",
				EntityID:    "req-roll-patch-retry",
				PayloadJSON: []byte(`{"request_id":"req-roll-patch-retry"}`),
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"), "req-roll-patch-retry")
	resp, err := svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if resp.RollSeq != rollEvent.Seq {
		t.Fatalf("roll seq = %d, want %d", resp.RollSeq, rollEvent.Seq)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 || domain.commands[0].Type != command.Type("action.outcome.apply") {
		t.Fatalf("expected only outcome apply command, got %+v", domain.commands)
	}
}

func TestApplyRollOutcome_AlreadyAppliedWithOpenGateRepairsSpotlight(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	dhStore.Snapshots["camp-1"] = storage.DaggerheartSnapshot{
		CampaignID: "camp-1",
		GMFear:     2,
	}

	svc.stores.SessionGate = &fakeOpenSessionGateStore{
		gate: storage.SessionGate{
			CampaignID: "camp-1",
			SessionID:  "sess-1",
			GateID:     "gate-open",
			GateType:   "gm_consequence",
			Reason:     "gm_consequence",
			Status:     session.GateStatusOpen,
			CreatedAt:  now,
		},
	}

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-open-gate",
		RollSeq:   1,
		Results:   map[string]any{"d20": 1},
		Outcome:   pb.Outcome_FAILURE_WITH_FEAR.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_ACTION.String(),
			"hope_fear":    true,
			"gm_move":      true,
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-roll-open-gate",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-open-gate",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	_, err = eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now.Add(time.Second),
		Type:        event.Type("action.outcome_applied"),
		SessionID:   "sess-1",
		RequestID:   "req-roll-open-gate",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "outcome",
		EntityID:    "req-roll-open-gate",
		PayloadJSON: []byte(`{"request_id":"req-roll-open-gate","roll_seq":1}`),
	})
	if err != nil {
		t.Fatalf("append outcome event: %v", err)
	}

	spotlightPayload := session.SpotlightSetPayload{SpotlightType: string(session.SpotlightTypeGM)}
	spotlightJSON, err := json.Marshal(spotlightPayload)
	if err != nil {
		t.Fatalf("encode spotlight payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("session.spotlight_set"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("session.spotlight_set"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-roll-open-gate",
				EntityType:  "session_spotlight",
				EntityID:    "sess-1",
				PayloadJSON: spotlightJSON,
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"), "req-roll-open-gate")
	resp, err := svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if !resp.RequiresComplication {
		t.Fatal("expected requires complication to be true")
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once for spotlight repair, got %d", domain.calls)
	}
	if len(domain.commands) != 1 || domain.commands[0].Type != command.Type("session.spotlight_set") {
		t.Fatalf("expected spotlight set command, got %+v", domain.commands)
	}
}

func TestApplyRollOutcome_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-1",
		RollSeq:   1,
		Results:   map[string]any{"d20": 20},
		Outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_ACTION.String(),
			"hope_fear":    true,
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-roll-1",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-1",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	state := dhStore.States["camp-1:char-1"]
	hopeBefore := state.Hope
	hopeMax := state.HopeMax
	if hopeMax == 0 {
		hopeMax = daggerheart.HopeMax
	}
	hopeAfter := hopeBefore + 1
	if hopeAfter > hopeMax {
		hopeAfter = hopeMax
	}
	stressBefore := state.Stress
	stressAfter := stressBefore
	patchPayload := daggerheart.CharacterStatePatchedPayload{
		CharacterID:  "char-1",
		HopeBefore:   &hopeBefore,
		HopeAfter:    &hopeAfter,
		StressBefore: &stressBefore,
		StressAfter:  &stressAfter,
	}
	patchJSON, err := json.Marshal(patchPayload)
	if err != nil {
		t.Fatalf("encode patch payload: %v", err)
	}

	svc.stores.Domain = &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-roll-1",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   patchJSON,
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_applied"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-roll-1",
				EntityType:  "outcome",
				EntityID:    "req-roll-1",
				PayloadJSON: []byte(`{"request_id":"req-roll-1"}`),
			}),
		},
	}}

	ctx := grpcmeta.WithRequestID(withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"), "req-roll-1")
	resp, err := svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if resp.RollSeq == 0 {
		t.Fatal("expected roll seq in response")
	}
}

func TestApplyRollOutcome_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	state := dhStore.States["camp-1:char-1"]
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-1",
		RollSeq:   1,
		Results:   map[string]any{"d20": 20},
		Outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}

	hopeBefore := state.Hope
	hopeMax := state.HopeMax
	if hopeMax == 0 {
		hopeMax = daggerheart.HopeMax
	}
	hopeAfter := hopeBefore + 1
	if hopeAfter > hopeMax {
		hopeAfter = hopeMax
	}
	stressBefore := state.Stress
	stressAfter := stressBefore
	patchPayload := daggerheart.CharacterStatePatchedPayload{
		CharacterID:  "char-1",
		HopeBefore:   &hopeBefore,
		HopeAfter:    &hopeAfter,
		StressBefore: &stressBefore,
		StressAfter:  &stressAfter,
	}
	patchJSON, err := json.Marshal(patchPayload)
	if err != nil {
		t.Fatalf("encode patch payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-roll-1",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-1",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-roll-1",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   patchJSON,
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_applied"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-roll-1",
				EntityType:  "outcome",
				EntityID:    "req-roll-1",
				PayloadJSON: []byte(`{"request_id":"req-roll-1","roll_seq":1}`),
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"), "req-roll-1")
	_, err = svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if domain.calls != 2 {
		t.Fatalf("expected domain to be called twice, got %d", domain.calls)
	}
	if len(domain.commands) != 2 {
		t.Fatalf("expected 2 domain commands, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.character_state.patch") {
		t.Fatalf("first command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.character_state.patch")
	}
	if domain.commands[1].Type != command.Type("action.outcome.apply") {
		t.Fatalf("second command type = %s, want %s", domain.commands[1].Type, "action.outcome.apply")
	}
	var outcomePayload action.OutcomeApplyPayload
	if err := json.Unmarshal(domain.commands[1].PayloadJSON, &outcomePayload); err != nil {
		t.Fatalf("decode outcome command payload: %v", err)
	}
	if len(outcomePayload.PreEffects) != 0 {
		t.Fatalf("pre_effects length = %d, want 0", len(outcomePayload.PreEffects))
	}
	found := false
	for _, evt := range eventStore.Events["camp-1"] {
		if evt.Type == event.Type("action.outcome_applied") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected outcome applied event")
	}
}

func TestApplyRollOutcome_UsesSystemAndCoreCommandBoundary(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	state := dhStore.States["camp-1:char-1"]
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-single-boundary",
		RollSeq:   1,
		Results:   map[string]any{"d20": 20},
		Outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-roll-single-boundary",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-single-boundary",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	hopeBefore := state.Hope
	hopeAfter := hopeBefore + 1
	patchPayload := daggerheart.CharacterStatePatchedPayload{
		CharacterID: "char-1",
		HopeBefore:  &hopeBefore,
		HopeAfter:   &hopeAfter,
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
				RequestID:     "req-roll-single-boundary",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   patchJSON,
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_applied"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-roll-single-boundary",
				EntityType:  "outcome",
				EntityID:    "req-roll-single-boundary",
				PayloadJSON: []byte(`{"request_id":"req-roll-single-boundary","roll_seq":1}`),
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(
		withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"),
		"req-roll-single-boundary",
	)
	_, err = svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if domain.calls != 2 {
		t.Fatalf("expected domain to be called twice, got %d", domain.calls)
	}
	if len(domain.commands) != 2 {
		t.Fatalf("expected 2 domain commands, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.character_state.patch") {
		t.Fatalf("first command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.character_state.patch")
	}
	if domain.commands[1].Type != command.Type("action.outcome.apply") {
		t.Fatalf("second command type = %s, want %s", domain.commands[1].Type, "action.outcome.apply")
	}
}

func TestApplyRollOutcome_UsesDomainEngineForGmFear(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.Snapshots["camp-1"] = storage.DaggerheartSnapshot{CampaignID: "camp-1", GMFear: 1}
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-1",
		RollSeq:   1,
		Results:   map[string]any{"d20": 1},
		Outcome:   pb.Outcome_FAILURE_WITH_FEAR.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-roll-1",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-1",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	gatePayload := session.GateOpenedPayload{
		GateID:   "gate-1",
		GateType: "gm_consequence",
		Reason:   "gm_consequence",
		Metadata: map[string]any{"roll_seq": uint64(rollEvent.Seq), "request_id": "req-roll-1"},
	}
	gateJSON, err := json.Marshal(gatePayload)
	if err != nil {
		t.Fatalf("encode gate payload: %v", err)
	}

	spotlightPayload := session.SpotlightSetPayload{SpotlightType: string(session.SpotlightTypeGM)}
	spotlightJSON, err := json.Marshal(spotlightPayload)
	if err != nil {
		t.Fatalf("encode spotlight payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.gm_fear.set"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.gm_fear_changed"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-roll-1",
				EntityType:    "campaign",
				EntityID:      "camp-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   []byte(`{"before":1,"after":2}`),
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(
				event.Event{
					CampaignID:  "camp-1",
					Type:        event.Type("action.outcome_applied"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					SessionID:   "sess-1",
					RequestID:   "req-roll-1",
					EntityType:  "outcome",
					EntityID:    "req-roll-1",
					PayloadJSON: []byte(`{"request_id":"req-roll-1","roll_seq":1}`),
				},
				event.Event{
					CampaignID:  "camp-1",
					Type:        event.Type("session.gate_opened"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					SessionID:   "sess-1",
					RequestID:   "req-roll-1",
					EntityType:  "session_gate",
					EntityID:    "gate-1",
					PayloadJSON: gateJSON,
				},
				event.Event{
					CampaignID:  "camp-1",
					Type:        event.Type("session.spotlight_set"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					SessionID:   "sess-1",
					RequestID:   "req-roll-1",
					EntityType:  "session_spotlight",
					EntityID:    "sess-1",
					PayloadJSON: spotlightJSON,
				},
			),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"), "req-roll-1")
	_, err = svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if domain.calls != 2 {
		t.Fatalf("expected domain to be called twice, got %d", domain.calls)
	}
	if len(domain.commands) != 2 {
		t.Fatalf("expected 2 domain commands, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.gm_fear.set") {
		t.Fatalf("first command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.gm_fear.set")
	}
	if domain.commands[1].Type != command.Type("action.outcome.apply") {
		t.Fatalf("second command type = %s, want %s", domain.commands[1].Type, "action.outcome.apply")
	}
	var payload action.OutcomeApplyPayload
	if err := json.Unmarshal(domain.commands[1].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode outcome command payload: %v", err)
	}
	if len(payload.PreEffects) != 0 {
		t.Fatalf("pre_effects length = %d, want 0", len(payload.PreEffects))
	}
	if len(payload.PostEffects) != 2 {
		t.Fatalf("post_effects length = %d, want 2", len(payload.PostEffects))
	}
	if got, want := payload.PostEffects[0].Type, "session.gate_opened"; got != want {
		t.Fatalf("post_effects[0].type = %s, want %s", got, want)
	}
	if got, want := payload.PostEffects[1].Type, "session.spotlight_set"; got != want {
		t.Fatalf("post_effects[1].type = %s, want %s", got, want)
	}
	var foundFearEvent bool
	var foundOutcomeEvent bool
	var foundGateEvent bool
	var foundSpotlightEvent bool
	for _, evt := range eventStore.Events["camp-1"] {
		switch evt.Type {
		case event.Type("sys.daggerheart.gm_fear_changed"):
			foundFearEvent = true
		case event.Type("action.outcome_applied"):
			foundOutcomeEvent = true
		case event.Type("session.gate_opened"):
			foundGateEvent = true
		case event.Type("session.spotlight_set"):
			foundSpotlightEvent = true
		}
	}
	if !foundFearEvent {
		t.Fatal("expected gm fear event")
	}
	if !foundOutcomeEvent {
		t.Fatal("expected outcome applied event")
	}
	if !foundGateEvent {
		t.Fatal("expected session gate opened event")
	}
	if !foundSpotlightEvent {
		t.Fatal("expected session spotlight set event")
	}
}

func TestApplyRollOutcome_UsesDomainEngineForGmConsequenceGate(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-1",
		RollSeq:   1,
		Results:   map[string]any{"d20": 1},
		Outcome:   pb.Outcome_FAILURE_WITH_FEAR.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-roll-1",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-1",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	gatePayload := session.GateOpenedPayload{
		GateID:   "gate-1",
		GateType: "gm_consequence",
		Reason:   "gm_consequence",
		Metadata: map[string]any{"roll_seq": uint64(rollEvent.Seq), "request_id": "req-roll-1"},
	}
	gateJSON, err := json.Marshal(gatePayload)
	if err != nil {
		t.Fatalf("encode gate payload: %v", err)
	}

	spotlightPayload := session.SpotlightSetPayload{SpotlightType: string(session.SpotlightTypeGM)}
	spotlightJSON, err := json.Marshal(spotlightPayload)
	if err != nil {
		t.Fatalf("encode spotlight payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.gm_fear.set"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.gm_fear_changed"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-roll-1",
				EntityType:    "campaign",
				EntityID:      "camp-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   []byte(`{"before":0,"after":1}`),
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(
				event.Event{
					CampaignID:  "camp-1",
					Type:        event.Type("action.outcome_applied"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					SessionID:   "sess-1",
					RequestID:   "req-roll-1",
					EntityType:  "outcome",
					EntityID:    "req-roll-1",
					PayloadJSON: []byte(`{"request_id":"req-roll-1","roll_seq":1}`),
				},
				event.Event{
					CampaignID:  "camp-1",
					Type:        event.Type("session.gate_opened"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					SessionID:   "sess-1",
					RequestID:   "req-roll-1",
					EntityType:  "session_gate",
					EntityID:    "gate-1",
					PayloadJSON: gateJSON,
				},
				event.Event{
					CampaignID:  "camp-1",
					Type:        event.Type("session.spotlight_set"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					SessionID:   "sess-1",
					RequestID:   "req-roll-1",
					EntityType:  "session_spotlight",
					EntityID:    "sess-1",
					PayloadJSON: spotlightJSON,
				},
			),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"), "req-roll-1")
	resp, err := svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if !resp.RequiresComplication {
		t.Fatal("expected requires complication to be true")
	}
	if domain.calls != 2 {
		t.Fatalf("expected domain to be called twice, got %d", domain.calls)
	}
	if len(domain.commands) != 2 {
		t.Fatalf("expected 2 domain commands, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.gm_fear.set") {
		t.Fatalf("first command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.gm_fear.set")
	}
	if domain.commands[1].Type != command.Type("action.outcome.apply") {
		t.Fatalf("second command type = %s, want %s", domain.commands[1].Type, "action.outcome.apply")
	}
	var payload action.OutcomeApplyPayload
	if err := json.Unmarshal(domain.commands[1].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode outcome command payload: %v", err)
	}
	if len(payload.PreEffects) != 0 {
		t.Fatalf("pre_effects length = %d, want 0", len(payload.PreEffects))
	}
	if len(payload.PostEffects) != 2 {
		t.Fatalf("post_effects length = %d, want 2", len(payload.PostEffects))
	}
	if payload.PostEffects[0].Type != "session.gate_opened" {
		t.Fatalf("post_effects[0].type = %s, want %s", payload.PostEffects[0].Type, "session.gate_opened")
	}
	if payload.PostEffects[1].Type != "session.spotlight_set" {
		t.Fatalf("post_effects[1].type = %s, want %s", payload.PostEffects[1].Type, "session.spotlight_set")
	}
}

func TestApplyRollOutcome_UsesDomainEngineForCharacterStatePatch(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	state := dhStore.States["camp-1:char-1"]
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-1",
		RollSeq:   1,
		Results:   map[string]any{"d20": 20},
		Outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-roll-1",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-1",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	hopeBefore := state.Hope
	hopeMax := state.HopeMax
	if hopeMax == 0 {
		hopeMax = daggerheart.HopeMax
	}
	hopeAfter := hopeBefore + 1
	if hopeAfter > hopeMax {
		hopeAfter = hopeMax
	}
	stressBefore := state.Stress
	stressAfter := stressBefore

	patchPayload := daggerheart.CharacterStatePatchedPayload{
		CharacterID:  "char-1",
		HopeBefore:   &hopeBefore,
		HopeAfter:    &hopeAfter,
		StressBefore: &stressBefore,
		StressAfter:  &stressAfter,
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
				RequestID:     "req-roll-1",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   patchJSON,
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_applied"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-roll-1",
				EntityType:  "outcome",
				EntityID:    "req-roll-1",
				PayloadJSON: []byte(`{"request_id":"req-roll-1","roll_seq":1}`),
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"), "req-roll-1")
	_, err = svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if domain.calls != 2 {
		t.Fatalf("expected domain to be called twice, got %d", domain.calls)
	}
	if len(domain.commands) != 2 {
		t.Fatalf("expected 2 domain commands, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.character_state.patch") {
		t.Fatalf("first command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.character_state.patch")
	}
	if domain.commands[1].Type != command.Type("action.outcome.apply") {
		t.Fatalf("second command type = %s, want %s", domain.commands[1].Type, "action.outcome.apply")
	}
	var got action.OutcomeApplyPayload
	if err := json.Unmarshal(domain.commands[1].PayloadJSON, &got); err != nil {
		t.Fatalf("decode outcome command payload: %v", err)
	}
	if len(got.PreEffects) != 0 {
		t.Fatalf("pre_effects length = %d, want 0", len(got.PreEffects))
	}
	var foundPatchEvent bool
	var foundOutcomeEvent bool
	for _, evt := range eventStore.Events["camp-1"] {
		switch evt.Type {
		case event.Type("sys.daggerheart.character_state_patched"):
			foundPatchEvent = true
		case event.Type("action.outcome_applied"):
			foundOutcomeEvent = true
		}
	}
	if !foundPatchEvent {
		t.Fatal("expected character state patched event")
	}
	if !foundOutcomeEvent {
		t.Fatal("expected outcome applied event")
	}
}

func TestApplyRollOutcome_UsesDomainEngineForConditionChange(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	profile := dhStore.Profiles["camp-1:char-1"]
	state := dhStore.States["camp-1:char-1"]
	state.Stress = profile.StressMax
	state.Conditions = []string{daggerheart.ConditionVulnerable}
	dhStore.States["camp-1:char-1"] = state
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-1",
		RollSeq:   1,
		Results:   map[string]any{"d20": 20},
		Outcome:   pb.Outcome_CRITICAL_SUCCESS.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"crit":         true,
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-roll-1",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-1",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	hopeBefore := state.Hope
	hopeMax := state.HopeMax
	if hopeMax == 0 {
		hopeMax = daggerheart.HopeMax
	}
	hopeAfter := hopeBefore + 1
	if hopeAfter > hopeMax {
		hopeAfter = hopeMax
	}
	stressBefore := profile.StressMax
	stressAfter := stressBefore - 1
	patchPayload := daggerheart.CharacterStatePatchedPayload{
		CharacterID:  "char-1",
		HopeBefore:   &hopeBefore,
		HopeAfter:    &hopeAfter,
		StressBefore: &stressBefore,
		StressAfter:  &stressAfter,
	}
	patchJSON, err := json.Marshal(patchPayload)
	if err != nil {
		t.Fatalf("encode patch payload: %v", err)
	}

	rollSeq := rollEvent.Seq
	conditionPayload := daggerheart.ConditionChangedPayload{
		CharacterID:      "char-1",
		ConditionsBefore: []string{daggerheart.ConditionVulnerable},
		ConditionsAfter:  []string{},
		Removed:          []string{daggerheart.ConditionVulnerable},
		RollSeq:          &rollSeq,
	}
	conditionJSON, err := json.Marshal(conditionPayload)
	if err != nil {
		t.Fatalf("encode condition payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-roll-1",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   patchJSON,
			}),
		},
		command.Type("sys.daggerheart.condition.change"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.condition_changed"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-roll-1",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   conditionJSON,
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_applied"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-roll-1",
				EntityType:  "outcome",
				EntityID:    "req-roll-1",
				PayloadJSON: []byte(`{"request_id":"req-roll-1","roll_seq":1}`),
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"), "req-roll-1")
	_, err = svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if domain.calls != 3 {
		t.Fatalf("expected domain to be called three times, got %d", domain.calls)
	}
	if len(domain.commands) != 3 {
		t.Fatalf("expected 3 domain commands, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.character_state.patch") {
		t.Fatalf("first command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.character_state.patch")
	}
	if domain.commands[1].Type != command.Type("sys.daggerheart.condition.change") {
		t.Fatalf("second command type = %s, want %s", domain.commands[1].Type, "sys.daggerheart.condition.change")
	}
	if domain.commands[2].Type != command.Type("action.outcome.apply") {
		t.Fatalf("third command type = %s, want %s", domain.commands[2].Type, "action.outcome.apply")
	}
	var payload action.OutcomeApplyPayload
	if err := json.Unmarshal(domain.commands[2].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode outcome command payload: %v", err)
	}
	if len(payload.PreEffects) != 0 {
		t.Fatalf("pre_effects length = %d, want 0", len(payload.PreEffects))
	}
	var foundConditionEvent bool
	var foundPatchEvent bool
	for _, evt := range eventStore.Events["camp-1"] {
		switch evt.Type {
		case event.Type("sys.daggerheart.condition_changed"):
			foundConditionEvent = true
		case event.Type("sys.daggerheart.character_state_patched"):
			foundPatchEvent = true
		}
	}
	if !foundConditionEvent {
		t.Fatal("expected condition changed event")
	}
	if !foundPatchEvent {
		t.Fatal("expected character state patched event")
	}
}

// --- ApplyAttackOutcome tests ---

func TestApplyAttackOutcome_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyAttackOutcome(context.Background(), &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "s1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyAttackOutcome_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyAttackOutcome(context.Background(), &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1, Targets: []string{"adv-1"},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAttackOutcome_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyAttackOutcome(context.Background(), &pb.DaggerheartApplyAttackOutcomeRequest{
		RollSeq: 1, Targets: []string{"adv-1"},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAttackOutcome_MissingRollSeq(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyAttackOutcome(ctx, &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "sess-1", Targets: []string{"adv-1"},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAttackOutcome_MissingTargets(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyAttackOutcome(ctx, &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAttackOutcome_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	svc.stores.Domain = nil

	rollPayload := action.RollResolvePayload{
		RequestID: "req-atk-outcome-required",
		RollSeq:   1,
		Results:   map[string]any{"d20": 20},
		Outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_ACTION.String(),
			"hope_fear":    true,
		},
	}
	rollPayloadJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-atk-outcome-required",
		ActorType:   event.ActorTypeSystem,
		EntityID:    "req-atk-outcome-required",
		EntityType:  "roll",
		PayloadJSON: rollPayloadJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	ctx := grpcmeta.WithRequestID(
		withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"),
		"req-atk-outcome-required",
	)
	_, err = svc.ApplyAttackOutcome(ctx, &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
		Targets:   []string{"char-2"},
	})
	if err != nil {
		t.Fatalf("ApplyAttackOutcome returned error: %v", err)
	}
}

func TestApplyAttackOutcome_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-atk-outcome-legacy",
		RollSeq:   1,
		Results:   map[string]any{"d20": 20},
		Outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_ACTION.String(),
			"hope_fear":    true,
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-atk-outcome-legacy",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-atk-outcome-legacy",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	svc.stores.Domain = nil

	ctx := grpcmeta.WithRequestID(
		withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"),
		"req-atk-outcome-legacy",
	)
	resp, err := svc.ApplyAttackOutcome(ctx, &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
		Targets:   []string{"char-2"},
	})
	if err != nil {
		t.Fatalf("ApplyAttackOutcome returned error: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("expected result in response")
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("expected character_id char-1, got %s", resp.CharacterId)
	}
	if len(resp.Targets) != 1 || resp.Targets[0] != "char-2" {
		t.Fatalf("expected targets [char-2], got %v", resp.Targets)
	}
	if resp.Result.GetOutcome() != pb.Outcome_SUCCESS_WITH_HOPE {
		t.Fatalf("expected outcome SUCCESS_WITH_HOPE, got %s", resp.Result.GetOutcome())
	}
	if !resp.Result.GetSuccess() {
		t.Fatal("expected attack outcome success")
	}
}

// --- ApplyAdversaryAttackOutcome tests ---

func TestApplyAdversaryAttackOutcome_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyAdversaryAttackOutcome(context.Background(), &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId: "s1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyAdversaryAttackOutcome_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyAdversaryAttackOutcome(context.Background(), &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1, Targets: []string{"char-1"},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryAttackOutcome_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyAdversaryAttackOutcome(context.Background(), &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		RollSeq: 1, Targets: []string{"char-1"},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryAttackOutcome_MissingRollSeq(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyAdversaryAttackOutcome(ctx, &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId: "sess-1", Targets: []string{"char-1"},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryAttackOutcome_MissingTargets(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyAdversaryAttackOutcome(ctx, &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryAttackOutcome_RequiresDomainEngine(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-adv-atk-outcome-required",
		RollSeq:   1,
		Results:   map[string]any{"rolls": []int{4}, "roll": 4, "modifier": 0, "total": 4, "advantage": 0, "disadvantage": 0},
		Outcome:   pb.Outcome_FAILURE_WITH_FEAR.String(),
		SystemData: map[string]any{
			"character_id": "adv-1",
			"adversary_id": "adv-1",
			"roll_kind":    "adversary_roll",
			"roll":         4,
			"modifier":     0,
			"total":        4,
			"advantage":    0,
			"disadvantage": 0,
		},
	}
	rollPayloadJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	outcomeJSON, err := json.Marshal(action.OutcomeApplyPayload{
		RequestID:      "req-adv-attack-1",
		RollSeq:        1,
		Targets:        []string{"char-1"},
		AppliedChanges: []action.OutcomeAppliedChange{},
	})
	if err != nil {
		t.Fatalf("encode outcome payload: %v", err)
	}

	svc.stores.Domain = &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.roll_resolved"),
				Timestamp:   time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-adv-atk-outcome-required",
				EntityType:  "adversary",
				EntityID:    "adv-1",
				PayloadJSON: rollPayloadJSON,
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("action.outcome_applied"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-adv-attack-1",
				EntityType:    "outcome",
				EntityID:      "req-adv-attack-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   outcomeJSON,
			}),
		},
	}}

	rollCtx := grpcmeta.WithRequestID(context.Background(), "req-adv-atk-outcome-required")
	rollResp, err := svc.SessionAdversaryAttackRoll(rollCtx, &pb.SessionAdversaryAttackRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
	})
	if err != nil {
		t.Fatalf("SessionAdversaryAttackRoll returned error: %v", err)
	}
	noDomainSvc := &DaggerheartService{stores: svc.stores, seedFunc: svc.seedFunc}
	noDomainSvc.stores.Domain = nil

	ctx := grpcmeta.WithRequestID(
		withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"),
		"req-adv-atk-outcome-required",
	)
	resp, err := noDomainSvc.ApplyAdversaryAttackOutcome(ctx, &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId:  "sess-1",
		RollSeq:    rollResp.RollSeq,
		Targets:    []string{"char-1"},
		Difficulty: 10,
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryAttackOutcome returned error: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("expected result in response")
	}
}

func TestApplyAdversaryAttackOutcome_UsesDomainEngine(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-adv-atk-outcome-legacy",
		RollSeq:   1,
		Results:   map[string]any{"rolls": []int{4}, "roll": 4, "modifier": 0, "total": 4, "advantage": 0, "disadvantage": 0},
		Outcome:   pb.Outcome_FAILURE_WITH_FEAR.String(),
		SystemData: map[string]any{
			"character_id": "adv-1",
			"adversary_id": "adv-1",
			"roll_kind":    "adversary_roll",
			"roll":         4,
			"modifier":     0,
			"total":        4,
			"advantage":    0,
			"disadvantage": 0,
		},
	}
	rollPayloadJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("action.roll_resolved"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-adv-atk-outcome-legacy",
				EntityType:    "adversary",
				EntityID:      "adv-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   rollPayloadJSON,
			}),
		},
	}}

	svc.stores.Domain = domain

	rollCtx := grpcmeta.WithRequestID(context.Background(), "req-adv-atk-outcome-legacy")
	rollResp, err := svc.SessionAdversaryAttackRoll(rollCtx, &pb.SessionAdversaryAttackRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
	})
	if err != nil {
		t.Fatalf("SessionAdversaryAttackRoll returned error: %v", err)
	}

	ctx := grpcmeta.WithRequestID(
		withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"),
		"req-adv-atk-outcome-legacy",
	)
	resp, err := svc.ApplyAdversaryAttackOutcome(ctx, &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId:  "sess-1",
		RollSeq:    rollResp.RollSeq,
		Targets:    []string{"char-1"},
		Difficulty: 10,
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryAttackOutcome returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if resp.Result == nil {
		t.Fatal("expected result in response")
	}
	if resp.AdversaryId != "adv-1" {
		t.Fatalf("expected adversary adv-1, got %s", resp.AdversaryId)
	}
}

// --- ApplyReactionOutcome tests ---

func TestApplyReactionOutcome_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyReactionOutcome(context.Background(), &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: "s1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyReactionOutcome_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyReactionOutcome(context.Background(), &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyReactionOutcome_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyReactionOutcome(context.Background(), &pb.DaggerheartApplyReactionOutcomeRequest{
		RollSeq: 1,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyReactionOutcome_MissingRollSeq(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyReactionOutcome(ctx, &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyReactionOutcome_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	rollCtx := grpcmeta.WithRequestID(context.Background(), "req-react-outcome-required")
	configureActionRollDomain(t, svc, "req-react-outcome-required")
	rollResp, err := svc.SessionActionRoll(rollCtx, &pb.SessionActionRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CharacterId: "char-1",
		Trait:       "agility",
		RollKind:    pb.RollKind_ROLL_KIND_REACTION,
		Difficulty:  10,
	})
	if err != nil {
		t.Fatalf("SessionActionRoll returned error: %v", err)
	}
	svc.stores.Domain = nil

	ctx := grpcmeta.WithRequestID(
		withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"),
		"req-react-outcome-required",
	)
	_, err = svc.ApplyReactionOutcome(ctx, &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollResp.RollSeq,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyReactionOutcome_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	configureNoopDomain(svc)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-react-outcome-legacy",
		RollSeq:   1,
		Results:   map[string]any{"d20": 12},
		Outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_REACTION.String(),
			"hope_fear":    false,
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-react-outcome-legacy",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-react-outcome-legacy",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	ctx := grpcmeta.WithRequestID(
		withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"),
		"req-react-outcome-legacy",
	)
	resp, err := svc.ApplyReactionOutcome(ctx, &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyReactionOutcome returned error: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("expected result in response")
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("expected character_id char-1, got %s", resp.CharacterId)
	}
	if resp.Result.GetOutcome() != pb.Outcome_SUCCESS_WITH_HOPE {
		t.Fatalf("expected outcome SUCCESS_WITH_HOPE, got %s", resp.Result.GetOutcome())
	}
	if !resp.Result.GetSuccess() {
		t.Fatal("expected reaction success")
	}
}

// --- Success path tests for flow handlers ---

func TestSessionAttackFlow_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-attack-1",
		RollSeq:   1,
		Results:   map[string]any{"d20": 8},
		Outcome:   pb.Outcome_FAILURE_WITH_FEAR.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_ACTION.String(),
			"hope_fear":    false,
			"gm_move":      false,
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	outcomeJSON, err := json.Marshal(action.OutcomeApplyPayload{
		RequestID:      "req-attack-1",
		RollSeq:        1,
		Targets:        []string{"char-2"},
		AppliedChanges: []action.OutcomeAppliedChange{},
	})
	if err != nil {
		t.Fatalf("encode outcome payload: %v", err)
	}

	svc.stores.Domain = &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.roll_resolved"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-attack-1",
				EntityType:  "roll",
				EntityID:    "req-attack-1",
				PayloadJSON: rollJSON,
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_applied"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-attack-1",
				EntityID:    "req-attack-1",
				EntityType:  "outcome",
				PayloadJSON: outcomeJSON,
			}),
		},
	}}
	ctx := grpcmeta.WithRequestID(context.Background(), "req-attack-1")
	resp, err := svc.SessionAttackFlow(ctx, &pb.SessionAttackFlowRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CharacterId: "char-1",
		Trait:       "agility",
		Difficulty:  10,
		TargetId:    "char-2",
		Damage: &pb.DaggerheartAttackDamageSpec{
			DamageType:         pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
			SourceCharacterIds: []string{"char-1"},
		},
		DamageDice: []*pb.DiceSpec{{Sides: 6, Count: 1}},
	})
	if err != nil {
		t.Fatalf("SessionAttackFlow returned error: %v", err)
	}
	if resp.ActionRoll == nil {
		t.Fatal("expected action roll in response")
	}
	if resp.AttackOutcome == nil {
		t.Fatal("expected attack outcome in response")
	}
}

func TestSessionAdversaryAttackFlow_Success(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-adv-attack-1",
		RollSeq:   1,
		Results: map[string]any{
			"rolls":        []int{1},
			"roll":         1,
			"modifier":     0,
			"total":        1,
			"advantage":    0,
			"disadvantage": 0,
		},
		Outcome: pb.Outcome_FAILURE_WITH_HOPE.String(),
		SystemData: map[string]any{
			"character_id": "adv-1",
			"adversary_id": "adv-1",
			"roll_kind":    "adversary_roll",
			"roll":         1,
			"modifier":     0,
			"total":        1,
			"advantage":    0,
			"disadvantage": 0,
		},
	}
	rollPayloadJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}

	svc.stores.Domain = &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("action.roll_resolved"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-adv-attack-1",
				EntityType:    "adversary",
				EntityID:      "adv-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   rollPayloadJSON,
			}),
		},
	}}

	ctx := grpcmeta.WithRequestID(context.Background(), "req-adv-attack-1")
	resp, err := svc.SessionAdversaryAttackFlow(ctx, &pb.SessionAdversaryAttackFlowRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
		TargetId:    "char-1",
		Difficulty:  10,
		Damage: &pb.DaggerheartAttackDamageSpec{
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
		DamageDice: []*pb.DiceSpec{{Sides: 6, Count: 1}},
	})
	if err != nil {
		t.Fatalf("SessionAdversaryAttackFlow returned error: %v", err)
	}
	if resp.AttackRoll == nil {
		t.Fatal("expected attack roll in response")
	}
	if resp.AttackOutcome == nil {
		t.Fatal("expected attack outcome in response")
	}
}

func TestSessionGroupActionFlow_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)

	rollJSON, err := json.Marshal(action.RollResolvePayload{
		RequestID: "req-group-1",
		RollSeq:   1,
		Results:   map[string]any{"d20": 20},
		Outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_ACTION.String(),
			"hope_fear":    false,
		},
	})
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	outcomeJSON, err := json.Marshal(action.OutcomeApplyPayload{
		RequestID: "req-group-1",
		RollSeq:   1,
		Targets:   []string{"char-1", "char-2"},
	})
	if err != nil {
		t.Fatalf("encode outcome payload: %v", err)
	}

	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	svc.stores.Domain = &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.roll_resolved"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-group-1",
				EntityType:  "roll",
				EntityID:    "req-group-1",
				PayloadJSON: rollJSON,
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_applied"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-group-1",
				EntityType:  "outcome",
				EntityID:    "req-group-1",
				PayloadJSON: outcomeJSON,
			}),
		},
	}}

	ctx := grpcmeta.WithRequestID(context.Background(), "req-group-1")
	resp, err := svc.SessionGroupActionFlow(ctx, &pb.SessionGroupActionFlowRequest{
		CampaignId:        "camp-1",
		SessionId:         "sess-1",
		LeaderCharacterId: "char-1",
		LeaderTrait:       "agility",
		Difficulty:        10,
		Supporters: []*pb.GroupActionSupporter{
			{CharacterId: "char-2", Trait: "strength"},
		},
	})
	if err != nil {
		t.Fatalf("SessionGroupActionFlow returned error: %v", err)
	}
	if resp.LeaderRoll == nil {
		t.Fatal("expected leader roll in response")
	}
	if len(resp.SupporterRolls) == 0 {
		t.Fatal("expected supporter rolls in response")
	}
}

func TestSessionTagTeamFlow_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)

	rollJSON, err := json.Marshal(action.RollResolvePayload{
		RequestID: "req-tagteam-1",
		RollSeq:   1,
		Results:   map[string]any{"d20": 18},
		Outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_ACTION.String(),
			"hope_fear":    false,
		},
	})
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	outcomeJSON, err := json.Marshal(action.OutcomeApplyPayload{
		RequestID: "req-tagteam-1",
		RollSeq:   1,
		Targets:   []string{"char-1", "char-2"},
	})
	if err != nil {
		t.Fatalf("encode outcome payload: %v", err)
	}

	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	svc.stores.Domain = &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.roll_resolved"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-tagteam-1",
				EntityType:  "roll",
				EntityID:    "req-tagteam-1",
				PayloadJSON: rollJSON,
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_applied"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-tagteam-1",
				EntityType:  "outcome",
				EntityID:    "req-tagteam-1",
				PayloadJSON: outcomeJSON,
			}),
		},
	}}

	ctx := grpcmeta.WithRequestID(context.Background(), "req-tagteam-1")
	resp, err := svc.SessionTagTeamFlow(ctx, &pb.SessionTagTeamFlowRequest{
		CampaignId:          "camp-1",
		SessionId:           "sess-1",
		Difficulty:          10,
		First:               &pb.TagTeamParticipant{CharacterId: "char-1", Trait: "agility"},
		Second:              &pb.TagTeamParticipant{CharacterId: "char-2", Trait: "strength"},
		SelectedCharacterId: "char-1",
	})
	if err != nil {
		t.Fatalf("SessionTagTeamFlow returned error: %v", err)
	}
	if resp.FirstRoll == nil {
		t.Fatal("expected first roll in response")
	}
	if resp.SecondRoll == nil {
		t.Fatal("expected second roll in response")
	}
}

func TestSessionGroupActionFlow_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-group-action",
		RollSeq:   1,
		Results:   map[string]any{"d20": 20},
		Outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_ACTION.String(),
			"hope_fear":    false,
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	outcomePayload, err := json.Marshal(action.OutcomeApplyPayload{
		RequestID: "req-group-action",
		RollSeq:   1,
		Targets:   []string{"char-1", "char-2"},
	})
	if err != nil {
		t.Fatalf("encode outcome payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.roll_resolved"),
				Timestamp:   time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-group-action",
				EntityType:  "roll",
				EntityID:    "req-group-action",
				PayloadJSON: rollJSON,
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_applied"),
				Timestamp:   time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-group-action",
				EntityType:  "outcome",
				EntityID:    "req-group-action",
				PayloadJSON: outcomePayload,
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(context.Background(), "req-group-action")
	_, err = svc.SessionGroupActionFlow(ctx, &pb.SessionGroupActionFlowRequest{
		CampaignId:        "camp-1",
		SessionId:         "sess-1",
		LeaderCharacterId: "char-1",
		LeaderTrait:       "agility",
		Difficulty:        10,
		Supporters: []*pb.GroupActionSupporter{
			{CharacterId: "char-2", Trait: "strength"},
		},
	})
	if err != nil {
		t.Fatalf("SessionGroupActionFlow returned error: %v", err)
	}
	if len(domain.commands) != 3 {
		t.Fatalf("expected 3 domain commands, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("action.roll.resolve") {
		t.Fatalf("first command type = %s, want %s", domain.commands[0].Type, "action.roll.resolve")
	}
	if domain.commands[1].Type != command.Type("action.roll.resolve") {
		t.Fatalf("second command type = %s, want %s", domain.commands[1].Type, "action.roll.resolve")
	}
	if domain.commands[2].Type != command.Type("action.outcome.apply") {
		t.Fatalf("third command type = %s, want %s", domain.commands[2].Type, "action.outcome.apply")
	}
}

func TestSessionTagTeamFlow_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-tag-team",
		RollSeq:   1,
		Results:   map[string]any{"d20": 18},
		Outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_ACTION.String(),
			"hope_fear":    false,
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	outcomePayload, err := json.Marshal(action.OutcomeApplyPayload{
		RequestID: "req-tag-team",
		RollSeq:   1,
		Targets:   []string{"char-1", "char-2"},
	})
	if err != nil {
		t.Fatalf("encode outcome payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.roll_resolved"),
				Timestamp:   time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-tag-team",
				EntityType:  "roll",
				EntityID:    "req-tag-team",
				PayloadJSON: rollJSON,
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_applied"),
				Timestamp:   time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-tag-team",
				EntityType:  "outcome",
				EntityID:    "req-tag-team",
				PayloadJSON: outcomePayload,
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(context.Background(), "req-tag-team")
	_, err = svc.SessionTagTeamFlow(ctx, &pb.SessionTagTeamFlowRequest{
		CampaignId:          "camp-1",
		SessionId:           "sess-1",
		Difficulty:          10,
		First:               &pb.TagTeamParticipant{CharacterId: "char-1", Trait: "agility"},
		Second:              &pb.TagTeamParticipant{CharacterId: "char-2", Trait: "strength"},
		SelectedCharacterId: "char-1",
	})
	if err != nil {
		t.Fatalf("SessionTagTeamFlow returned error: %v", err)
	}
	if len(domain.commands) != 3 {
		t.Fatalf("expected 3 domain commands, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("action.roll.resolve") {
		t.Fatalf("first command type = %s, want %s", domain.commands[0].Type, "action.roll.resolve")
	}
	if domain.commands[1].Type != command.Type("action.roll.resolve") {
		t.Fatalf("second command type = %s, want %s", domain.commands[1].Type, "action.roll.resolve")
	}
	if domain.commands[2].Type != command.Type("action.outcome.apply") {
		t.Fatalf("third command type = %s, want %s", domain.commands[2].Type, "action.outcome.apply")
	}
}

func TestApplyAttackOutcome_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-atk-outcome-1",
		RollSeq:   1,
		Results:   map[string]any{"d20": 18},
		Outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_ACTION.String(),
			"hope_fear":    true,
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-atk-outcome-1",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-atk-outcome-1",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	ctx := grpcmeta.WithRequestID(
		withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"),
		"req-atk-outcome-1",
	)
	resp, err := svc.ApplyAttackOutcome(ctx, &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
		Targets:   []string{"char-2"},
	})
	if err != nil {
		t.Fatalf("ApplyAttackOutcome returned error: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("expected result in response")
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("expected attacker char-1, got %s", resp.CharacterId)
	}
	if resp.Result == nil {
		t.Fatal("expected result in response")
	}
	if resp.Result.GetOutcome() != pb.Outcome_SUCCESS_WITH_HOPE {
		t.Fatalf("expected outcome SUCCESS_WITH_HOPE, got %s", resp.Result.GetOutcome())
	}
	if !resp.Result.GetSuccess() {
		t.Fatal("expected successful attack outcome")
	}
}

func TestApplyAdversaryAttackOutcome_Success(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-adv-atk-outcome-1",
		RollSeq:   1,
		Results: map[string]any{
			"rolls":        []int{3},
			"roll":         3,
			"modifier":     0,
			"total":        3,
			"advantage":    0,
			"disadvantage": 0,
		},
		Outcome: pb.Outcome_FAILURE_WITH_FEAR.String(),
		SystemData: map[string]any{
			"character_id": "adv-1",
			"adversary_id": "adv-1",
			"roll_kind":    "adversary_roll",
			"roll":         3,
			"modifier":     0,
			"total":        3,
			"advantage":    0,
			"disadvantage": 0,
		},
	}
	rollPayloadJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-adv-atk-outcome-1",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-adv-atk-outcome-1",
		PayloadJSON: rollPayloadJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	ctx := grpcmeta.WithRequestID(
		withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"),
		"req-adv-atk-outcome-1",
	)
	resp, err := svc.ApplyAdversaryAttackOutcome(ctx, &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId:  "sess-1",
		RollSeq:    rollEvent.Seq,
		Targets:    []string{"char-1"},
		Difficulty: 10,
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryAttackOutcome returned error: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("expected result in response")
	}
	if resp.AdversaryId != "adv-1" {
		t.Fatalf("expected adversary adv-1, got %s", resp.AdversaryId)
	}
	if resp.Result == nil {
		t.Fatal("expected result in response")
	}
	if resp.Result.GetSuccess() {
		t.Fatal("expected adversary attack outcome failure")
	}
	if resp.Result.GetRoll() != 3 {
		t.Fatalf("expected roll=3, got %d", resp.Result.GetRoll())
	}
	if resp.Result.GetTotal() != 3 {
		t.Fatalf("expected total=3, got %d", resp.Result.GetTotal())
	}
	if resp.Result.GetDifficulty() != 10 {
		t.Fatalf("expected difficulty=10, got %d", resp.Result.GetDifficulty())
	}
}
