package app

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"google.golang.org/grpc"
)

func TestServer_Run_AcksSucceeded(t *testing.T) {
	now := time.Date(2026, 2, 21, 23, 0, 0, 0, time.UTC)
	authClient := &fakeAuthOutboxClient{
		leaseResponses: []*authv1.LeaseIntegrationOutboxEventsResponse{
			{
				Events: []*authv1.IntegrationOutboxEvent{
					{
						Id:           "evt-1",
						EventType:    "auth.signup_completed",
						PayloadJson:  `{"user_id":"user-1"}`,
						AttemptCount: 0,
					},
				},
			},
			{Events: []*authv1.IntegrationOutboxEvent{}},
		},
	}

	server := New(
		authClient,
		nil,
		map[string]EventHandler{
			"auth.signup_completed": EventHandlerFunc(func(context.Context, *authv1.IntegrationOutboxEvent) error {
				return nil
			}),
		},
		Config{
			Consumer:      "worker-1",
			PollInterval:  5 * time.Millisecond,
			LeaseTTL:      time.Minute,
			MaxAttempts:   3,
			RetryBackoff:  10 * time.Second,
			RetryMaxDelay: time.Minute,
		},
		func() time.Time { return now },
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- server.Run(ctx)
	}()

	waitForCondition(t, time.Second, func() bool {
		authClient.mu.Lock()
		defer authClient.mu.Unlock()
		return len(authClient.ackRequests) >= 1
	})
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run: %v", err)
	}

	authClient.mu.Lock()
	defer authClient.mu.Unlock()
	if len(authClient.ackRequests) != 1 {
		t.Fatalf("ack requests len = %d, want 1", len(authClient.ackRequests))
	}
	if got := authClient.ackRequests[0].GetOutcome(); got != authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_SUCCEEDED {
		t.Fatalf("ack outcome = %v, want %v", got, authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_SUCCEEDED)
	}
}

func TestServer_Run_RetryThenDead(t *testing.T) {
	now := time.Date(2026, 2, 21, 23, 5, 0, 0, time.UTC)
	authClient := &fakeAuthOutboxClient{
		leaseResponses: []*authv1.LeaseIntegrationOutboxEventsResponse{
			{
				Events: []*authv1.IntegrationOutboxEvent{
					{
						Id:           "evt-1",
						EventType:    "auth.signup_completed",
						PayloadJson:  `{"user_id":"user-1"}`,
						AttemptCount: 0,
					},
				},
			},
			{
				Events: []*authv1.IntegrationOutboxEvent{
					{
						Id:           "evt-1",
						EventType:    "auth.signup_completed",
						PayloadJson:  `{"user_id":"user-1"}`,
						AttemptCount: 1,
					},
				},
			},
			{Events: []*authv1.IntegrationOutboxEvent{}},
		},
	}

	server := New(
		authClient,
		nil,
		map[string]EventHandler{
			"auth.signup_completed": EventHandlerFunc(func(context.Context, *authv1.IntegrationOutboxEvent) error {
				return errors.New("transient failure")
			}),
		},
		Config{
			Consumer:      "worker-1",
			PollInterval:  5 * time.Millisecond,
			LeaseTTL:      time.Minute,
			MaxAttempts:   2,
			RetryBackoff:  10 * time.Second,
			RetryMaxDelay: time.Minute,
		},
		func() time.Time { return now },
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- server.Run(ctx)
	}()

	waitForCondition(t, time.Second, func() bool {
		authClient.mu.Lock()
		defer authClient.mu.Unlock()
		return len(authClient.ackRequests) >= 2
	})
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run: %v", err)
	}

	authClient.mu.Lock()
	defer authClient.mu.Unlock()
	if len(authClient.ackRequests) < 2 {
		t.Fatalf("ack requests len = %d, want >=2", len(authClient.ackRequests))
	}
	if got := authClient.ackRequests[0].GetOutcome(); got != authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_RETRY {
		t.Fatalf("first ack outcome = %v, want %v", got, authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_RETRY)
	}
	if got := authClient.ackRequests[1].GetOutcome(); got != authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_DEAD {
		t.Fatalf("second ack outcome = %v, want %v", got, authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_DEAD)
	}
}

