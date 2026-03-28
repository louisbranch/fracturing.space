package app

import (
	"context"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	workerdomain "github.com/louisbranch/fracturing.space/internal/services/worker/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestNewAuthOutboxClientAdapter_Nil(t *testing.T) {
	t.Parallel()

	if got := newAuthOutboxClientAdapter(nil); got != nil {
		t.Fatalf("newAuthOutboxClientAdapter(nil) = %T, want nil", got)
	}
}

func TestNewGameOutboxClientAdapter_Nil(t *testing.T) {
	t.Parallel()

	if got := newGameOutboxClientAdapter(nil); got != nil {
		t.Fatalf("newGameOutboxClientAdapter(nil) = %T, want nil", got)
	}
}

func TestAuthOutboxClientAdapter_LeaseAndAck(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 10, 5, 0, 0, 0, time.UTC)
	client := &fakeAuthIntegrationOutboxClient{
		leaseResp: &authv1.LeaseIntegrationOutboxEventsResponse{
			Events: []*authv1.IntegrationOutboxEvent{{
				Id:           "evt-1",
				EventType:    "auth.signup_completed",
				PayloadJson:  `{"user_id":"user-1"}`,
				AttemptCount: 2,
			}},
		},
	}
	adapter := newAuthOutboxClientAdapter(client)

	events, err := adapter.Lease(context.Background(), LeaseRequest{
		Consumer: "worker-1",
		Limit:    7,
		LeaseTTL: 30 * time.Second,
		Now:      now,
	})
	if err != nil {
		t.Fatalf("lease: %v", err)
	}
	if len(events) != 1 || events[0].GetId() != "evt-1" {
		t.Fatalf("leased events = %+v, want evt-1", events)
	}
	if client.lastLease == nil || client.lastLease.GetConsumer() != "worker-1" || client.lastLease.GetLimit() != 7 {
		t.Fatalf("lease request = %+v", client.lastLease)
	}
	assertWorkerServiceID(t, client.lastLeaseCtx)

	retryAt := now.Add(2 * time.Minute)
	if err := adapter.Ack(context.Background(), AckRequest{
		EventID:       "evt-1",
		Consumer:      "worker-1",
		Outcome:       workerdomain.AckOutcomeRetry,
		NextAttemptAt: retryAt,
		LastError:     "temporary",
	}); err != nil {
		t.Fatalf("ack retry: %v", err)
	}
	if client.lastAck == nil || client.lastAck.GetOutcome() != authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_RETRY {
		t.Fatalf("ack request = %+v", client.lastAck)
	}
	if got := client.lastAck.GetNextAttemptAt().AsTime(); !got.Equal(retryAt) {
		t.Fatalf("ack next attempt at = %v, want %v", got, retryAt)
	}
	assertWorkerServiceID(t, client.lastAckCtx)
}

func TestAuthOutboxClientAdapter_LeaseError(t *testing.T) {
	t.Parallel()

	want := context.DeadlineExceeded
	adapter := newAuthOutboxClientAdapter(&fakeAuthIntegrationOutboxClient{leaseErr: want})

	if _, err := adapter.Lease(context.Background(), LeaseRequest{}); err != want {
		t.Fatalf("Lease error = %v, want %v", err, want)
	}
}

func TestAuthOutboxClientAdapter_AckDead(t *testing.T) {
	t.Parallel()

	client := &fakeAuthIntegrationOutboxClient{}
	adapter := newAuthOutboxClientAdapter(client)

	if err := adapter.Ack(context.Background(), AckRequest{
		EventID:   "evt-1",
		Consumer:  "worker-1",
		Outcome:   workerdomain.AckOutcomeDead,
		LastError: "permanent",
	}); err != nil {
		t.Fatalf("ack dead: %v", err)
	}
	if client.lastAck == nil || client.lastAck.GetOutcome() != authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_DEAD {
		t.Fatalf("ack request = %+v", client.lastAck)
	}
}

func TestAuthOutboxClientAdapter_AckSucceeded(t *testing.T) {
	t.Parallel()

	client := &fakeAuthIntegrationOutboxClient{}
	adapter := newAuthOutboxClientAdapter(client)

	if err := adapter.Ack(context.Background(), AckRequest{
		EventID:  "evt-1",
		Consumer: "worker-1",
		Outcome:  workerdomain.AckOutcomeSucceeded,
	}); err != nil {
		t.Fatalf("ack succeeded: %v", err)
	}
	if client.lastAck == nil || client.lastAck.GetOutcome() != authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_SUCCEEDED {
		t.Fatalf("ack request = %+v, want succeeded outcome", client.lastAck)
	}
	if client.lastAck.GetNextAttemptAt() != nil {
		t.Fatalf("ack next attempt at = %v, want nil", client.lastAck.GetNextAttemptAt())
	}
}

