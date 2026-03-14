package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	workerdomain "github.com/louisbranch/fracturing.space/internal/services/worker/domain"
	workersqlite "github.com/louisbranch/fracturing.space/internal/services/worker/storage/sqlite"
	"google.golang.org/grpc"
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
		Outcome:      workerdomain.AckOutcomeSucceeded,
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
		outcome workerdomain.AckOutcome
		want    string
	}{
		{name: "succeeded", outcome: workerdomain.AckOutcomeSucceeded, want: "succeeded"},
		{name: "retry", outcome: workerdomain.AckOutcomeRetry, want: "retry"},
		{name: "dead", outcome: workerdomain.AckOutcomeDead, want: "dead"},
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
		EventHandlerFunc(func(context.Context, workerdomain.OutboxEvent) error {
			called = append(called, "first")
			return nil
		}),
		EventHandlerFunc(func(context.Context, workerdomain.OutboxEvent) error {
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
		EventHandlerFunc(func(context.Context, workerdomain.OutboxEvent) error {
			called = append(called, "first")
			return errors.New("boom")
		}),
		EventHandlerFunc(func(context.Context, workerdomain.OutboxEvent) error {
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

func TestSyncSocialUserDirectory_BackfillsAllUsers(t *testing.T) {
	authClient := &authDirectoryBootstrapClientStub{
		responses: []*authv1.ListUsersResponse{
			{
				Users: []*authv1.User{
					{Id: "user-1", Username: "alpha"},
					{Id: "user-2", Username: "beta"},
				},
				NextPageToken: "page-2",
			},
			{
				Users: []*authv1.User{
					{Id: "user-3", Username: "gamma"},
				},
			},
		},
	}
	socialClient := &socialDirectoryBootstrapClientStub{}

	if err := syncSocialUserDirectory(context.Background(), authClient, socialClient); err != nil {
		t.Fatalf("sync social user directory: %v", err)
	}
	if len(socialClient.requests) != 3 {
		t.Fatalf("sync requests len = %d, want 3", len(socialClient.requests))
	}
	if socialClient.requests[0].GetUsername() != "alpha" || socialClient.requests[2].GetUsername() != "gamma" {
		t.Fatalf("sync requests = %+v, want alpha..gamma", socialClient.requests)
	}
}

func TestSyncSocialUserDirectory_PropagatesSyncErrors(t *testing.T) {
	authClient := &authDirectoryBootstrapClientStub{
		responses: []*authv1.ListUsersResponse{
			{Users: []*authv1.User{{Id: "user-1", Username: "alpha"}}},
		},
	}
	socialClient := &socialDirectoryBootstrapClientStub{err: errors.New("boom")}

	if err := syncSocialUserDirectory(context.Background(), authClient, socialClient); err == nil {
		t.Fatal("expected sync error")
	}
}

type authDirectoryBootstrapClientStub struct {
	responses []*authv1.ListUsersResponse
	err       error
	callCount int
}

func (s *authDirectoryBootstrapClientStub) ListUsers(context.Context, *authv1.ListUsersRequest, ...grpc.CallOption) (*authv1.ListUsersResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.callCount >= len(s.responses) {
		return &authv1.ListUsersResponse{}, nil
	}
	resp := s.responses[s.callCount]
	s.callCount++
	return resp, nil
}

type socialDirectoryBootstrapClientStub struct {
	requests []*socialv1.SyncDirectoryUserRequest
	err      error
}

func (s *socialDirectoryBootstrapClientStub) SyncDirectoryUser(_ context.Context, in *socialv1.SyncDirectoryUserRequest, _ ...grpc.CallOption) (*socialv1.SyncDirectoryUserResponse, error) {
	s.requests = append(s.requests, in)
	if s.err != nil {
		return nil, s.err
	}
	return &socialv1.SyncDirectoryUserResponse{}, nil
}
