package auth

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	invite "github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeUserStore struct {
	users        map[string]user.User
	outboxEvents []storage.IntegrationOutboxEvent
	outboxPutErr error
	putErr       error
	getErr       error
	listErr      error
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{
		users: make(map[string]user.User),
	}
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

func (s *fakeUserStore) ListUsers(_ context.Context, pageSize int, pageToken string) (storage.UserPage, error) {
	if s.listErr != nil {
		return storage.UserPage{}, s.listErr
	}
	users := make([]user.User, 0, len(s.users))
	for _, u := range s.users {
		users = append(users, u)
	}
	return storage.UserPage{Users: users, NextPageToken: ""}, nil
}

func (s *fakeUserStore) EnqueueIntegrationOutboxEvent(_ context.Context, event storage.IntegrationOutboxEvent) error {
	if s.outboxPutErr != nil {
		return s.outboxPutErr
	}
	s.outboxEvents = append(s.outboxEvents, event)
	return nil
}

func TestCreateUser_NilRequest(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), nil, nil)
	_, err := svc.CreateUser(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateUser_MissingStore(t *testing.T) {
	svc := NewAuthService(nil, nil, nil)
	_, err := svc.CreateUser(context.Background(), &authv1.CreateUserRequest{Email: "alice@example.com"})
	assertStatusCode(t, err, codes.Internal)
}

func TestCreateUser_EmptyUsername(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), nil, nil)
	_, err := svc.CreateUser(context.Background(), &authv1.CreateUserRequest{Email: "  "})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateUser_Success(t *testing.T) {
	store := newFakeUserStore()
	svc := NewAuthService(store, nil, nil)
	fixedTime := time.Date(2026, 1, 23, 10, 0, 0, 0, time.UTC)
	svc.clock = func() time.Time { return fixedTime }
	svc.idGenerator = func() (string, error) { return "user-123", nil }

	resp, err := svc.CreateUser(context.Background(), &authv1.CreateUserRequest{Email: "  Alice@example.COM  "})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if resp.GetUser().GetId() != "user-123" {
		t.Fatalf("expected id user-123, got %q", resp.GetUser().GetId())
	}
	if resp.GetUser().GetEmail() != "alice@example.com" {
		t.Fatalf("expected normalized email, got %q", resp.GetUser().GetEmail())
	}
}

