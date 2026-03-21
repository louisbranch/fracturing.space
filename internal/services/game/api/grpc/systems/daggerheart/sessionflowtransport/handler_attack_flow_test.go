package sessionflowtransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

func TestHandlerSessionAttackFlowSuccess(t *testing.T) {
	var applyDamageReq *pb.DaggerheartApplyDamageRequest
	handler := NewHandler(Dependencies{
		SessionActionRoll: func(context.Context, *pb.SessionActionRollRequest) (*pb.SessionActionRollResponse, error) {
			return &pb.SessionActionRollResponse{RollSeq: 11}, nil
		},
		SessionDamageRoll: func(context.Context, *pb.SessionDamageRollRequest) (*pb.SessionDamageRollResponse, error) {
			return &pb.SessionDamageRollResponse{RollSeq: 12, Total: 7}, nil
		},
		ApplyRollOutcome: func(ctx context.Context, in *pb.ApplyRollOutcomeRequest) (*pb.ApplyRollOutcomeResponse, error) {
			assertContextIDs(t, ctx, "camp-1", "sess-1")
			if in.GetRollSeq() != 11 {
				t.Fatalf("roll outcome roll_seq = %d, want 11", in.GetRollSeq())
			}
			return &pb.ApplyRollOutcomeResponse{}, nil
		},
		ApplyAttackOutcome: func(ctx context.Context, in *pb.DaggerheartApplyAttackOutcomeRequest) (*pb.DaggerheartApplyAttackOutcomeResponse, error) {
			assertContextIDs(t, ctx, "camp-1", "sess-1")
			if got := in.GetTargets(); len(got) != 1 || got[0] != "char-2" {
				t.Fatalf("targets = %v", got)
			}
			return &pb.DaggerheartApplyAttackOutcomeResponse{
				Result: &pb.DaggerheartAttackOutcomeResult{Success: true, Crit: true},
			}, nil
		},
		ApplyDamage: func(ctx context.Context, in *pb.DaggerheartApplyDamageRequest) (*pb.DaggerheartApplyDamageResponse, error) {
			assertContextIDs(t, ctx, "camp-1", "sess-1")
			applyDamageReq = in
			return &pb.DaggerheartApplyDamageResponse{CharacterId: in.GetCharacterId()}, nil
		},
		LoadCharacterProfile: func(context.Context, string, string) (projectionstore.DaggerheartCharacterProfile, error) {
			return projectionstore.DaggerheartCharacterProfile{}, nil
		},
		LoadCharacterState: func(context.Context, string, string) (projectionstore.DaggerheartCharacterState, error) {
			return projectionstore.DaggerheartCharacterState{}, nil
		},
		LoadSubclass: func(context.Context, string) (contentstore.DaggerheartSubclass, error) {
			return contentstore.DaggerheartSubclass{}, nil
		},
		LoadArmor: func(context.Context, string) (contentstore.DaggerheartArmor, error) {
			return contentstore.DaggerheartArmor{}, nil
		},
	})

	resp, err := handler.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		SceneId:     "scene-1",
		CharacterId: "char-1",
		TargetId:    "char-2",
		AttackProfile: &pb.SessionAttackFlowRequest_StandardAttack{
			StandardAttack: &pb.SessionStandardAttackProfile{
				Trait:       "agility",
				DamageDice:  []*pb.DiceSpec{{Sides: 6, Count: 1}},
				AttackRange: pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_MELEE,
			},
		},
		Damage: &pb.DaggerheartAttackDamageSpec{
			DamageType:         pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
			SourceCharacterIds: []string{"char-1", "char-1"},
		},
	})
	if err != nil {
		t.Fatalf("SessionAttackFlow returned error: %v", err)
	}
	if resp.GetDamageRoll() == nil || resp.GetDamageApplied() == nil {
		t.Fatal("expected damage path to run")
	}
	if applyDamageReq == nil {
		t.Fatal("expected apply damage request")
	}
	if got := applyDamageReq.GetDamage().GetSourceCharacterIds(); len(got) != 1 || got[0] != "char-1" {
		t.Fatalf("source_character_ids = %v", got)
	}
}
