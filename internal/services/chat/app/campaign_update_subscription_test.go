package server

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type testCampaignUpdateEventClient struct {
	headSeq uint64
	listErr error

	subscribeFn    func(context.Context, *gamev1.SubscribeCampaignUpdatesRequest) (grpc.ServerStreamingClient[gamev1.CampaignUpdate], error)
	subscribeCalls int
	subscribeReqs  []*gamev1.SubscribeCampaignUpdatesRequest
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

func (c *testCampaignUpdateEventClient) SubscribeCampaignUpdates(ctx context.Context, req *gamev1.SubscribeCampaignUpdatesRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[gamev1.CampaignUpdate], error) {
	c.subscribeCalls++
	c.subscribeReqs = append(c.subscribeReqs, req)
	if c.subscribeFn != nil {
		return c.subscribeFn(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

type testCampaignUpdateStream struct {
	ctx     context.Context
	updates []*gamev1.CampaignUpdate
	recvErr error
	index   int
}

func (s *testCampaignUpdateStream) Recv() (*gamev1.CampaignUpdate, error) {
	if s.index < len(s.updates) {
		update := s.updates[s.index]
		s.index++
		return update, nil
	}
	if s.recvErr != nil {
		return nil, s.recvErr
	}
	if s.ctx != nil {
		<-s.ctx.Done()
		return nil, s.ctx.Err()
	}
	return nil, io.EOF
}

func (s *testCampaignUpdateStream) Header() (metadata.MD, error) { return metadata.MD{}, nil }
func (s *testCampaignUpdateStream) Trailer() metadata.MD         { return metadata.MD{} }
func (s *testCampaignUpdateStream) CloseSend() error             { return nil }
func (s *testCampaignUpdateStream) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
func (s *testCampaignUpdateStream) SendMsg(any) error { return nil }
func (s *testCampaignUpdateStream) RecvMsg(any) error { return nil }

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

func TestConsumeCampaignEventCommittedUpdatesEmitsEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := &testCampaignUpdateEventClient{headSeq: 17}
	client.subscribeFn = func(ctx context.Context, req *gamev1.SubscribeCampaignUpdatesRequest) (grpc.ServerStreamingClient[gamev1.CampaignUpdate], error) {
		return &testCampaignUpdateStream{
			ctx: ctx,
			updates: []*gamev1.CampaignUpdate{
				{Seq: 18, EventType: "campaign.ai_bound"},
			},
			recvErr: io.EOF,
		}, nil
	}

	emitted := make([]string, 0, 1)
	consumeCampaignEventCommittedUpdates(ctx, client, "camp-1", func(campaignID, eventType string) {
		emitted = append(emitted, campaignID+":"+eventType)
		cancel()
	})

	if len(emitted) != 1 || emitted[0] != "camp-1:campaign.ai_bound" {
		t.Fatalf("emitted = %v, want [camp-1:campaign.ai_bound]", emitted)
	}
	if client.subscribeCalls != 1 {
		t.Fatalf("subscribe calls = %d, want %d", client.subscribeCalls, 1)
	}
	if len(client.subscribeReqs) != 1 || client.subscribeReqs[0].GetAfterSeq() != 17 {
		t.Fatalf("subscribe after seq = %d, want %d", client.subscribeReqs[0].GetAfterSeq(), 17)
	}
}

func TestConsumeCampaignEventCommittedUpdatesRetriesOnSubscribeError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := &testCampaignUpdateEventClient{}
	client.subscribeFn = func(ctx context.Context, req *gamev1.SubscribeCampaignUpdatesRequest) (grpc.ServerStreamingClient[gamev1.CampaignUpdate], error) {
		if client.subscribeCalls == 1 {
			return nil, errors.New("temporary")
		}
		cancel()
		return &testCampaignUpdateStream{ctx: ctx, recvErr: io.EOF}, nil
	}

	consumeCampaignEventCommittedUpdates(ctx, client, "camp-1", nil)

	if client.subscribeCalls < 2 {
		t.Fatalf("subscribe calls = %d, want at least 2", client.subscribeCalls)
	}
}

func TestCampaignEventSubscriptionWorkerLifecycle(t *testing.T) {
	if ensure, release, stop, done := startCampaignEventCommittedSubscriptionWorker(nil, nil); ensure != nil || release != nil || stop != nil || done != nil {
		t.Fatal("expected nil worker hooks when event client is nil")
	}

	ctx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()
	client := &testCampaignUpdateEventClient{}
	client.subscribeFn = func(ctx context.Context, req *gamev1.SubscribeCampaignUpdatesRequest) (grpc.ServerStreamingClient[gamev1.CampaignUpdate], error) {
		return &testCampaignUpdateStream{ctx: ctx}, nil
	}
	worker := &campaignEventCommittedSubscriptionWorker{
		ctx:         ctx,
		eventClient: client,
		subscribers: map[string]context.CancelFunc{},
	}

	worker.ensureCampaignSubscription("  camp-1  ")
	worker.ensureCampaignSubscription("camp-1")
	time.Sleep(20 * time.Millisecond)
	if client.subscribeCalls != 1 {
		t.Fatalf("subscribe calls = %d, want %d", client.subscribeCalls, 1)
	}
	if len(worker.subscribers) != 1 {
		t.Fatalf("subscriber count = %d, want %d", len(worker.subscribers), 1)
	}

	worker.releaseCampaignSubscription("camp-1")
	if len(worker.subscribers) != 0 {
		t.Fatalf("subscriber count = %d, want 0", len(worker.subscribers))
	}
	worker.wg.Wait()
}

func TestWaitCampaignEventSubscriptionRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if waitCampaignEventSubscriptionRetry(ctx, 50*time.Millisecond) {
		t.Fatal("expected canceled context to stop retry wait")
	}

	if !waitCampaignEventSubscriptionRetry(context.Background(), time.Millisecond) {
		t.Fatal("expected retry wait to return true when timer elapses")
	}
}
