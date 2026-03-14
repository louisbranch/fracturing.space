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
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	"github.com/louisbranch/fracturing.space/internal/services/shared/joingrant"
	"google.golang.org/grpc/codes"
)

func TestLookupUserByUsername_NilRequest(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), nil, nil)
	_, err := svc.LookupUserByUsername(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestLookupUserByUsername_MissingStore(t *testing.T) {
	svc := NewAuthService(nil, nil, nil)
	_, err := svc.LookupUserByUsername(context.Background(), &authv1.LookupUserByUsernameRequest{Username: "alice"})
	assertStatusCode(t, err, codes.Internal)
}

func TestLookupUserByUsername_Success(t *testing.T) {
	store := newFakeUserStore()
	now := time.Date(2026, 2, 23, 15, 0, 0, 0, time.UTC)
	store.users["user-1"] = user.User{ID: "user-1", Username: "alice", CreatedAt: now, UpdatedAt: now}

	svc := NewAuthService(store, nil, nil)
	resp, err := svc.LookupUserByUsername(context.Background(), &authv1.LookupUserByUsernameRequest{Username: "alice"})
	if err != nil {
		t.Fatalf("lookup user by username: %v", err)
	}
	if got := resp.GetUser().GetId(); got != "user-1" {
		t.Fatalf("user id = %q, want %q", got, "user-1")
	}
	if got := resp.GetUser().GetUsername(); got != "alice" {
		t.Fatalf("username = %q, want %q", got, "alice")
	}
}

func TestGetUser_Success(t *testing.T) {
	store := newFakeUserStore()
	now := time.Date(2026, 2, 23, 15, 0, 0, 0, time.UTC)
	store.users["user-123"] = user.User{ID: "user-123", Username: "alice", CreatedAt: now, UpdatedAt: now}
	svc := NewAuthService(store, nil, nil)

	resp, err := svc.GetUser(context.Background(), &authv1.GetUserRequest{UserId: "user-123"})
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got := resp.GetUser().GetUsername(); got != "alice" {
		t.Fatalf("username = %q, want %q", got, "alice")
	}
}

func TestCreateWebSession_Success(t *testing.T) {
	store := openTempAuthStore(t)
	now := time.Date(2026, 2, 23, 15, 0, 0, 0, time.UTC)
	if err := store.PutUser(context.Background(), user.User{ID: "user-1", Username: "alpha", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("put user: %v", err)
	}
	svc := NewAuthService(store, store, nil)
	svc.clock = func() time.Time { return now }
	svc.idGenerator = func() (string, error) { return "ws-1", nil }

	resp, err := svc.CreateWebSession(context.Background(), &authv1.CreateWebSessionRequest{UserId: "user-1"})
	if err != nil {
		t.Fatalf("create web session: %v", err)
	}
	if resp.GetSession().GetId() != "ws-1" {
		t.Fatalf("session id = %q, want %q", resp.GetSession().GetId(), "ws-1")
	}
}

func TestRevokeWebSession_RevokesSession(t *testing.T) {
	store := openTempAuthStore(t)
	now := time.Date(2026, 2, 23, 15, 0, 0, 0, time.UTC)
	if err := store.PutUser(context.Background(), user.User{ID: "user-1", Username: "alpha", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("put user: %v", err)
	}
	svc := NewAuthService(store, store, nil)
	svc.clock = func() time.Time { return now }
	svc.idGenerator = func() (string, error) { return "ws-1", nil }

	if _, err := svc.CreateWebSession(context.Background(), &authv1.CreateWebSessionRequest{UserId: "user-1"}); err != nil {
		t.Fatalf("create web session: %v", err)
	}
	if _, err := svc.RevokeWebSession(context.Background(), &authv1.RevokeWebSessionRequest{SessionId: "ws-1"}); err != nil {
		t.Fatalf("revoke web session: %v", err)
	}
	_, err := svc.GetWebSession(context.Background(), &authv1.GetWebSessionRequest{SessionId: "ws-1"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestListUsers_Success(t *testing.T) {
	store := newFakeUserStore()
	now := time.Date(2026, 2, 23, 15, 0, 0, 0, time.UTC)
	store.users["user-1"] = user.User{ID: "user-1", Username: "alpha", CreatedAt: now, UpdatedAt: now}
	store.users["user-2"] = user.User{ID: "user-2", Username: "beta", CreatedAt: now, UpdatedAt: now}

	svc := NewAuthService(store, nil, nil)
	resp, err := svc.ListUsers(context.Background(), &authv1.ListUsersRequest{PageSize: 10})
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if len(resp.GetUsers()) != 2 {
		t.Fatalf("users len = %d, want 2", len(resp.GetUsers()))
	}
}

func TestIssueJoinGrant_Success(t *testing.T) {
	store := newFakeUserStore()
	now := time.Date(2026, 2, 23, 15, 0, 0, 0, time.UTC)
	store.users["user-1"] = user.User{ID: "user-1", Username: "alpha", CreatedAt: now, UpdatedAt: now}

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
	svc.clock = func() time.Time { return now }

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

	claims, err := joingrant.Validate(resp.GetJoinGrant(), joingrant.Expectation{
		CampaignID: "campaign-1",
		InviteID:   "invite-1",
		UserID:     "user-1",
	}, joingrant.Config{
		Issuer:   issuer,
		Audience: audience,
		Key:      publicKey,
		Now:      func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("validate join grant: %v", err)
	}
	if claims.JWTID != resp.GetJti() {
		t.Fatalf("jti = %s, want %s", claims.JWTID, resp.GetJti())
	}
}

func TestCreateWebSession_RollsBackMissingUser(t *testing.T) {
	store := openTempAuthStore(t)
	svc := NewAuthService(store, store, nil)
	_, err := svc.CreateWebSession(context.Background(), &authv1.CreateWebSessionRequest{UserId: "missing"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestLookupUserByUsername_NotFound(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), nil, nil)
	_, err := svc.LookupUserByUsername(context.Background(), &authv1.LookupUserByUsernameRequest{Username: "missing"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestLookupUserByUsername_EmptyUsername(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), nil, nil)
	_, err := svc.LookupUserByUsername(context.Background(), &authv1.LookupUserByUsernameRequest{Username: "  "})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListUsers_StoreError(t *testing.T) {
	store := newFakeUserStore()
	store.listErr = errors.New("boom")
	svc := NewAuthService(store, nil, nil)
	_, err := svc.ListUsers(context.Background(), &authv1.ListUsersRequest{})
	assertStatusCode(t, err, codes.Internal)
}

func TestGetUser_NotFound(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), nil, nil)
	_, err := svc.GetUser(context.Background(), &authv1.GetUserRequest{UserId: "missing"})
	assertStatusCode(t, err, codes.NotFound)
}
