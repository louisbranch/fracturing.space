package app

import (
	"context"
	"io"
	"sync"
	"testing"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestConsumeCampaignProjectionUpdatesInvalidatesOnMatchingProjectionUpdate(t *testing.T) {
	t.Parallel()

	eventClient := &eventClientStub{
		listResp: &gamev1.ListEventsResponse{
			Events: []*gamev1.Event{{Seq: 4}},
		},
		stream: &campaignUpdateStreamStub{
			updates: []*gamev1.CampaignUpdate{{CampaignId: "camp-1", Seq: 5, Update: &gamev1.CampaignUpdate_ProjectionApplied{
				ProjectionApplied: &gamev1.ProjectionApplied{Scopes: []string{"campaign_sessions"}},
			}}},
		},
	}
	var (
		mu          sync.Mutex
		invalidated []string
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go consumeCampaignProjectionUpdates(ctx, eventClient, "camp-1", func(campaignID string) {
		mu.Lock()
		defer mu.Unlock()
		invalidated = append(invalidated, campaignID)
		cancel()
	})

	<-ctx.Done()

	if eventClient.subscribeReq == nil {
		t.Fatalf("expected SubscribeCampaignUpdates call")
	}
	if eventClient.subscribeReq.GetAfterSeq() != 4 {
		t.Fatalf("AfterSeq = %d, want 4", eventClient.subscribeReq.GetAfterSeq())
	}
	if got := eventClient.subscribeReq.GetProjectionScopes(); len(got) != len(dashboardProjectionScopes) {
		t.Fatalf("ProjectionScopes len = %d, want %d", len(got), len(dashboardProjectionScopes))
	}
	mu.Lock()
	defer mu.Unlock()
	if len(invalidated) != 1 || invalidated[0] != "camp-1" {
		t.Fatalf("invalidated = %v, want [camp-1]", invalidated)
	}
}

type eventClientStub struct {
	listResp     *gamev1.ListEventsResponse
	listErr      error
	stream       grpc.ServerStreamingClient[gamev1.CampaignUpdate]
	subscribeReq *gamev1.SubscribeCampaignUpdatesRequest
}

func (s *eventClientStub) AppendEvent(context.Context, *gamev1.AppendEventRequest, ...grpc.CallOption) (*gamev1.AppendEventResponse, error) {
	return nil, nil
}

func (s *eventClientStub) ListEvents(_ context.Context, req *gamev1.ListEventsRequest, _ ...grpc.CallOption) (*gamev1.ListEventsResponse, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.listResp, nil
}

func (s *eventClientStub) ListTimelineEntries(context.Context, *gamev1.ListTimelineEntriesRequest, ...grpc.CallOption) (*gamev1.ListTimelineEntriesResponse, error) {
	return nil, nil
}

func (s *eventClientStub) SubscribeCampaignUpdates(_ context.Context, req *gamev1.SubscribeCampaignUpdatesRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[gamev1.CampaignUpdate], error) {
	s.subscribeReq = req
	return s.stream, nil
}

type campaignUpdateStreamStub struct {
	updates []*gamev1.CampaignUpdate
	index   int
}

func (s *campaignUpdateStreamStub) Header() (metadata.MD, error) { return nil, nil }
func (s *campaignUpdateStreamStub) Trailer() metadata.MD         { return nil }
func (s *campaignUpdateStreamStub) CloseSend() error             { return nil }
func (s *campaignUpdateStreamStub) Context() context.Context     { return context.Background() }
func (s *campaignUpdateStreamStub) SendMsg(any) error            { return nil }
func (s *campaignUpdateStreamStub) RecvMsg(any) error            { return nil }

func (s *campaignUpdateStreamStub) Recv() (*gamev1.CampaignUpdate, error) {
	if s.index >= len(s.updates) {
		return nil, io.EOF
	}
	update := s.updates[s.index]
	s.index++
	return update, nil
}
