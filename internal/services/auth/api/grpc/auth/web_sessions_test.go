package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
)

// fakeWebSessionStore is a test double for storage.WebSessionStore.
type fakeWebSessionStore struct {
	sessions  map[string]storage.WebSession
	putErr    error
	getErr    error
	revokeErr error
}

func newFakeWebSessionStore() *fakeWebSessionStore {
	return &fakeWebSessionStore{sessions: make(map[string]storage.WebSession)}
}

func (s *fakeWebSessionStore) PutWebSession(_ context.Context, session storage.WebSession) error {
	if s.putErr != nil {
		return s.putErr
	}
	s.sessions[session.ID] = session
	return nil
}

func (s *fakeWebSessionStore) GetWebSession(_ context.Context, id string) (storage.WebSession, error) {
	if s.getErr != nil {
		return storage.WebSession{}, s.getErr
	}
	sess, ok := s.sessions[id]
	if !ok {
		return storage.WebSession{}, storage.ErrNotFound
	}
	return sess, nil
}

func (s *fakeWebSessionStore) RevokeWebSession(_ context.Context, id string, revokedAt time.Time) error {
	if s.revokeErr != nil {
		return s.revokeErr
	}
	sess, ok := s.sessions[id]
	if !ok {
		return storage.ErrNotFound
	}
	sess.RevokedAt = &revokedAt
	s.sessions[id] = sess
	return nil
}

func (s *fakeWebSessionStore) RevokeWebSessionsByUser(_ context.Context, userID string, revokedAt time.Time) error {
	for id, sess := range s.sessions {
		if sess.UserID == userID {
			sess.RevokedAt = &revokedAt
			s.sessions[id] = sess
		}
	}
	return nil
}

func (s *fakeWebSessionStore) DeleteExpiredWebSessions(_ context.Context, _ time.Time) error {
	return nil
}

// newTestAuthServiceWithSessions creates an AuthService with both a user store and web session store.
func newTestAuthServiceWithSessions(userStore *fakeUserStore, sessionStore *fakeWebSessionStore) *AuthService {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	seq := 0
	return &AuthService{
		store:           userStore,
		webSessionStore: sessionStore,
		clock:           func() time.Time { return now },
		idGenerator: func() (string, error) {
			seq++
			return "gen-id-" + string(rune('0'+seq)), nil
		},
	}
}

// --- CreateWebSession ---

