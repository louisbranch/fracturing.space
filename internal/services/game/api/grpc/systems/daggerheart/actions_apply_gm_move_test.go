package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
)

func directAdditionalMoveRequest(campaignID, sessionID string, fearSpent int32) *pb.DaggerheartApplyGmMoveRequest {
	return &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
		FearSpent:  fearSpent,
		SpendTarget: &pb.DaggerheartApplyGmMoveRequest_DirectMove{
			DirectMove: &pb.DaggerheartDirectGmMoveTarget{
				Kind:  pb.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_ADDITIONAL_MOVE,
				Shape: pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_SHIFT_ENVIRONMENT,
			},
		},
	}
}

func TestApplyGmMove_FearSpentExceedsAvailable(t *testing.T) {
	svc := newActionTestService()
	// Snapshot has 0 fear.
	_, err := svc.ApplyGmMove(context.Background(), directAdditionalMoveRequest("camp-1", "sess-1", 10))
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestApplyGmMove_CampaignNotFound(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyGmMove(context.Background(), directAdditionalMoveRequest("nonexistent", "sess-1", 1))
	grpcassert.StatusCode(t, err, codes.NotFound)
}

func TestApplyGmMove_SessionNotFound(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyGmMove(context.Background(), directAdditionalMoveRequest("camp-1", "nonexistent", 1))
	grpcassert.StatusCode(t, err, codes.NotFound)
}
