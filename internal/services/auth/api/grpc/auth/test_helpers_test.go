package auth

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	authsqlite "github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeUserStore struct {
	users        map[string]user.User
	outboxEvents []storage.IntegrationOutboxEvent
	outboxPutErr error
	signupPutErr error
	putErr       error
	getErr       error
	listErr      error
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{users: make(map[string]user.User)}
}

func (s *fakeUserStore) PutUser(_ context.Context, u user.User) error {
	if s.putErr != nil {
		return s.putErr
	}
	s.users[u.ID] = u
	return nil
}

func (s *fakeUserStore) GetUser(_ context.Context, userID string) (user.User, error) {
	if s.getErr != nil {
		return user.User{}, s.getErr
	}
	u, ok := s.users[userID]
	if !ok {
		return user.User{}, storage.ErrNotFound
	}
	return u, nil
}

func (s *fakeUserStore) GetUserByUsername(_ context.Context, username string) (user.User, error) {
	if s.getErr != nil {
		return user.User{}, s.getErr
	}
	for _, u := range s.users {
		if u.Username == username {
			return u, nil
		}
	}
	return user.User{}, storage.ErrNotFound
}

func (s *fakeUserStore) ListUsers(_ context.Context, pageSize int, pageToken string) (storage.UserPage, error) {
	if s.listErr != nil {
		return storage.UserPage{}, s.listErr
	}
	users := make([]user.User, 0, len(s.users))
	for _, u := range s.users {
		users = append(users, u)
	}
	return storage.UserPage{Users: users}, nil
}

func (s *fakeUserStore) EnqueueIntegrationOutboxEvent(_ context.Context, event storage.IntegrationOutboxEvent) error {
	if s.outboxPutErr != nil {
		return s.outboxPutErr
	}
	s.outboxEvents = append(s.outboxEvents, event)
	return nil
}

func (s *fakeUserStore) PutUserPasskeyWithIntegrationOutboxEvent(ctx context.Context, u user.User, _ storage.PasskeyCredential, _ storage.WebSession, event storage.IntegrationOutboxEvent) error {
	if s.signupPutErr != nil {
		return s.signupPutErr
	}
	if err := s.PutUser(ctx, u); err != nil {
		return err
	}
	return s.EnqueueIntegrationOutboxEvent(ctx, event)
}

func openTempAuthStore(t *testing.T) *authsqlite.Store {
	t.Helper()
	path := t.TempDir() + "/auth.db"
	store, err := authsqlite.Open(path)
	if err != nil {
		t.Fatalf("open auth store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected status %v, got nil", want)
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != want {
		t.Fatalf("expected status %v, got %v", want, st.Code())
	}
}
