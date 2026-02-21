package server

import (
	"context"
	"testing"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type testCampaignUpdateEventClient struct {
	headSeq uint64
	listErr error
}

func (c *testCampaignUpdateEventClient) AppendEvent(context.Context, *gamev1.AppendEventRequest, ...grpc.CallOption) (*gamev1.AppendEventResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (c *testCampaignUpdateEventClient) ListEvents(_ context.Context, req *gamev1.ListEventsRequest, _ ...grpc.CallOption) (*gamev1.ListEventsResponse, error) {
	if c.listErr != nil {
		return nil, c.listErr
	}
	if c.headSeq == 0 {
		return &gamev1.ListEventsResponse{}, nil
	}
	return &gamev1.ListEventsResponse{
		Events: []*gamev1.Event{
			{
				CampaignId: req.GetCampaignId(),
				Seq:        c.headSeq,
			},
		},
	}, nil
}

func (c *testCampaignUpdateEventClient) ListTimelineEntries(context.Context, *gamev1.ListTimelineEntriesRequest, ...grpc.CallOption) (*gamev1.ListTimelineEntriesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (c *testCampaignUpdateEventClient) SubscribeCampaignUpdates(context.Context, *gamev1.SubscribeCampaignUpdatesRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[gamev1.CampaignUpdate], error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func TestCampaignEventCommittedInitialAfterSeqUsesHead(t *testing.T) {
	client := &testCampaignUpdateEventClient{headSeq: 17}

	afterSeq := campaignEventCommittedInitialAfterSeq(context.Background(), client, "camp-1")
	if afterSeq != 17 {
		t.Fatalf("after seq = %d, want %d", afterSeq, 17)
	}
}

func TestCampaignEventCommittedInitialAfterSeqReturnsZeroOnListError(t *testing.T) {
	client := &testCampaignUpdateEventClient{listErr: status.Error(codes.Internal, "boom")}

	afterSeq := campaignEventCommittedInitialAfterSeq(context.Background(), client, "camp-1")
	if afterSeq != 0 {
		t.Fatalf("after seq = %d, want %d", afterSeq, 0)
	}
}
