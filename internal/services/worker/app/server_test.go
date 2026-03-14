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

	workerdomain "github.com/louisbranch/fracturing.space/internal/services/worker/domain"
)

type fakeOutboxEvent struct {
	id           string
	eventType    string
	payloadJSON  string
	attemptCount int32
}

func (e fakeOutboxEvent) GetId() string          { return e.id }
func (e fakeOutboxEvent) GetEventType() string   { return e.eventType }
func (e fakeOutboxEvent) GetPayloadJson() string { return e.payloadJSON }
func (e fakeOutboxEvent) GetAttemptCount() int32 { return e.attemptCount }

func TestServer_Run_AcksSucceeded(t *testing.T) {
	now := time.Date(2026, 2, 21, 23, 0, 0, 0, time.UTC)
	client := &fakeOutboxClient{
		leaseResponses: [][]workerdomain.OutboxEvent{
			{
				fakeOutboxEvent{id: "evt-1", eventType: "auth.signup_completed", payloadJSON: `{"user_id":"user-1"}`},
			},
			{},
		},
	}

	server := New(
		"auth",
		client,
		nil,
		map[string]EventHandler{
			"auth.signup_completed": EventHandlerFunc(func(context.Context, workerdomain.OutboxEvent) error {
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
		client.mu.Lock()
		defer client.mu.Unlock()
		return len(client.ackRequests) >= 1
	})
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run: %v", err)
	}

	client.mu.Lock()
	defer client.mu.Unlock()
	if len(client.ackRequests) != 1 {
		t.Fatalf("ack requests len = %d, want 1", len(client.ackRequests))
	}
	if got := client.ackRequests[0].Outcome; got != workerdomain.AckOutcomeSucceeded {
		t.Fatalf("ack outcome = %v, want %v", got, workerdomain.AckOutcomeSucceeded)
	}
}

func TestServer_Run_RetryThenDead(t *testing.T) {
	now := time.Date(2026, 2, 21, 23, 5, 0, 0, time.UTC)
	client := &fakeOutboxClient{
		leaseResponses: [][]workerdomain.OutboxEvent{
			{
				fakeOutboxEvent{id: "evt-1", eventType: "auth.signup_completed", payloadJSON: `{"user_id":"user-1"}`},
			},
			{
				fakeOutboxEvent{id: "evt-1", eventType: "auth.signup_completed", payloadJSON: `{"user_id":"user-1"}`, attemptCount: 1},
			},
			{},
		},
	}

	server := New(
		"auth",
		client,
		nil,
		map[string]EventHandler{
			"auth.signup_completed": EventHandlerFunc(func(context.Context, workerdomain.OutboxEvent) error {
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
		client.mu.Lock()
		defer client.mu.Unlock()
		return len(client.ackRequests) >= 2
	})
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run: %v", err)
	}

	client.mu.Lock()
	defer client.mu.Unlock()
	if len(client.ackRequests) < 2 {
		t.Fatalf("ack requests len = %d, want >=2", len(client.ackRequests))
	}
	if got := client.ackRequests[0].Outcome; got != workerdomain.AckOutcomeRetry {
		t.Fatalf("first ack outcome = %v, want %v", got, workerdomain.AckOutcomeRetry)
	}
	if got := client.ackRequests[1].Outcome; got != workerdomain.AckOutcomeDead {
		t.Fatalf("second ack outcome = %v, want %v", got, workerdomain.AckOutcomeDead)
	}
}

func TestServer_Run_UnsupportedEventTypeDeadLetters(t *testing.T) {
	now := time.Date(2026, 2, 21, 23, 8, 0, 0, time.UTC)
	client := &fakeOutboxClient{
		leaseResponses: [][]workerdomain.OutboxEvent{
			{
				fakeOutboxEvent{id: "evt-unknown", eventType: "auth.unknown", payloadJSON: `{}`},
			},
		},
	}

	server := New(
		"auth",
		client,
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
		client.mu.Lock()
		defer client.mu.Unlock()
		return len(client.ackRequests) >= 1
	})
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run: %v", err)
	}

	client.mu.Lock()
	defer client.mu.Unlock()
	if got := client.ackRequests[0].Outcome; got != workerdomain.AckOutcomeDead {
		t.Fatalf("ack outcome = %v, want %v", got, workerdomain.AckOutcomeDead)
	}
}

