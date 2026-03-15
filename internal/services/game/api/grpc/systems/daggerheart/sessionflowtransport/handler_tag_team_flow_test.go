package sessionflowtransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

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
