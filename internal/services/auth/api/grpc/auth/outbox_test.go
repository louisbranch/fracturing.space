package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
)

// fakeTransactionalUserStore implements both UserStore and UserOutboxTransactionalStore.
type fakeTransactionalUserStore struct {
	fakeUserStore
	txPutErr error
	txCalled bool
}

func (s *fakeTransactionalUserStore) PutUserWithIntegrationOutboxEvent(ctx context.Context, u user.User, event storage.IntegrationOutboxEvent) error {
	s.txCalled = true
	if s.txPutErr != nil {
		return s.txPutErr
	}
	if err := s.PutUser(ctx, u); err != nil {
		return err
	}
	return s.EnqueueIntegrationOutboxEvent(ctx, event)
}

// --- signupCompletedOutboxEvent ---

func TestSignupCompletedOutboxEvent_Success(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	svc := &AuthService{
		clock:       func() time.Time { return now },
		idGenerator: func() (string, error) { return "event-1", nil },
	}
	u := user.User{ID: "u1", Username: "alice"}
	event, err := svc.signupCompletedOutboxEvent(u, "passkey")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.ID != "event-1" {
		t.Fatalf("event id = %q, want %q", event.ID, "event-1")
	}
	if event.EventType != signupCompletedEventType {
		t.Fatalf("event type = %q, want %q", event.EventType, signupCompletedEventType)
	}
	if event.Status != storage.IntegrationOutboxStatusPending {
		t.Fatalf("status = %q, want pending", event.Status)
	}
	if want := signupCompletedDedupeKey("u1"); event.DedupeKey != want {
		t.Fatalf("dedupe key = %q, want %q", event.DedupeKey, want)
	}
}

func TestSignupCompletedOutboxEvent_IDGeneratorError(t *testing.T) {
	svc := &AuthService{
		clock:       func() time.Time { return time.Now() },
		idGenerator: func() (string, error) { return "", errors.New("id gen fail") },
	}
	_, err := svc.signupCompletedOutboxEvent(user.User{ID: "u1"}, "passkey")
	if err == nil {
		t.Fatal("expected error from id generator")
	}
}

// --- persistUserWithSignupCompletedOutbox ---

func TestPersistUserWithSignupCompletedOutbox_NilService(t *testing.T) {
	var svc *AuthService
	err := svc.persistUserWithSignupCompletedOutbox(context.Background(), user.User{ID: "u1"}, "passkey")
	if err != nil {
		t.Fatalf("nil service should return nil: %v", err)
	}
}

func TestPersistUserWithSignupCompletedOutbox_TransactionalPath(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	txStore := &fakeTransactionalUserStore{
		fakeUserStore: fakeUserStore{users: make(map[string]user.User)},
	}
	svc := &AuthService{
		store:       txStore,
		clock:       func() time.Time { return now },
		idGenerator: func() (string, error) { return "event-1", nil },
	}

	err := svc.persistUserWithSignupCompletedOutbox(context.Background(), user.User{ID: "u1", Username: "alice"}, "passkey")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !txStore.txCalled {
		t.Fatal("expected transactional store path to be used")
	}
}

func TestPersistUserWithSignupCompletedOutbox_FallbackPath(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	store := newFakeUserStore()
	svc := &AuthService{
		store:       store,
		clock:       func() time.Time { return now },
		idGenerator: func() (string, error) { return "event-1", nil },
	}

	err := svc.persistUserWithSignupCompletedOutbox(context.Background(), user.User{ID: "u1", Username: "alice"}, "passkey")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := store.users["u1"]; !ok {
		t.Fatal("expected user to be persisted via fallback path")
	}
	if len(store.outboxEvents) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(store.outboxEvents))
	}
}

func TestPersistUserWithSignupCompletedOutbox_PutUserError(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	store := newFakeUserStore()
	store.putErr = errors.New("put fail")
	svc := &AuthService{
		store:       store,
		clock:       func() time.Time { return now },
		idGenerator: func() (string, error) { return "event-1", nil },
	}

	err := svc.persistUserWithSignupCompletedOutbox(context.Background(), user.User{ID: "u1"}, "passkey")
	if err == nil {
		t.Fatal("expected error from put user")
	}
}

func TestPersistUserWithSignupCompletedOutbox_OutboxEnqueueError(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	store := newFakeUserStore()
	store.outboxPutErr = errors.New("enqueue fail")
	svc := &AuthService{
		store:       store,
		clock:       func() time.Time { return now },
		idGenerator: func() (string, error) { return "event-1", nil },
	}

	err := svc.persistUserWithSignupCompletedOutbox(context.Background(), user.User{ID: "u1"}, "passkey")
	if err == nil {
		t.Fatal("expected error from outbox enqueue")
	}
}

// --- enqueueSignupCompletedOutbox ---

func TestEnqueueSignupCompletedOutbox_NilService(t *testing.T) {
	var svc *AuthService
	err := svc.enqueueSignupCompletedOutbox(context.Background(), user.User{ID: "u1"}, "passkey")
	if err != nil {
		t.Fatalf("nil service should return nil: %v", err)
	}
}

func TestEnqueueSignupCompletedOutbox_StoreNotEnqueuer(t *testing.T) {
	// Use a store that does not implement signupOutboxEnqueuer (a minimal fake).
	svc := &AuthService{store: &nonEnqueuerStore{}}
	err := svc.enqueueSignupCompletedOutbox(context.Background(), user.User{ID: "u1"}, "passkey")
	if err != nil {
		t.Fatalf("non-enqueuer store should return nil: %v", err)
	}
}

func TestEnqueueSignupCompletedOutbox_Success(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	store := newFakeUserStore()
	svc := &AuthService{
		store:       store,
		clock:       func() time.Time { return now },
		idGenerator: func() (string, error) { return "event-1", nil },
	}

	err := svc.enqueueSignupCompletedOutbox(context.Background(), user.User{ID: "u1", Username: "alice"}, "passkey")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.outboxEvents) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(store.outboxEvents))
	}
}

// nonEnqueuerStore is a minimal UserStore that does NOT implement signupOutboxEnqueuer.
type nonEnqueuerStore struct{}

func (s *nonEnqueuerStore) PutUser(_ context.Context, _ user.User) error { return nil }
func (s *nonEnqueuerStore) GetUser(_ context.Context, _ string) (user.User, error) {
	return user.User{}, nil
}
func (s *nonEnqueuerStore) GetUserByUsername(_ context.Context, _ string) (user.User, error) {
	return user.User{}, nil
}
func (s *nonEnqueuerStore) ListUsers(_ context.Context, _ int, _ string) (storage.UserPage, error) {
	return storage.UserPage{}, nil
}