func TestGameOutboxClientAdapter_LeaseAndAck(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 10, 5, 10, 0, 0, time.UTC)
	client := &fakeGameIntegrationOutboxClient{
		leaseResp: &gamev1.LeaseIntegrationOutboxEventsResponse{
			Events: []*gamev1.IntegrationOutboxEvent{{
				Id:           "evt-2",
				EventType:    "game.invite.created.v1",
				PayloadJson:  `{"invite_id":"invite-1"}`,
				AttemptCount: 1,
			}},
		},
	}
	adapter := newGameOutboxClientAdapter(client)

	events, err := adapter.Lease(context.Background(), LeaseRequest{
		Consumer: "worker-2",
		Limit:    3,
		LeaseTTL: time.Minute,
		Now:      now,
	})
	if err != nil {
		t.Fatalf("lease: %v", err)
	}
	if len(events) != 1 || events[0].GetId() != "evt-2" {
		t.Fatalf("leased events = %+v, want evt-2", events)
	}
	if client.lastLease == nil || client.lastLease.GetConsumer() != "worker-2" || client.lastLease.GetLimit() != 3 {
		t.Fatalf("lease request = %+v", client.lastLease)
	}
	assertWorkerServiceID(t, client.lastLeaseCtx)

	if err := adapter.Ack(context.Background(), AckRequest{
		EventID:  "evt-2",
		Consumer: "worker-2",
		Outcome:  workerdomain.AckOutcomeSucceeded,
	}); err != nil {
		t.Fatalf("ack succeeded: %v", err)
	}
	if client.lastAck == nil || client.lastAck.GetOutcome() != gamev1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_SUCCEEDED {
		t.Fatalf("ack request = %+v", client.lastAck)
	}
	assertWorkerServiceID(t, client.lastAckCtx)
}

func TestGameOutboxClientAdapter_LeaseError(t *testing.T) {
	t.Parallel()

	want := context.DeadlineExceeded
	adapter := newGameOutboxClientAdapter(&fakeGameIntegrationOutboxClient{leaseErr: want})

	if _, err := adapter.Lease(context.Background(), LeaseRequest{}); err != want {
		t.Fatalf("Lease error = %v, want %v", err, want)
	}
}

func TestGameOutboxClientAdapter_AckRetry(t *testing.T) {
	t.Parallel()

	retryAt := time.Date(2026, 3, 10, 5, 12, 0, 0, time.UTC)
	client := &fakeGameIntegrationOutboxClient{}
	adapter := newGameOutboxClientAdapter(client)

	if err := adapter.Ack(context.Background(), AckRequest{
		EventID:       "evt-2",
		Consumer:      "worker-2",
		Outcome:       workerdomain.AckOutcomeRetry,
		NextAttemptAt: retryAt,
		LastError:     "temporary",
	}); err != nil {
		t.Fatalf("ack retry: %v", err)
	}
	if client.lastAck == nil || client.lastAck.GetOutcome() != gamev1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_RETRY {
		t.Fatalf("ack request = %+v", client.lastAck)
	}
	if got := client.lastAck.GetNextAttemptAt().AsTime(); !got.Equal(retryAt) {
		t.Fatalf("ack next attempt at = %v, want %v", got, retryAt)
	}
}

func TestGameOutboxClientAdapter_AckDead(t *testing.T) {
	t.Parallel()

	client := &fakeGameIntegrationOutboxClient{}
	adapter := newGameOutboxClientAdapter(client)

	if err := adapter.Ack(context.Background(), AckRequest{
		EventID:   "evt-2",
		Consumer:  "worker-2",
		Outcome:   workerdomain.AckOutcomeDead,
		LastError: "permanent",
	}); err != nil {
		t.Fatalf("ack dead: %v", err)
	}
	if client.lastAck == nil || client.lastAck.GetOutcome() != gamev1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_DEAD {
		t.Fatalf("ack request = %+v, want dead outcome", client.lastAck)
	}
	if client.lastAck.GetNextAttemptAt() != nil {
		t.Fatalf("ack next attempt at = %v, want nil", client.lastAck.GetNextAttemptAt())
	}
}

