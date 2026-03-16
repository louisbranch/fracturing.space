package sessionflowtransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
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
		LoadAdversary: func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
			return projectionstore.DaggerheartAdversary{AdversaryID: "adv-1", SessionID: "sess-1"}, nil
		},
		LoadAdversaryEntry: func(context.Context, string) (contentstore.DaggerheartAdversaryEntry, error) {
			return contentstore.DaggerheartAdversaryEntry{
				StandardAttack: contentstore.DaggerheartAdversaryAttack{
					DamageDice: []contentstore.DaggerheartDamageDie{{Sides: 8, Count: 1}},
					DamageType: "magic",
				},
			}, nil
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
