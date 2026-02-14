package auth

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/invite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeUserStore struct {
	users   map[string]user.User
	putErr  error
	getErr  error
	listErr error
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

func TestCreateUser_NilRequest(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), nil, nil)
	_, err := svc.CreateUser(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateUser_MissingStore(t *testing.T) {
	svc := NewAuthService(nil, nil, nil)
	_, err := svc.CreateUser(context.Background(), &authv1.CreateUserRequest{DisplayName: "Alice"})
	assertStatusCode(t, err, codes.Internal)
}

func TestCreateUser_EmptyDisplayName(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), nil, nil)
	_, err := svc.CreateUser(context.Background(), &authv1.CreateUserRequest{DisplayName: "  "})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateUser_Success(t *testing.T) {
	store := newFakeUserStore()
	svc := NewAuthService(store, nil, nil)
	fixedTime := time.Date(2026, 1, 23, 10, 0, 0, 0, time.UTC)
	svc.clock = func() time.Time { return fixedTime }
	svc.idGenerator = func() (string, error) { return "user-123", nil }

	resp, err := svc.CreateUser(context.Background(), &authv1.CreateUserRequest{DisplayName: "  Alice  "})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if resp.GetUser().GetId() != "user-123" {
		t.Fatalf("expected id user-123, got %q", resp.GetUser().GetId())
	}
	if resp.GetUser().GetDisplayName() != "Alice" {
		t.Fatalf("expected trimmed display name, got %q", resp.GetUser().GetDisplayName())
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
	store.users["user-123"] = user.User{ID: "user-123", DisplayName: "Alice", CreatedAt: time.Now(), UpdatedAt: time.Now()}
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
	store.users["user-1"] = user.User{ID: "user-1", DisplayName: "Alpha", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	store.users["user-2"] = user.User{ID: "user-2", DisplayName: "Beta", CreatedAt: time.Now(), UpdatedAt: time.Now()}

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
	store.users["user-1"] = user.User{ID: "user-1", DisplayName: "Alpha", CreatedAt: time.Now(), UpdatedAt: time.Now()}

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
