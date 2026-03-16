package sessionflowtransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
)

func TestSessionAttackFlowForwardsNemesisSwapToOutcomeHandlers(t *testing.T) {
	var (
		rollOutcomeReq   *pb.ApplyRollOutcomeRequest
		attackOutcomeReq *pb.DaggerheartApplyAttackOutcomeRequest
	)
	handler := NewHandler(Dependencies{
		SessionActionRoll: func(context.Context, *pb.SessionActionRollRequest) (*pb.SessionActionRollResponse, error) {
			return &pb.SessionActionRollResponse{RollSeq: 11}, nil
		},
		SessionDamageRoll: func(context.Context, *pb.SessionDamageRollRequest) (*pb.SessionDamageRollResponse, error) {
			return &pb.SessionDamageRollResponse{}, nil
		},
		ApplyRollOutcome: func(_ context.Context, in *pb.ApplyRollOutcomeRequest) (*pb.ApplyRollOutcomeResponse, error) {
			rollOutcomeReq = in
			return &pb.ApplyRollOutcomeResponse{}, nil
		},
		ApplyAttackOutcome: func(_ context.Context, in *pb.DaggerheartApplyAttackOutcomeRequest) (*pb.DaggerheartApplyAttackOutcomeResponse, error) {
			attackOutcomeReq = in
			return &pb.DaggerheartApplyAttackOutcomeResponse{
				Result: &pb.DaggerheartAttackOutcomeResult{Success: false},
			}, nil
		},
		ApplyDamage: func(context.Context, *pb.DaggerheartApplyDamageRequest) (*pb.DaggerheartApplyDamageResponse, error) {
			return &pb.DaggerheartApplyDamageResponse{}, nil
		},
		LoadCharacterProfile: func(context.Context, string, string) (projectionstore.DaggerheartCharacterProfile, error) {
			return projectionstore.DaggerheartCharacterProfile{}, nil
		},
		LoadCharacterState: func(context.Context, string, string) (projectionstore.DaggerheartCharacterState, error) {
			return projectionstore.DaggerheartCharacterState{
				SubclassState: &projectionstore.DaggerheartSubclassState{
					NemesisTargetID: "target-1",
				},
			}, nil
		},
		LoadSubclass: func(context.Context, string) (contentstore.DaggerheartSubclass, error) {
			return contentstore.DaggerheartSubclass{}, nil
		},
		LoadArmor: func(context.Context, string) (contentstore.DaggerheartArmor, error) {
			return contentstore.DaggerheartArmor{}, nil
		},
	})

	_, err := handler.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		SceneId:     "scene-1",
		CharacterId: "char-1",
		TargetId:    "target-1",
		Damage: &pb.DaggerheartAttackDamageSpec{
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
		AttackProfile: &pb.SessionAttackFlowRequest_StandardAttack{
			StandardAttack: &pb.SessionStandardAttackProfile{
				Trait:       "agility",
				AttackRange: pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_MELEE,
				DamageDice:  []*pb.DiceSpec{{Sides: 6, Count: 1}},
			},
		},
	})
	if err != nil {
		t.Fatalf("SessionAttackFlow returned error: %v", err)
	}
	if rollOutcomeReq == nil || !rollOutcomeReq.GetSwapHopeFear() {
		t.Fatalf("roll outcome request = %+v, want swap_hope_fear=true", rollOutcomeReq)
	}
	if attackOutcomeReq == nil || !attackOutcomeReq.GetSwapHopeFear() {
		t.Fatalf("attack outcome request = %+v, want swap_hope_fear=true", attackOutcomeReq)
	}
}
