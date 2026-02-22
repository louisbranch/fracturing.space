package daggerheart

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
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
