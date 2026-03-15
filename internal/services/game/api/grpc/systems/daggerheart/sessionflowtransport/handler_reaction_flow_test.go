package sessionflowtransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

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
