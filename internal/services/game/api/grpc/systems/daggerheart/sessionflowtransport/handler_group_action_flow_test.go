package sessionflowtransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

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
