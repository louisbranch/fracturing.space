package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
)

func TestApplyRest_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyRest(context.Background(), &pb.DaggerheartApplyRestRequest{
		CampaignId: "c1",
	})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestApplyRest_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId: "camp-1",
		Rest: &pb.DaggerheartRestRequest{
			RestType: pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_SHORT,
			Participants: []*pb.DaggerheartRestParticipant{
				{CharacterId: "char-1"},
			},
		},
	})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestApplyRest_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRest_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyRest(context.Background(), &pb.DaggerheartApplyRestRequest{
		CampaignId: "camp-1",
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRest_MissingRest(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId: "camp-1",
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRest_UnspecifiedRestType(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId: "camp-1",
		Rest: &pb.DaggerheartRestRequest{
			RestType: pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_UNSPECIFIED,
		},
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}
