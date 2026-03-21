package sessionflowtransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

func TestHandlerSessionAdversaryAttackFlowFocusTargetDisadvantage(t *testing.T) {
	var rollReq *pb.SessionAdversaryAttackRollRequest
	handler := NewHandler(Dependencies{
		SessionAdversaryAttackRoll: func(_ context.Context, in *pb.SessionAdversaryAttackRollRequest) (*pb.SessionAdversaryAttackRollResponse, error) {
			rollReq = in
			return &pb.SessionAdversaryAttackRollResponse{RollSeq: 31}, nil
		},
		ApplyAdversaryAttackOutcome: func(context.Context, *pb.DaggerheartApplyAdversaryAttackOutcomeRequest) (*pb.DaggerheartApplyAdversaryAttackOutcomeResponse, error) {
			return &pb.DaggerheartApplyAdversaryAttackOutcomeResponse{
				Result: &pb.DaggerheartAdversaryAttackOutcomeResult{Success: true, Crit: true},
			}, nil
		},
		SessionDamageRoll: func(context.Context, *pb.SessionDamageRollRequest) (*pb.SessionDamageRollResponse, error) {
			return &pb.SessionDamageRollResponse{RollSeq: 32, Total: 5}, nil
		},
		ApplyDamage: func(context.Context, *pb.DaggerheartApplyDamageRequest) (*pb.DaggerheartApplyDamageResponse, error) {
			return &pb.DaggerheartApplyDamageResponse{}, nil
		},
		LoadAdversary: func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
			return projectionstore.DaggerheartAdversary{
				AdversaryID: "adv-1",
				SessionID:   "sess-1",
				FeatureStates: []projectionstore.DaggerheartAdversaryFeatureState{
					{FeatureID: "feat-box-in", Status: "active", FocusedTargetID: "char-focused"},
				},
			}, nil
		},
		LoadAdversaryEntry: func(context.Context, string) (contentstore.DaggerheartAdversaryEntry, error) {
			return contentstore.DaggerheartAdversaryEntry{
				Features: []contentstore.DaggerheartAdversaryFeature{
					{ID: "feat-box-in", Name: "Box In", Kind: "passive"},
				},
				StandardAttack: contentstore.DaggerheartAdversaryAttack{
					DamageDice: []contentstore.DaggerheartDamageDie{{Sides: 6, Count: 1}},
					DamageType: "physical",
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

	// Attack the focused target — should gain +1 disadvantage.
	resp, err := handler.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
		TargetId:    "char-focused",
		FeatureId:   "feat-box-in",
		Difficulty:  12,
		Damage: &pb.DaggerheartAttackDamageSpec{
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if err != nil {
		t.Fatalf("SessionAdversaryAttackFlow returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if rollReq == nil {
		t.Fatal("expected attack roll to be called")
	}
	if rollReq.GetDisadvantage() != 1 {
		t.Fatalf("disadvantage = %d, want 1 (focus target disadvantage)", rollReq.GetDisadvantage())
	}

	// Attack a non-focused target — should have no extra disadvantage.
	rollReq = nil
	resp, err = handler.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
		TargetId:    "char-other",
		FeatureId:   "feat-box-in",
		Difficulty:  12,
		Damage: &pb.DaggerheartAttackDamageSpec{
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if err != nil {
		t.Fatalf("SessionAdversaryAttackFlow (non-focused) returned error: %v", err)
	}
	if resp == nil || rollReq == nil {
		t.Fatal("expected non-nil response and roll request")
	}
	if rollReq.GetDisadvantage() != 0 {
		t.Fatalf("disadvantage = %d, want 0 (not the focused target)", rollReq.GetDisadvantage())
	}
}

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
