package daggerheart

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
)

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

func TestSessionAttackFlow_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := testTimestamp

	rollPayload := action.RollResolvePayload{
		RequestID: "req-attack-1",
		RollSeq:   1,
		Results:   map[string]any{"d20": 8},
		Outcome:   pb.Outcome_FAILURE_WITH_FEAR.String(),
		SystemData: map[string]any{
			workflowtransport.KeyCharacterID: "char-1",
			workflowtransport.KeyRollKind:    pb.RollKind_ROLL_KIND_ACTION.String(),
			workflowtransport.KeyHopeFear:    false,
			"gm_move":                        false,
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

	svc.stores.Write.Executor = &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
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
	if resp.AttackOutcome.GetResult() == nil {
		t.Fatal("expected attack outcome result")
	}
}
