package daggerheart

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
)

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

func TestSessionAdversaryAttackFlow_Success(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-adv-attack-1",
		RollSeq:   1,
		Results: map[string]any{
			"rolls":                       []int{1},
			workflowtransport.KeyRoll:     1,
			workflowtransport.KeyModifier: 0,
			workflowtransport.KeyTotal:    1,
			"advantage":                   0,
			"disadvantage":                0,
		},
		Outcome: pb.Outcome_FAILURE_WITH_HOPE.String(),
		SystemData: map[string]any{
			workflowtransport.KeyCharacterID: "adv-1",
			workflowtransport.KeyAdversaryID: "adv-1",
			workflowtransport.KeyRollKind:    "adversary_roll",
			workflowtransport.KeyRoll:        1,
			workflowtransport.KeyModifier:    0,
			workflowtransport.KeyTotal:       1,
			"advantage":                      0,
			"disadvantage":                   0,
		},
	}
	rollPayloadJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}

	svc.stores.Write.Executor = &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("action.roll_resolved"),
				Timestamp:     testTimestamp,
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
