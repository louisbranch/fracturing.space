package sessionflowtransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
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
	})

	resp, err := handler.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		SceneId:     "scene-1",
		CharacterId: "char-1",
		Trait:       "agility",
		TargetId:    "char-2",
		Damage: &pb.DaggerheartAttackDamageSpec{
			DamageType:         pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
			SourceCharacterIds: []string{"char-1", "char-1"},
		},
		DamageDice: []*pb.DiceSpec{{Sides: 6, Count: 1}},
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

func TestHandlerSessionReactionFlowForwardsReactionParameters(t *testing.T) {
	var actionRollReq *pb.SessionActionRollRequest
	handler := NewHandler(Dependencies{
		SessionActionRoll: func(_ context.Context, in *pb.SessionActionRollRequest) (*pb.SessionActionRollResponse, error) {
			cloned := *in
			actionRollReq = &cloned
			return &pb.SessionActionRollResponse{RollSeq: 21}, nil
		},
		ApplyRollOutcome: func(context.Context, *pb.ApplyRollOutcomeRequest) (*pb.ApplyRollOutcomeResponse, error) {
			return &pb.ApplyRollOutcomeResponse{}, nil
		},
		ApplyReactionOutcome: func(context.Context, *pb.DaggerheartApplyReactionOutcomeRequest) (*pb.DaggerheartApplyReactionOutcomeResponse, error) {
			return &pb.DaggerheartApplyReactionOutcomeResponse{}, nil
		},
	})
	_, err := handler.SessionReactionFlow(context.Background(), &pb.SessionReactionFlowRequest{
		CampaignId:   "camp-1",
		SessionId:    "sess-1",
		SceneId:      "scene-1",
		CharacterId:  "char-1",
		Trait:        "instinct",
		Advantage:    1,
		Disadvantage: 2,
	})
	if err != nil {
		t.Fatalf("SessionReactionFlow returned error: %v", err)
	}
	if actionRollReq == nil {
		t.Fatal("expected action roll request")
	}
}

func TestHandlerSessionGroupActionFlowBuildsLeaderSupportModifier(t *testing.T) {
	call := 0
	handler := NewHandler(Dependencies{
		SessionActionRoll: func(_ context.Context, in *pb.SessionActionRollRequest) (*pb.SessionActionRollResponse, error) {
			call++
			if call == 1 {
				return &pb.SessionActionRollResponse{RollSeq: 1, Success: true}, nil
			}
			if call == 2 {
				return &pb.SessionActionRollResponse{RollSeq: 2, Success: false}, nil
			}
			if got := in.GetModifiers(); len(got) != 0 {
				t.Fatalf("leader modifiers = %+v", got)
			}
			return &pb.SessionActionRollResponse{RollSeq: 3}, nil
		},
		ApplyRollOutcome: func(context.Context, *pb.ApplyRollOutcomeRequest) (*pb.ApplyRollOutcomeResponse, error) {
			return &pb.ApplyRollOutcomeResponse{}, nil
		},
	})

	resp, err := handler.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId:        "camp-1",
		SessionId:         "sess-1",
		SceneId:           "scene-1",
		LeaderCharacterId: "leader-1",
		LeaderTrait:       "presence",
		Difficulty:        12,
		Supporters: []*pb.GroupActionSupporter{
			{CharacterId: "support-1", Trait: "instinct"},
			{CharacterId: "support-2", Trait: "agility"},
		},
	})
	if err != nil {
		t.Fatalf("SessionGroupActionFlow returned error: %v", err)
	}
	if got := resp.GetSupportSuccesses(); got != 1 {
		t.Fatalf("support_successes = %d, want 1", got)
	}
	if got := resp.GetSupportFailures(); got != 1 {
		t.Fatalf("support_failures = %d, want 1", got)
	}
	if got := resp.GetSupportModifier(); got != 0 {
		t.Fatalf("support_modifier = %d, want 0", got)
	}
}

func TestHandlerSessionTagTeamFlowUsesSelectedRollAndTargets(t *testing.T) {
	var outcomeReq *pb.ApplyRollOutcomeRequest
	call := 0
	handler := NewHandler(Dependencies{
		SessionActionRoll: func(context.Context, *pb.SessionActionRollRequest) (*pb.SessionActionRollResponse, error) {
			call++
			return &pb.SessionActionRollResponse{RollSeq: uint64(call)}, nil
		},
		ApplyRollOutcome: func(_ context.Context, in *pb.ApplyRollOutcomeRequest) (*pb.ApplyRollOutcomeResponse, error) {
			outcomeReq = &pb.ApplyRollOutcomeRequest{
				SessionId: in.GetSessionId(),
				SceneId:   in.GetSceneId(),
				RollSeq:   in.GetRollSeq(),
				Targets:   append([]string(nil), in.GetTargets()...),
			}
			return &pb.ApplyRollOutcomeResponse{}, nil
		},
	})

	resp, err := handler.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId:          "camp-1",
		SessionId:           "sess-1",
		SceneId:             "scene-1",
		Difficulty:          10,
		SelectedCharacterId: "char-2",
		First:               &pb.TagTeamParticipant{CharacterId: "char-1", Trait: "agility"},
		Second:              &pb.TagTeamParticipant{CharacterId: "char-2", Trait: "presence"},
	})
	if err != nil {
		t.Fatalf("SessionTagTeamFlow returned error: %v", err)
	}
	if outcomeReq == nil {
		t.Fatal("expected outcome request")
	}
	if outcomeReq.GetRollSeq() != 2 {
		t.Fatalf("selected roll_seq = %d, want 2", outcomeReq.GetRollSeq())
	}
	if got := outcomeReq.GetTargets(); len(got) != 2 || got[0] != "char-1" || got[1] != "char-2" {
		t.Fatalf("targets = %v", got)
	}
	if resp.GetSelectedRollSeq() != 2 {
		t.Fatalf("selected_roll_seq = %d, want 2", resp.GetSelectedRollSeq())
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

func assertContextIDs(t *testing.T, ctx context.Context, campaignID, sessionID string) {
	t.Helper()
	if got := grpcmeta.CampaignIDFromContext(ctx); got != campaignID {
		t.Fatalf("campaign_id metadata = %q, want %q", got, campaignID)
	}
	if got := grpcmeta.SessionIDFromContext(ctx); got != sessionID {
		t.Fatalf("session_id metadata = %q, want %q", got, sessionID)
	}
}
