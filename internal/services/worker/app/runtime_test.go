package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	workersqlite "github.com/louisbranch/fracturing.space/internal/services/worker/storage/sqlite"
)

func TestAttemptStoreRecorder_EmptyConsumerUsesDefault(t *testing.T) {
	store := openTempWorkerStore(t)
	recorder := &attemptStoreRecorder{
		store:    store,
		consumer: "",
	}

	err := recorder.RecordAttempt(context.Background(), Attempt{
		EventID:      "evt-1",
		EventType:    "auth.signup_completed",
		Outcome:      authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_SUCCEEDED,
		AttemptCount: 1,
		CreatedAt:    time.Date(2026, 2, 22, 0, 20, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("record attempt: %v", err)
	}

	attempts, err := store.ListAttempts(context.Background(), 10)
	if err != nil {
		t.Fatalf("list attempts: %v", err)
	}
	if len(attempts) != 1 {
		t.Fatalf("attempts len = %d, want 1", len(attempts))
	}
	if attempts[0].Consumer != defaultConsumer {
		t.Fatalf("consumer = %q, want %q", attempts[0].Consumer, defaultConsumer)
	}
}

func TestAttemptStoreRecorder_StoresCanonicalOutcomeValues(t *testing.T) {
	store := openTempWorkerStore(t)
	recorder := &attemptStoreRecorder{
		store:    store,
		consumer: defaultConsumer,
	}
	now := time.Date(2026, 2, 22, 0, 25, 0, 0, time.UTC)

	cases := []struct {
		name    string
		outcome authv1.IntegrationOutboxAckOutcome
		want    string
	}{
		{
			name:    "succeeded",
			outcome: authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_SUCCEEDED,
			want:    "succeeded",
		},
		{
			name:    "retry",
			outcome: authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_RETRY,
			want:    "retry",
		},
		{
			name:    "dead",
			outcome: authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_DEAD,
			want:    "dead",
		},
	}

	for i, tc := range cases {
		if err := recorder.RecordAttempt(context.Background(), Attempt{
			EventID:      "evt-" + tc.want,
			EventType:    "auth.signup_completed",
			Outcome:      tc.outcome,
			AttemptCount: int32(i + 1),
			CreatedAt:    now.Add(time.Duration(i) * time.Second),
		}); err != nil {
			t.Fatalf("record attempt (%s): %v", tc.name, err)
		}
	}

	attempts, err := store.ListAttempts(context.Background(), 10)
	if err != nil {
		t.Fatalf("list attempts: %v", err)
	}
	if len(attempts) != len(cases) {
		t.Fatalf("attempts len = %d, want %d", len(attempts), len(cases))
	}

	got := map[string]bool{}
	for _, attempt := range attempts {
		got[attempt.Outcome] = true
	}
	for _, tc := range cases {
		if !got[tc.want] {
			t.Fatalf("missing canonical outcome %q in stored attempts: %v", tc.want, got)
		}
	}
}

func openTempWorkerStore(t *testing.T) *workersqlite.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "worker.db")
	store, err := workersqlite.Open(path)
	if err != nil {
		t.Fatalf("open worker store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close worker store: %v", err)
		}
	})
	return store
}

func TestFanoutEventHandlers_RunsHandlersInOrder(t *testing.T) {
	called := make([]string, 0, 2)
	handler := fanoutEventHandlers(
		EventHandlerFunc(func(context.Context, *authv1.IntegrationOutboxEvent) error {
			called = append(called, "first")
			return nil
		}),
		EventHandlerFunc(func(context.Context, *authv1.IntegrationOutboxEvent) error {
			called = append(called, "second")
			return nil
		}),
	)
	if handler == nil {
		t.Fatal("expected non-nil fanout handler")
	}
	err := handler.Handle(context.Background(), &authv1.IntegrationOutboxEvent{Id: "evt-1"})
	if err != nil {
		t.Fatalf("handle fanout: %v", err)
	}
	if len(called) != 2 || called[0] != "first" || called[1] != "second" {
		t.Fatalf("called order = %v, want [first second]", called)
	}
}

func TestFanoutEventHandlers_StopsAtFirstError(t *testing.T) {
	called := make([]string, 0, 2)
	handler := fanoutEventHandlers(
		EventHandlerFunc(func(context.Context, *authv1.IntegrationOutboxEvent) error {
			called = append(called, "first")
			return errors.New("boom")
		}),
		EventHandlerFunc(func(context.Context, *authv1.IntegrationOutboxEvent) error {
			called = append(called, "second")
			return nil
		}),
	)
	err := handler.Handle(context.Background(), &authv1.IntegrationOutboxEvent{Id: "evt-1"})
	if err == nil {
		t.Fatal("expected fanout error")
	}
	if len(called) != 1 || called[0] != "first" {
		t.Fatalf("called order = %v, want [first]", called)
	}
}