func TestOutboxClientAdapter_LeaseSkipsNilEvents(t *testing.T) {
	t.Parallel()

	authAdapter := newAuthOutboxClientAdapter(&fakeAuthIntegrationOutboxClient{
		leaseResp: &authv1.LeaseIntegrationOutboxEventsResponse{
			Events: []*authv1.IntegrationOutboxEvent{
				nil,
				{Id: "evt-auth"},
			},
		},
	})
	authEvents, err := authAdapter.Lease(context.Background(), LeaseRequest{})
	if err != nil {
		t.Fatalf("auth lease: %v", err)
	}
	if len(authEvents) != 1 || authEvents[0].GetId() != "evt-auth" {
		t.Fatalf("auth leased events = %+v, want evt-auth only", authEvents)
	}

	gameAdapter := newGameOutboxClientAdapter(&fakeGameIntegrationOutboxClient{
		leaseResp: &gamev1.LeaseIntegrationOutboxEventsResponse{
			Events: []*gamev1.IntegrationOutboxEvent{
				nil,
				{Id: "evt-game"},
			},
		},
	})
	gameEvents, err := gameAdapter.Lease(context.Background(), LeaseRequest{})
	if err != nil {
		t.Fatalf("game lease: %v", err)
	}
	if len(gameEvents) != 1 || gameEvents[0].GetId() != "evt-game" {
		t.Fatalf("game leased events = %+v, want evt-game only", gameEvents)
	}
}

func TestOutboxClientAdapter_AckRejectsUnknownOutcome(t *testing.T) {
	t.Parallel()

	authAdapter := newAuthOutboxClientAdapter(&fakeAuthIntegrationOutboxClient{})
	if err := authAdapter.Ack(context.Background(), AckRequest{
		EventID:  "evt-1",
		Consumer: "worker-1",
		Outcome:  workerdomain.AckOutcomeUnknown,
	}); err == nil {
		t.Fatal("expected auth ack error for unknown outcome")
	}

	gameAdapter := newGameOutboxClientAdapter(&fakeGameIntegrationOutboxClient{})
	if err := gameAdapter.Ack(context.Background(), AckRequest{
		EventID:  "evt-2",
		Consumer: "worker-2",
		Outcome:  workerdomain.AckOutcomeUnknown,
	}); err == nil {
		t.Fatal("expected game ack error for unknown outcome")
	}
}

func assertWorkerServiceID(t *testing.T, ctx context.Context) {
	t.Helper()
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatal("expected outgoing metadata")
	}
	values := md.Get(grpcmeta.ServiceIDHeader)
	if len(values) == 0 || values[0] != serviceaddr.ServiceWorker {
		t.Fatalf("service-id metadata = %v, want %q", values, serviceaddr.ServiceWorker)
	}
}

type fakeAuthIntegrationOutboxClient struct {
	leaseResp    *authv1.LeaseIntegrationOutboxEventsResponse
	leaseErr     error
	lastLease    *authv1.LeaseIntegrationOutboxEventsRequest
	lastLeaseCtx context.Context

	ackErr     error
	lastAck    *authv1.AckIntegrationOutboxEventRequest
	lastAckCtx context.Context
}

func (f *fakeAuthIntegrationOutboxClient) LeaseIntegrationOutboxEvents(ctx context.Context, in *authv1.LeaseIntegrationOutboxEventsRequest, _ ...grpc.CallOption) (*authv1.LeaseIntegrationOutboxEventsResponse, error) {
	f.lastLeaseCtx = ctx
	f.lastLease = in
	if f.leaseErr != nil {
		return nil, f.leaseErr
	}
	return f.leaseResp, nil
}

func (f *fakeAuthIntegrationOutboxClient) AckIntegrationOutboxEvent(ctx context.Context, in *authv1.AckIntegrationOutboxEventRequest, _ ...grpc.CallOption) (*authv1.AckIntegrationOutboxEventResponse, error) {
	f.lastAckCtx = ctx
	f.lastAck = in
	if f.ackErr != nil {
		return nil, f.ackErr
	}
	return &authv1.AckIntegrationOutboxEventResponse{}, nil
}

type fakeGameIntegrationOutboxClient struct {
	leaseResp    *gamev1.LeaseIntegrationOutboxEventsResponse
	leaseErr     error
	lastLease    *gamev1.LeaseIntegrationOutboxEventsRequest
	lastLeaseCtx context.Context

	ackErr     error
	lastAck    *gamev1.AckIntegrationOutboxEventRequest
	lastAckCtx context.Context
}

func (f *fakeGameIntegrationOutboxClient) LeaseIntegrationOutboxEvents(ctx context.Context, in *gamev1.LeaseIntegrationOutboxEventsRequest, _ ...grpc.CallOption) (*gamev1.LeaseIntegrationOutboxEventsResponse, error) {
	f.lastLeaseCtx = ctx
	f.lastLease = in
	if f.leaseErr != nil {
		return nil, f.leaseErr
	}
	return f.leaseResp, nil
}

func (f *fakeGameIntegrationOutboxClient) AckIntegrationOutboxEvent(ctx context.Context, in *gamev1.AckIntegrationOutboxEventRequest, _ ...grpc.CallOption) (*gamev1.AckIntegrationOutboxEventResponse, error) {
	f.lastAckCtx = ctx
	f.lastAck = in
	if f.ackErr != nil {
		return nil, f.ackErr
	}
	return &gamev1.AckIntegrationOutboxEventResponse{}, nil
}