func TestCreateWebSession_NilRequest(t *testing.T) {
	svc := newTestAuthServiceWithSessions(newFakeUserStore(), newFakeWebSessionStore())
	_, err := svc.CreateWebSession(context.Background(), nil)
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestCreateWebSession_MissingUserStore(t *testing.T) {
	svc := &AuthService{webSessionStore: newFakeWebSessionStore()}
	_, err := svc.CreateWebSession(context.Background(), &authv1.CreateWebSessionRequest{UserId: "u1"})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestCreateWebSession_MissingSessionStore(t *testing.T) {
	svc := &AuthService{store: newFakeUserStore()}
	_, err := svc.CreateWebSession(context.Background(), &authv1.CreateWebSessionRequest{UserId: "u1"})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestCreateWebSession_EmptyUserID(t *testing.T) {
	svc := newTestAuthServiceWithSessions(newFakeUserStore(), newFakeWebSessionStore())
	_, err := svc.CreateWebSession(context.Background(), &authv1.CreateWebSessionRequest{UserId: "  "})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestCreateWebSession_UserNotFound(t *testing.T) {
	svc := newTestAuthServiceWithSessions(newFakeUserStore(), newFakeWebSessionStore())
	_, err := svc.CreateWebSession(context.Background(), &authv1.CreateWebSessionRequest{UserId: "missing"})
	grpcassert.StatusCode(t, err, codes.NotFound)
}

func TestCreateWebSession_DefaultTTL(t *testing.T) {
	userStore := newFakeUserStore()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	userStore.users["u1"] = user.User{ID: "u1", Username: "alice", CreatedAt: now, UpdatedAt: now}
	sessionStore := newFakeWebSessionStore()
	svc := newTestAuthServiceWithSessions(userStore, sessionStore)

	resp, err := svc.CreateWebSession(context.Background(), &authv1.CreateWebSessionRequest{UserId: "u1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetSession().GetUserId() != "u1" {
		t.Fatalf("user id = %q, want %q", resp.GetSession().GetUserId(), "u1")
	}
	// Invariant: default TTL is 24h.
	expiresAt := resp.GetSession().GetExpiresAt().AsTime()
	createdAt := resp.GetSession().GetCreatedAt().AsTime()
	if got := expiresAt.Sub(createdAt); got != 24*time.Hour {
		t.Fatalf("ttl = %v, want 24h", got)
	}
}

func TestCreateWebSession_CustomTTL(t *testing.T) {
	userStore := newFakeUserStore()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	userStore.users["u1"] = user.User{ID: "u1", Username: "alice", CreatedAt: now, UpdatedAt: now}
	sessionStore := newFakeWebSessionStore()
	svc := newTestAuthServiceWithSessions(userStore, sessionStore)

	resp, err := svc.CreateWebSession(context.Background(), &authv1.CreateWebSessionRequest{UserId: "u1", TtlSeconds: 3600})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expiresAt := resp.GetSession().GetExpiresAt().AsTime()
	createdAt := resp.GetSession().GetCreatedAt().AsTime()
	if got := expiresAt.Sub(createdAt); got != time.Hour {
		t.Fatalf("ttl = %v, want 1h", got)
	}
}

func TestCreateWebSession_PutStoreError(t *testing.T) {
	userStore := newFakeUserStore()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	userStore.users["u1"] = user.User{ID: "u1", Username: "alice", CreatedAt: now, UpdatedAt: now}
	sessionStore := newFakeWebSessionStore()
	sessionStore.putErr = errors.New("db write fail")
	svc := newTestAuthServiceWithSessions(userStore, sessionStore)

	_, err := svc.CreateWebSession(context.Background(), &authv1.CreateWebSessionRequest{UserId: "u1"})
	grpcassert.StatusCode(t, err, codes.Internal)
}

// --- GetWebSession ---

func TestGetWebSession_NilRequest(t *testing.T) {
	svc := newTestAuthServiceWithSessions(newFakeUserStore(), newFakeWebSessionStore())
	_, err := svc.GetWebSession(context.Background(), nil)
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestGetWebSession_EmptySessionID(t *testing.T) {
	svc := newTestAuthServiceWithSessions(newFakeUserStore(), newFakeWebSessionStore())
	_, err := svc.GetWebSession(context.Background(), &authv1.GetWebSessionRequest{SessionId: " "})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestGetWebSession_NotFound(t *testing.T) {
	svc := newTestAuthServiceWithSessions(newFakeUserStore(), newFakeWebSessionStore())
	_, err := svc.GetWebSession(context.Background(), &authv1.GetWebSessionRequest{SessionId: "missing"})
	grpcassert.StatusCode(t, err, codes.NotFound)
}

func TestGetWebSession_StoreError(t *testing.T) {
	sessionStore := newFakeWebSessionStore()
	sessionStore.getErr = errors.New("db read fail")
	svc := newTestAuthServiceWithSessions(newFakeUserStore(), sessionStore)

	_, err := svc.GetWebSession(context.Background(), &authv1.GetWebSessionRequest{SessionId: "s1"})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestGetWebSession_ExpiredSession(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	userStore := newFakeUserStore()
	userStore.users["u1"] = user.User{ID: "u1", Username: "alice", CreatedAt: now, UpdatedAt: now}
	sessionStore := newFakeWebSessionStore()
	sessionStore.sessions["s1"] = storage.WebSession{
		ID:        "s1",
		UserID:    "u1",
		CreatedAt: now.Add(-25 * time.Hour),
		ExpiresAt: now.Add(-1 * time.Hour), // expired 1h ago
	}
	svc := newTestAuthServiceWithSessions(userStore, sessionStore)

	// Invariant: expired sessions are treated as not found.
	_, err := svc.GetWebSession(context.Background(), &authv1.GetWebSessionRequest{SessionId: "s1"})
	grpcassert.StatusCode(t, err, codes.NotFound)
}

func TestGetWebSession_RevokedSession(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	revokedAt := now.Add(-30 * time.Minute)
	userStore := newFakeUserStore()
	userStore.users["u1"] = user.User{ID: "u1", Username: "alice", CreatedAt: now, UpdatedAt: now}
	sessionStore := newFakeWebSessionStore()
	sessionStore.sessions["s1"] = storage.WebSession{
		ID:        "s1",
		UserID:    "u1",
		CreatedAt: now.Add(-1 * time.Hour),
		ExpiresAt: now.Add(23 * time.Hour),
		RevokedAt: &revokedAt,
	}
	svc := newTestAuthServiceWithSessions(userStore, sessionStore)

	// Invariant: revoked sessions are treated as not found.
	_, err := svc.GetWebSession(context.Background(), &authv1.GetWebSessionRequest{SessionId: "s1"})
	grpcassert.StatusCode(t, err, codes.NotFound)
}

func TestGetWebSession_ValidSession(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	userStore := newFakeUserStore()
	userStore.users["u1"] = user.User{ID: "u1", Username: "alice", CreatedAt: now, UpdatedAt: now}
	sessionStore := newFakeWebSessionStore()
	sessionStore.sessions["s1"] = storage.WebSession{
		ID:        "s1",
		UserID:    "u1",
		CreatedAt: now.Add(-1 * time.Hour),
		ExpiresAt: now.Add(23 * time.Hour),
	}
	svc := newTestAuthServiceWithSessions(userStore, sessionStore)

	resp, err := svc.GetWebSession(context.Background(), &authv1.GetWebSessionRequest{SessionId: "s1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := resp.GetSession().GetId(); got != "s1" {
		t.Fatalf("session id = %q, want %q", got, "s1")
	}
	if got := resp.GetUser().GetUsername(); got != "alice" {
		t.Fatalf("username = %q, want %q", got, "alice")
	}
}

// --- RevokeWebSession ---

func TestRevokeWebSession_NilRequest(t *testing.T) {
	svc := newTestAuthServiceWithSessions(newFakeUserStore(), newFakeWebSessionStore())
	_, err := svc.RevokeWebSession(context.Background(), nil)
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestRevokeWebSession_EmptyID(t *testing.T) {
	svc := newTestAuthServiceWithSessions(newFakeUserStore(), newFakeWebSessionStore())
	_, err := svc.RevokeWebSession(context.Background(), &authv1.RevokeWebSessionRequest{SessionId: ""})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestRevokeWebSession_NotFoundIsIdempotent(t *testing.T) {
	svc := newTestAuthServiceWithSessions(newFakeUserStore(), newFakeWebSessionStore())
	resp, err := svc.RevokeWebSession(context.Background(), &authv1.RevokeWebSessionRequest{SessionId: "gone"})
	if err != nil {
		t.Fatalf("revoke of missing session should succeed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestRevokeWebSession_StoreError(t *testing.T) {
	sessionStore := newFakeWebSessionStore()
	sessionStore.sessions["s1"] = storage.WebSession{ID: "s1", UserID: "u1"}
	sessionStore.revokeErr = errors.New("db write fail")
	svc := newTestAuthServiceWithSessions(newFakeUserStore(), sessionStore)

	_, err := svc.RevokeWebSession(context.Background(), &authv1.RevokeWebSessionRequest{SessionId: "s1"})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestRevokeWebSession_Success(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionStore := newFakeWebSessionStore()
	sessionStore.sessions["s1"] = storage.WebSession{
		ID:        "s1",
		UserID:    "u1",
		CreatedAt: now.Add(-1 * time.Hour),
		ExpiresAt: now.Add(23 * time.Hour),
	}
	svc := newTestAuthServiceWithSessions(newFakeUserStore(), sessionStore)

	_, err := svc.RevokeWebSession(context.Background(), &authv1.RevokeWebSessionRequest{SessionId: "s1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sessionStore.sessions["s1"].RevokedAt == nil {
		t.Fatal("expected session to be revoked")
	}
}

// --- webSessionToProto ---

func TestWebSessionToProto_WithRevokedAt(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	revokedAt := now.Add(1 * time.Hour)
	session := storage.WebSession{
		ID:        "s1",
		UserID:    "u1",
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
		RevokedAt: &revokedAt,
	}
	proto := webSessionToProto(session)
	if proto.GetRevokedAt() == nil {
		t.Fatal("expected RevokedAt to be set")
	}
	if got := proto.GetRevokedAt().AsTime(); !got.Equal(revokedAt) {
		t.Fatalf("RevokedAt = %v, want %v", got, revokedAt)
	}
}

func TestWebSessionToProto_WithoutRevokedAt(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	session := storage.WebSession{
		ID:        "s1",
		UserID:    "u1",
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}
	proto := webSessionToProto(session)
	if proto.GetRevokedAt() != nil {
		t.Fatal("expected RevokedAt to be nil")
	}
}