func TestServer_Run_UnsupportedEventTypeDeadLetters(t *testing.T) {
	now := time.Date(2026, 2, 21, 23, 8, 0, 0, time.UTC)
	authClient := &fakeAuthOutboxClient{
		leaseResponses: []*authv1.LeaseIntegrationOutboxEventsResponse{
			{
				Events: []*authv1.IntegrationOutboxEvent{
					{
						Id:           "evt-unknown",
						EventType:    "auth.unknown",
						PayloadJson:  `{}`,
						AttemptCount: 0,
					},
				},
			},
		},
	}

	server := New(
		authClient,
		nil,
		map[string]EventHandler{},
		Config{
			Consumer:      "worker-1",
			PollInterval:  5 * time.Millisecond,
			LeaseTTL:      time.Minute,
			MaxAttempts:   2,
			RetryBackoff:  10 * time.Second,
			RetryMaxDelay: time.Minute,
		},
		func() time.Time { return now },
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- server.Run(ctx)
	}()

	waitForCondition(t, time.Second, func() bool {
		authClient.mu.Lock()
		defer authClient.mu.Unlock()
		return len(authClient.ackRequests) >= 1
	})
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run: %v", err)
	}

	authClient.mu.Lock()
	defer authClient.mu.Unlock()
	if got := authClient.ackRequests[0].GetOutcome(); got != authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_DEAD {
		t.Fatalf("ack outcome = %v, want %v", got, authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_DEAD)
	}
}

func TestServer_Run_LogsLeaseErrors(t *testing.T) {
	now := time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC)
	authClient := &fakeAuthOutboxClient{
		leaseErr: errors.New("lease unavailable"),
	}

	var logs bytes.Buffer
	restoreLog := captureLogs(&logs)
	defer restoreLog()

	server := New(
		authClient,
		nil,
		map[string]EventHandler{},
		Config{
			Consumer:      "worker-1",
			PollInterval:  5 * time.Millisecond,
			LeaseTTL:      time.Minute,
			MaxAttempts:   2,
			RetryBackoff:  10 * time.Second,
			RetryMaxDelay: time.Minute,
		},
		func() time.Time { return now },
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- server.Run(ctx)
	}()

	waitForCondition(t, time.Second, func() bool {
		authClient.mu.Lock()
		defer authClient.mu.Unlock()
		return authClient.leaseCalls >= 1
	})
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run: %v", err)
	}

	if !strings.Contains(logs.String(), "lease integration outbox events") {
		t.Fatalf("expected lease error log, got %q", logs.String())
	}
}

func TestServer_Run_LogsAckErrors(t *testing.T) {
	now := time.Date(2026, 2, 22, 0, 5, 0, 0, time.UTC)
	authClient := &fakeAuthOutboxClient{
		leaseResponses: []*authv1.LeaseIntegrationOutboxEventsResponse{
			{
				Events: []*authv1.IntegrationOutboxEvent{
					{
						Id:           "evt-1",
						EventType:    "auth.signup_completed",
						PayloadJson:  `{"user_id":"user-1"}`,
						AttemptCount: 0,
					},
				},
			},
		},
		ackErr: errors.New("ack unavailable"),
	}

	var logs bytes.Buffer
	restoreLog := captureLogs(&logs)
	defer restoreLog()

	server := New(
		authClient,
		nil,
		map[string]EventHandler{
			"auth.signup_completed": EventHandlerFunc(func(context.Context, *authv1.IntegrationOutboxEvent) error {
				return nil
			}),
		},
		Config{
			Consumer:      "worker-1",
			PollInterval:  5 * time.Millisecond,
			LeaseTTL:      time.Minute,
			MaxAttempts:   2,
			RetryBackoff:  10 * time.Second,
			RetryMaxDelay: time.Minute,
		},
		func() time.Time { return now },
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- server.Run(ctx)
	}()

	waitForCondition(t, time.Second, func() bool {
		authClient.mu.Lock()
		defer authClient.mu.Unlock()
		return len(authClient.ackRequests) >= 1
	})
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run: %v", err)
	}

	if !strings.Contains(logs.String(), "ack integration outbox event") {
		t.Fatalf("expected ack error log, got %q", logs.String())
	}
}

