package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc/codes"
)

func TestApplyGmMove_FearSpentExceedsAvailable(t *testing.T) {
	svc := newActionTestService()
	// Snapshot has 0 fear.
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Move: "test_move", FearSpent: 10,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyGmMove_CampaignNotFound(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "nonexistent", SessionId: "sess-1", Move: "test",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyGmMove_SessionNotFound(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1", SessionId: "nonexistent", Move: "test",
	})
	assertStatusCode(t, err, codes.Internal)
}
