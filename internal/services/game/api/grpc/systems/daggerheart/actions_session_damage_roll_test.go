package daggerheart

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
)

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
			"rolls":                       []int{3, 4},
			"base_total":                  7,
			workflowtransport.KeyModifier: 0,
			"critical_bonus":              0,
			workflowtransport.KeyTotal:    7,
		},
		SystemData: map[string]any{
			workflowtransport.KeyCharacterID: "char-1",
			workflowtransport.KeyRollKind:    "damage_roll",
			workflowtransport.KeyRoll:        7,
			"base_total":                     7,
			workflowtransport.KeyModifier:    0,
			"critical":                       false,
			"critical_bonus":                 0,
			workflowtransport.KeyTotal:       7,
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
				Timestamp:   testTimestamp,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-damage-roll-success",
				EntityType:  "roll",
				EntityID:    "req-damage-roll-success",
				PayloadJSON: rollPayloadJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = serviceDomain
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
			"rolls":                       []int{3, 4},
			"base_total":                  7,
			workflowtransport.KeyModifier: 0,
			"critical_bonus":              0,
			workflowtransport.KeyTotal:    7,
		},
		SystemData: map[string]any{
			workflowtransport.KeyCharacterID: "char-1",
			workflowtransport.KeyRollKind:    "damage_roll",
			workflowtransport.KeyRoll:        7,
			"base_total":                     7,
			workflowtransport.KeyModifier:    0,
			"critical":                       false,
			"critical_bonus":                 0,
			workflowtransport.KeyTotal:       7,
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
				Timestamp:   testTimestamp,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-damage-roll-legacy",
				EntityType:  "roll",
				EntityID:    "req-damage-roll-legacy",
				PayloadJSON: payloadJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = domain

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
	characterID, ok := got.SystemData[workflowtransport.KeyCharacterID].(string)
	if !ok || characterID != "char-1" {
		t.Fatalf("command character id = %v, want %s", got.SystemData[workflowtransport.KeyCharacterID], "char-1")
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
