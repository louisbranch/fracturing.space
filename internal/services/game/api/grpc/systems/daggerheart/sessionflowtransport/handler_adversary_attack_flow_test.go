package sessionflowtransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func TestHandlerSessionAdversaryAttackFlowAddsAdversarySourceCharacter(t *testing.T) {
	var applyDamageReq *pb.DaggerheartApplyDamageRequest
	handler := NewHandler(Dependencies{
		SessionAdversaryAttackRoll: func(context.Context, *pb.SessionAdversaryAttackRollRequest) (*pb.SessionAdversaryAttackRollResponse, error) {
			return &pb.SessionAdversaryAttackRollResponse{RollSeq: 31}, nil
		},
		ApplyAdversaryAttackOutcome: func(context.Context, *pb.DaggerheartApplyAdversaryAttackOutcomeRequest) (*pb.DaggerheartApplyAdversaryAttackOutcomeResponse, error) {
			return &pb.DaggerheartApplyAdversaryAttackOutcomeResponse{
				Result: &pb.DaggerheartAdversaryAttackOutcomeResult{Success: true, Crit: true},
			}, nil
		},
		SessionDamageRoll: func(context.Context, *pb.SessionDamageRollRequest) (*pb.SessionDamageRollResponse, error) {
			return &pb.SessionDamageRollResponse{RollSeq: 32, Total: 9}, nil
		},
		ApplyDamage: func(_ context.Context, in *pb.DaggerheartApplyDamageRequest) (*pb.DaggerheartApplyDamageResponse, error) {
			applyDamageReq = in
			return &pb.DaggerheartApplyDamageResponse{}, nil
		},
	})

	resp, err := handler.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
		TargetId:    "char-1",
		Difficulty:  14,
		Damage: &pb.DaggerheartAttackDamageSpec{
			DamageType:         pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC,
			SourceCharacterIds: []string{"support-1", "support-1"},
		},
		DamageDice: []*pb.DiceSpec{{Sides: 8, Count: 1}},
	})
	if err != nil {
		t.Fatalf("SessionAdversaryAttackFlow returned error: %v", err)
	}
	if resp.GetDamageRoll() == nil || applyDamageReq == nil {
		t.Fatal("expected damage path to run")
	}
	if got := applyDamageReq.GetDamage().GetSourceCharacterIds(); len(got) != 2 || got[0] != "support-1" || got[1] != "adv-1" {
		t.Fatalf("source_character_ids = %v", got)
	}
}