func TestServer_Run_LogsLeaseErrors(t *testing.T) {
	now := time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC)
	client := &fakeOutboxClient{leaseErr: errors.New("lease unavailable")}

	var logs bytes.Buffer
	restoreLog := captureLogs(&logs)
	defer restoreLog()

	server := New(
		"auth",
		client,
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
		client.mu.Lock()
		defer client.mu.Unlock()
		return client.leaseCalls >= 1
	})
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run: %v", err)
	}

	if !strings.Contains(logs.String(), "lease auth integration outbox events") {
		t.Fatalf("expected lease error log, got %q", logs.String())
	}
}

func TestServer_Run_LogsAckErrors(t *testing.T) {
	now := time.Date(2026, 2, 22, 0, 5, 0, 0, time.UTC)
	client := &fakeOutboxClient{
		leaseResponses: [][]workerdomain.OutboxEvent{
			{
				fakeOutboxEvent{id: "evt-1", eventType: "auth.signup_completed", payloadJSON: `{"user_id":"user-1"}`},
			},
		},
		ackErr: errors.New("ack unavailable"),
	}

	var logs bytes.Buffer
	restoreLog := captureLogs(&logs)
	defer restoreLog()

	server := New(
		"auth",
		client,
		nil,
		map[string]EventHandler{
			"auth.signup_completed": EventHandlerFunc(func(context.Context, workerdomain.OutboxEvent) error {
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
		client.mu.Lock()
		defer client.mu.Unlock()
		return len(client.ackRequests) >= 1
	})
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run: %v", err)
	}

	if !strings.Contains(logs.String(), "ack auth integration outbox event") {
		t.Fatalf("expected ack error log, got %q", logs.String())
	}
}

func TestServer_RunOnce_BackoffsOnAckErrors(t *testing.T) {
	now := time.Date(2026, 2, 22, 0, 10, 0, 0, time.UTC)
	client := &fakeOutboxClient{
		leaseResponses: [][]workerdomain.OutboxEvent{
			{
				fakeOutboxEvent{id: "evt-1", eventType: "auth.signup_completed", payloadJSON: `{"user_id":"user-1"}`},
				fakeOutboxEvent{id: "evt-2", eventType: "auth.signup_completed", payloadJSON: `{"user_id":"user-2"}`},
			},
		},
		ackErr: errors.New("ack unavailable"),
	}

	server := New(
		"auth",
		client,
		nil,
		map[string]EventHandler{
			"auth.signup_completed": EventHandlerFunc(func(context.Context, workerdomain.OutboxEvent) error {
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

func TestParallelLoop_Run(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		var ran int
		loop := parallelLoop{loops: []workerLoop{
			loopRunnerFunc(func(context.Context) error {
				ran++
				return nil
			}),
			nil,
			loopRunnerFunc(func(context.Context) error {
				ran++
				return nil
			}),
		}}

		if err := loop.Run(context.Background()); err != nil {
			t.Fatalf("Run: %v", err)
		}
		if ran != 2 {
			t.Fatalf("ran = %d, want 2", ran)
		}
	})

	t.Run("returns first non-cancel error", func(t *testing.T) {
		t.Parallel()

		want := errors.New("boom")
		loop := parallelLoop{loops: []workerLoop{
			loopRunnerFunc(func(context.Context) error { return want }),
			loopRunnerFunc(func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			}),
		}}

		if err := loop.Run(context.Background()); !errors.Is(err, want) {
			t.Fatalf("Run error = %v, want %v", err, want)
		}
	})
}

type fakeOutboxClient struct {
	mu             sync.Mutex
	leaseResponses [][]workerdomain.OutboxEvent
	leaseCalls     int
	leaseErr       error
	ackRequests    []AckRequest
	ackErr         error
}

func (f *fakeOutboxClient) Lease(_ context.Context, _ LeaseRequest) ([]workerdomain.OutboxEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.leaseCalls++
	if f.leaseErr != nil {
		return nil, f.leaseErr
	}
	if f.leaseCalls > len(f.leaseResponses) {
		return nil, nil
	}
	return f.leaseResponses[f.leaseCalls-1], nil
}

func (f *fakeOutboxClient) Ack(_ context.Context, req AckRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ackRequests = append(f.ackRequests, req)
	if f.ackErr != nil {
		return f.ackErr
	}
	return nil
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