func TestCreateUser_EnqueuesSignupCompletedOutbox(t *testing.T) {
	store := newFakeUserStore()
	svc := NewAuthService(store, nil, nil)
	fixedTime := time.Date(2026, 2, 21, 18, 0, 0, 0, time.UTC)
	svc.clock = func() time.Time { return fixedTime }
	svc.idGenerator = func() (string, error) { return "user-123", nil }

	resp, err := svc.CreateUser(context.Background(), &authv1.CreateUserRequest{Email: "Alice@example.com"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if resp.GetUser().GetId() != "user-123" {
		t.Fatalf("user id = %q, want %q", resp.GetUser().GetId(), "user-123")
	}

	if len(store.outboxEvents) != 1 {
		t.Fatalf("outbox events len = %d, want 1", len(store.outboxEvents))
	}
	event := store.outboxEvents[0]
	if event.EventType != "auth.signup_completed" {
		t.Fatalf("outbox event type = %q, want %q", event.EventType, "auth.signup_completed")
	}
	if event.DedupeKey != "signup_completed:user:user-123:v1" {
		t.Fatalf("outbox dedupe key = %q, want %q", event.DedupeKey, "signup_completed:user:user-123:v1")
	}
}

func TestCreateUser_OutboxFailureRollsBackUser(t *testing.T) {
	store := openTempAuthStore(t)
	seededAt := time.Date(2026, 2, 21, 19, 0, 0, 0, time.UTC)
	if err := store.EnqueueIntegrationOutboxEvent(context.Background(), storage.IntegrationOutboxEvent{
		ID:            "dup-id",
		EventType:     "auth.signup_completed",
		PayloadJSON:   "{}",
		DedupeKey:     "seed:dup-id",
		Status:        storage.IntegrationOutboxStatusPending,
		AttemptCount:  0,
		NextAttemptAt: seededAt,
		CreatedAt:     seededAt,
		UpdatedAt:     seededAt,
	}); err != nil {
		t.Fatalf("seed integration outbox: %v", err)
	}

	svc := NewAuthService(store, store, nil)
	svc.clock = func() time.Time { return seededAt.Add(time.Minute) }
	svc.idGenerator = func() (string, error) { return "dup-id", nil }

	_, err := svc.CreateUser(context.Background(), &authv1.CreateUserRequest{Email: "Alice@example.com"})
	assertStatusCode(t, err, codes.Internal)

	_, getErr := store.GetUser(context.Background(), "dup-id")
	if !errors.Is(getErr, storage.ErrNotFound) {
		t.Fatalf("get user err = %v, want %v", getErr, storage.ErrNotFound)
	}
}

func TestCreateUser_PersistsPrimaryEmail(t *testing.T) {
	store := openTempAuthStore(t)
	svc := NewAuthService(store, store, nil)
	svc.clock = func() time.Time { return time.Date(2026, 1, 23, 10, 0, 0, 0, time.UTC) }
	svc.idGenerator = func() (string, error) { return "user-123", nil }

	resp, err := svc.CreateUser(context.Background(), &authv1.CreateUserRequest{Email: "  Alice@example.COM  "})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if resp.GetUser().GetId() != "user-123" {
		t.Fatalf("expected id user-123, got %q", resp.GetUser().GetId())
	}

	emails, err := store.ListUserEmailsByUser(context.Background(), "user-123")
	if err != nil {
		t.Fatalf("list user emails: %v", err)
	}
	if len(emails) != 1 {
		t.Fatalf("expected 1 email, got %d", len(emails))
	}
	if emails[0].Email != "alice@example.com" {
		t.Fatalf("expected normalized primary email, got %q", emails[0].Email)
	}
}

func TestGetUser_NilRequest(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), nil, nil)
	_, err := svc.GetUser(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetUser_MissingStore(t *testing.T) {
	svc := NewAuthService(nil, nil, nil)
	_, err := svc.GetUser(context.Background(), &authv1.GetUserRequest{UserId: "user-1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestGetUser_EmptyID(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), nil, nil)
	_, err := svc.GetUser(context.Background(), &authv1.GetUserRequest{UserId: "  "})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetUser_NotFound(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), nil, nil)
	_, err := svc.GetUser(context.Background(), &authv1.GetUserRequest{UserId: "missing"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetUser_Success(t *testing.T) {
	store := newFakeUserStore()
	store.users["user-123"] = user.User{ID: "user-123", Email: "alice", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	svc := NewAuthService(store, nil, nil)

	resp, err := svc.GetUser(context.Background(), &authv1.GetUserRequest{UserId: "user-123"})
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if resp.GetUser().GetId() != "user-123" {
		t.Fatalf("expected id user-123, got %q", resp.GetUser().GetId())
	}
}

func TestListUsers_NilRequest(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), nil, nil)
	_, err := svc.ListUsers(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListUsers_MissingStore(t *testing.T) {
	svc := NewAuthService(nil, nil, nil)
	_, err := svc.ListUsers(context.Background(), &authv1.ListUsersRequest{})
	assertStatusCode(t, err, codes.Internal)
}

func TestListUsers_Success(t *testing.T) {
	store := newFakeUserStore()
	store.users["user-1"] = user.User{ID: "user-1", Email: "alpha", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	store.users["user-2"] = user.User{ID: "user-2", Email: "beta", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	svc := NewAuthService(store, nil, nil)
	resp, err := svc.ListUsers(context.Background(), &authv1.ListUsersRequest{PageSize: 10})
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if len(resp.GetUsers()) != 2 {
		t.Fatalf("expected 2 users, got %d", len(resp.GetUsers()))
	}
}

func TestIssueJoinGrant_Success(t *testing.T) {
	store := newFakeUserStore()
	store.users["user-1"] = user.User{ID: "user-1", Email: "alpha", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate join grant key: %v", err)
	}
	issuer := "test-issuer"
	audience := "game-service"
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", issuer)
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", audience)
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY", base64.RawStdEncoding.EncodeToString(privateKey))
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_TTL", "5m")

	svc := NewAuthService(store, nil, nil)
	fixedTime := time.Date(2026, 1, 23, 10, 0, 0, 0, time.UTC)
	svc.clock = func() time.Time { return fixedTime }

	resp, err := svc.IssueJoinGrant(context.Background(), &authv1.IssueJoinGrantRequest{
		UserId:        "user-1",
		CampaignId:    "campaign-1",
		InviteId:      "invite-1",
		ParticipantId: "participant-1",
	})
	if err != nil {
		t.Fatalf("issue join grant: %v", err)
	}
	if resp.GetJoinGrant() == "" {
		t.Fatal("expected join grant")
	}
	if resp.GetJti() == "" {
		t.Fatal("expected jti")
	}
	if resp.GetExpiresAt() == nil {
		t.Fatal("expected expires_at")
	}

	claims, err := invite.ValidateJoinGrant(resp.GetJoinGrant(), invite.JoinGrantExpectation{
		CampaignID: "campaign-1",
		InviteID:   "invite-1",
		UserID:     "user-1",
	}, invite.JoinGrantConfig{
		Issuer:   issuer,
		Audience: audience,
		Key:      publicKey,
		Now:      func() time.Time { return fixedTime },
	})
	if err != nil {
		t.Fatalf("validate join grant: %v", err)
	}
	if claims.JWTID != resp.GetJti() {
		t.Fatalf("jti = %s, want %s", claims.JWTID, resp.GetJti())
	}
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
