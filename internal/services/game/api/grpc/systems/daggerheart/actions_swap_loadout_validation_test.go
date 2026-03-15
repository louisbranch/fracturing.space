package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc/codes"
)

func TestSwapLoadout_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SwapLoadout(context.Background(), &pb.DaggerheartSwapLoadoutRequest{
		CampaignId: "c1", CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSwapLoadout_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.SwapLoadout(ctx, &pb.DaggerheartSwapLoadoutRequest{
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSwapLoadout_MissingCharacterId(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.SwapLoadout(ctx, &pb.DaggerheartSwapLoadoutRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSwapLoadout_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SwapLoadout(context.Background(), &pb.DaggerheartSwapLoadoutRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSwapLoadout_MissingSwap(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.SwapLoadout(ctx, &pb.DaggerheartSwapLoadoutRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSwapLoadout_MissingCardId(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.SwapLoadout(ctx, &pb.DaggerheartSwapLoadoutRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Swap:        &pb.DaggerheartLoadoutSwapRequest{},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSwapLoadout_NegativeRecallCost(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.SwapLoadout(ctx, &pb.DaggerheartSwapLoadoutRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Swap: &pb.DaggerheartLoadoutSwapRequest{
			CardId:     "card-1",
			RecallCost: -1,
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}
