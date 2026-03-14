package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type testInvocationClient struct {
	submitFn       func(context.Context, *aiv1.SubmitCampaignTurnRequest) (*aiv1.SubmitCampaignTurnResponse, error)
	submitCalls    int
	submitReqs     []*aiv1.SubmitCampaignTurnRequest
	subscribeFn    func(context.Context, *aiv1.SubscribeCampaignTurnEventsRequest) (grpc.ServerStreamingClient[aiv1.CampaignTurnEvent], error)
	subscribeCalls int
	subscribeReqs  []*aiv1.SubscribeCampaignTurnEventsRequest
}

func (c *testInvocationClient) InvokeAgent(context.Context, *aiv1.InvokeAgentRequest, ...grpc.CallOption) (*aiv1.InvokeAgentResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (c *testInvocationClient) SubmitCampaignTurn(ctx context.Context, req *aiv1.SubmitCampaignTurnRequest, _ ...grpc.CallOption) (*aiv1.SubmitCampaignTurnResponse, error) {
	c.submitCalls++
	c.submitReqs = append(c.submitReqs, req)
	if c.submitFn != nil {
		return c.submitFn(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (c *testInvocationClient) SubscribeCampaignTurnEvents(ctx context.Context, req *aiv1.SubscribeCampaignTurnEventsRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[aiv1.CampaignTurnEvent], error) {
	c.subscribeCalls++
	c.subscribeReqs = append(c.subscribeReqs, req)
	if c.subscribeFn != nil {
		return c.subscribeFn(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

type testCampaignTurnStream struct {
	ctx     context.Context
	updates []*aiv1.CampaignTurnEvent
	recvErr error
	onDrain func()
	index   int
}

func (s *testCampaignTurnStream) Recv() (*aiv1.CampaignTurnEvent, error) {
	if s.index < len(s.updates) {
		update := s.updates[s.index]
		s.index++
		return update, nil
	}
	if s.onDrain != nil {
		s.onDrain()
		s.onDrain = nil
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

func (s *testCampaignTurnStream) Header() (metadata.MD, error) { return metadata.MD{}, nil }
func (s *testCampaignTurnStream) Trailer() metadata.MD         { return metadata.MD{} }
func (s *testCampaignTurnStream) CloseSend() error             { return nil }
func (s *testCampaignTurnStream) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
func (s *testCampaignTurnStream) SendMsg(any) error { return nil }
func (s *testCampaignTurnStream) RecvMsg(any) error { return nil }

func TestStartCampaignAITurnSubscriptionWorkerRequiresDependencies(t *testing.T) {
	t.Parallel()

	if ensure, release, stop, done := startCampaignAITurnSubscriptionWorker(nil, &testInvocationClient{}, newRoomHub()); ensure != nil || release != nil || stop != nil || done != nil {
		t.Fatal("expected nil worker hooks when context is nil")
	}
	if ensure, release, stop, done := startCampaignAITurnSubscriptionWorker(context.Background(), nil, newRoomHub()); ensure != nil || release != nil || stop != nil || done != nil {
		t.Fatal("expected nil worker hooks when invocation client is nil")
	}
	if ensure, release, stop, done := startCampaignAITurnSubscriptionWorker(context.Background(), &testInvocationClient{}, nil); ensure != nil || release != nil || stop != nil || done != nil {
		t.Fatal("expected nil worker hooks when room hub is nil")
	}
}

func TestCampaignAITurnSubscriptionWorkerLifecycle(t *testing.T) {
	t.Parallel()

	ctx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	client := &testInvocationClient{}
	client.subscribeFn = func(ctx context.Context, req *aiv1.SubscribeCampaignTurnEventsRequest) (grpc.ServerStreamingClient[aiv1.CampaignTurnEvent], error) {
		return &testCampaignTurnStream{ctx: ctx}, nil
	}
	worker := &campaignAITurnSubscriptionWorker{
		ctx:              ctx,
		invocationClient: client,
		roomHub:          newRoomHub(),
		subscribers:      map[string]context.CancelFunc{},
	}
	room := worker.roomHub.room("camp-1")
	room.setSessionID("session-1")
	room.setAIBinding("AI", "agent-1")
	room.setAISessionGrant("grant-token", 1, time.Now().UTC().Add(time.Minute))

	worker.ensureCampaignSubscription(" camp-1 ", "", "")
	worker.ensureCampaignSubscription("camp-1", "", "")
	waitForCampaignTurnSubscribeCalls(t, client, 1, 200*time.Millisecond)
	if len(worker.subscribers) != 1 {
		t.Fatalf("subscriber count = %d, want %d", len(worker.subscribers), 1)
	}

	worker.releaseCampaignSubscription("camp-1")
	if len(worker.subscribers) != 0 {
		t.Fatalf("subscriber count = %d, want 0", len(worker.subscribers))
	}
	worker.wg.Wait()
}

func TestConsumeCampaignAITurnUpdatesPublishesVisibleMessages(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := newRoomHub()
	room := hub.room("camp-1")
	room.setSessionID("session-1")
	room.setAIBinding("AI", "agent-1")
	room.setAISessionGrant("grant-token", 2, time.Now().UTC().Add(time.Minute))

	buf := &bytes.Buffer{}
	peer := newWSPeer(json.NewEncoder(buf))
	room.join(newWSSession("user-1", peer), []string{chatDefaultStreamID("camp-1")})

	client := &testInvocationClient{}
	client.subscribeFn = func(ctx context.Context, req *aiv1.SubscribeCampaignTurnEventsRequest) (grpc.ServerStreamingClient[aiv1.CampaignTurnEvent], error) {
		return &testCampaignTurnStream{
			ctx: ctx,
			updates: []*aiv1.CampaignTurnEvent{
				nil,
				{SequenceId: 1, ParticipantVisible: false, Content: "hidden", SessionId: "session-1"},
				{SequenceId: 2, ParticipantVisible: true, Content: "   ", SessionId: "session-1"},
				{SequenceId: 3, ParticipantVisible: true, Content: "AI says hi", SessionId: "session-1", CorrelationMessageId: "corr-1"},
				{SequenceId: 4, ParticipantVisible: true, Content: "AI says hi", SessionId: "session-1", CorrelationMessageId: "corr-1"},
			},
			recvErr: io.EOF,
			onDrain: cancel,
		}, nil
	}

	consumeCampaignAITurnUpdates(ctx, client, hub, "camp-1")

	if client.subscribeCalls != 1 {
		t.Fatalf("subscribe calls = %d, want %d", client.subscribeCalls, 1)
	}
	if len(client.subscribeReqs) != 1 {
		t.Fatalf("subscribe req count = %d, want %d", len(client.subscribeReqs), 1)
	}
	if req := client.subscribeReqs[0]; req.GetCampaignId() != "camp-1" || req.GetAfterSequenceId() != 0 || req.GetSessionGrant() != "grant-token" {
		t.Fatalf("unexpected subscribe request: %+v", req)
	}
	messages := room.messagesByStream[chatDefaultStreamID("camp-1")]
	if len(messages) != 1 {
		t.Fatalf("room messages = %d, want %d", len(messages), 1)
	}
	if messages[0].Kind != "ai" || messages[0].Body != "AI says hi" {
		t.Fatalf("unexpected ai message: %+v", messages[0])
	}
	if buf.Len() == 0 {
		t.Fatal("expected subscriber frame output")
	}
}

func TestConsumeCampaignAITurnUpdatesRetriesOnSubscribeError(t *testing.T) {
	setCampaignAITurnRetryDelay(t, 5*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := newRoomHub()
	room := hub.room("camp-1")
	room.setSessionID("session-1")
	room.setAIBinding("AI", "agent-1")
	room.setAISessionGrant("grant-token", 2, time.Now().UTC().Add(time.Minute))

	client := &testInvocationClient{}
	client.subscribeFn = func(ctx context.Context, req *aiv1.SubscribeCampaignTurnEventsRequest) (grpc.ServerStreamingClient[aiv1.CampaignTurnEvent], error) {
		if client.subscribeCalls == 1 {
			return nil, errors.New("temporary")
		}
		cancel()
		return &testCampaignTurnStream{ctx: ctx, recvErr: io.EOF}, nil
	}

	consumeCampaignAITurnUpdates(ctx, client, hub, "camp-1")

	if client.subscribeCalls < 2 {
		t.Fatalf("subscribe calls = %d, want at least 2", client.subscribeCalls)
	}
}

func TestWaitCampaignAITurnSubscriptionRetry(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if waitCampaignAITurnSubscriptionRetry(ctx, 50*time.Millisecond) {
		t.Fatal("expected canceled context to stop retry wait")
	}

	if !waitCampaignAITurnSubscriptionRetry(context.Background(), time.Millisecond) {
		t.Fatal("expected retry wait to return true when timer elapses")
	}
}

func setCampaignAITurnRetryDelay(t *testing.T, delay time.Duration) {
	t.Helper()

	previous := campaignAITurnSubscriptionRetryDelay
	campaignAITurnSubscriptionRetryDelay = delay
	t.Cleanup(func() {
		campaignAITurnSubscriptionRetryDelay = previous
	})
}

func waitForCampaignTurnSubscribeCalls(t *testing.T, client *testInvocationClient, want int, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if client.subscribeCalls >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("subscribe calls = %d, want at least %d", client.subscribeCalls, want)
}