func TestServer_RunOnce_BackoffsOnAckErrors(t *testing.T) {
	now := time.Date(2026, 2, 22, 0, 10, 0, 0, time.UTC)
	authClient := &fakeAuthOutboxClient{
		leaseResponses: []*authv1.LeaseIntegrationOutboxEventsResponse{
			{
				Events: []*authv1.IntegrationOutboxEvent{
					{
						Id:           "evt-1",
						EventType:    "auth.signup_completed",
						PayloadJson:  `{"user_id":"user-1"}`,
						AttemptCount: 0,
					},
					{
						Id:           "evt-2",
						EventType:    "auth.signup_completed",
						PayloadJson:  `{"user_id":"user-2"}`,
						AttemptCount: 0,
					},
				},
			},
		},
		ackErr: errors.New("ack unavailable"),
	}

	server := New(
		authClient,
		nil,
		map[string]EventHandler{
			"auth.signup_completed": EventHandlerFunc(func(context.Context, *authv1.IntegrationOutboxEvent) error {
				return nil
			}),
		},
		Config{
			Consumer:      "worker-1",
			PollInterval:  5 * time.Millisecond,
			LeaseTTL:      time.Minute,
			MaxAttempts:   2,
			RetryBackoff:  10 * time.Second,
			RetryMaxDelay: time.Minute,
		},
		func() time.Time { return now },
	)

	start := time.Now()
	server.runOnce(context.Background())
	elapsed := time.Since(start)
	if elapsed < 15*time.Millisecond {
		t.Fatalf("runOnce elapsed = %v, want >= 15ms for ack failure backoff", elapsed)
	}
}

type fakeAuthOutboxClient struct {
	mu             sync.Mutex
	leaseResponses []*authv1.LeaseIntegrationOutboxEventsResponse
	leaseCalls     int
	leaseErr       error
	ackRequests    []*authv1.AckIntegrationOutboxEventRequest
	ackErr         error
}

func (f *fakeAuthOutboxClient) LeaseIntegrationOutboxEvents(_ context.Context, _ *authv1.LeaseIntegrationOutboxEventsRequest, _ ...grpc.CallOption) (*authv1.LeaseIntegrationOutboxEventsResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.leaseCalls++
	if f.leaseErr != nil {
		return nil, f.leaseErr
	}
	if f.leaseCalls > len(f.leaseResponses) {
		return &authv1.LeaseIntegrationOutboxEventsResponse{}, nil
	}
	resp := f.leaseResponses[f.leaseCalls-1]
	return resp, nil
}

func (f *fakeAuthOutboxClient) AckIntegrationOutboxEvent(_ context.Context, in *authv1.AckIntegrationOutboxEventRequest, _ ...grpc.CallOption) (*authv1.AckIntegrationOutboxEventResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ackRequests = append(f.ackRequests, in)
	if f.ackErr != nil {
		return nil, f.ackErr
	}
	return &authv1.AckIntegrationOutboxEventResponse{}, nil
}

func waitForCondition(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}

func captureLogs(output *bytes.Buffer) func() {
	previousWriter := log.Writer()
	previousFlags := log.Flags()
	previousPrefix := log.Prefix()
	log.SetOutput(output)
	log.SetFlags(0)
	log.SetPrefix("")
	return func() {
		log.SetOutput(previousWriter)
		log.SetFlags(previousFlags)
		log.SetPrefix(previousPrefix)
	}
}
