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
	invite "github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeUserStore struct {
	users    map[string]user.User
	contacts map[string]map[string]storage.Contact
	putErr   error
	getErr   error
	listErr  error
}

type userOnlyStore struct {
	users map[string]user.User
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{
		users:    make(map[string]user.User),
		contacts: make(map[string]map[string]storage.Contact),
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

func (s *userOnlyStore) PutUser(_ context.Context, u user.User) error {
	s.users[u.ID] = u
	return nil
}

func (s *userOnlyStore) GetUser(_ context.Context, userID string) (user.User, error) {
	u, ok := s.users[userID]
	if !ok {
		return user.User{}, storage.ErrNotFound
	}
	return u, nil
}

func (s *userOnlyStore) ListUsers(_ context.Context, pageSize int, pageToken string) (storage.UserPage, error) {
	users := make([]user.User, 0, len(s.users))
	for _, u := range s.users {
		users = append(users, u)
	}
	return storage.UserPage{Users: users}, nil
}

func (s *fakeUserStore) PutContact(_ context.Context, contact storage.Contact) error {
	if _, ok := s.contacts[contact.OwnerUserID]; !ok {
		s.contacts[contact.OwnerUserID] = make(map[string]storage.Contact)
	}
	s.contacts[contact.OwnerUserID][contact.ContactUserID] = contact
	return nil
}

func (s *fakeUserStore) GetContact(_ context.Context, ownerUserID string, contactUserID string) (storage.Contact, error) {
	byOwner, ok := s.contacts[ownerUserID]
	if !ok {
		return storage.Contact{}, storage.ErrNotFound
	}
	contact, ok := byOwner[contactUserID]
	if !ok {
		return storage.Contact{}, storage.ErrNotFound
	}
	return contact, nil
}

func (s *fakeUserStore) DeleteContact(_ context.Context, ownerUserID string, contactUserID string) error {
	if byOwner, ok := s.contacts[ownerUserID]; ok {
		delete(byOwner, contactUserID)
	}
	return nil
}

func (s *fakeUserStore) ListContacts(_ context.Context, ownerUserID string, pageSize int, pageToken string) (storage.ContactPage, error) {
	all := make([]storage.Contact, 0)
	if byOwner, ok := s.contacts[ownerUserID]; ok {
		for _, contact := range byOwner {
			all = append(all, contact)
		}
	}
	return storage.ContactPage{
		Contacts:      all,
		NextPageToken: "",
	}, nil
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

func TestAddContact_NilRequest(t *testing.T) {
	svc := NewAuthService(newFakeUserStore(), nil, nil)
	_, err := svc.AddContact(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAddContact_RequiresContactStore(t *testing.T) {
	store := &userOnlyStore{users: make(map[string]user.User)}
	svc := NewAuthService(store, nil, nil)
	_, err := svc.AddContact(context.Background(), &authv1.AddContactRequest{
		OwnerUserId:   "user-1",
		ContactUserId: "user-2",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestAddContact_SelfContactRejected(t *testing.T) {
	store := newFakeUserStore()
	store.users["user-1"] = user.User{ID: "user-1", Email: "alpha@example.com", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	svc := NewAuthService(store, nil, nil)

	_, err := svc.AddContact(context.Background(), &authv1.AddContactRequest{
		OwnerUserId:   "user-1",
		ContactUserId: "user-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAddContact_SuccessAndIdempotent(t *testing.T) {
	store := newFakeUserStore()
	now := time.Date(2026, 2, 21, 20, 30, 0, 0, time.UTC)
	store.users["user-1"] = user.User{ID: "user-1", Email: "alpha@example.com", CreatedAt: now, UpdatedAt: now}
	store.users["user-2"] = user.User{ID: "user-2", Email: "beta@example.com", CreatedAt: now, UpdatedAt: now}
	svc := NewAuthService(store, nil, nil)
	svc.clock = func() time.Time { return now }

	for range 2 {
		resp, err := svc.AddContact(context.Background(), &authv1.AddContactRequest{
			OwnerUserId:   "user-1",
			ContactUserId: "user-2",
		})
		if err != nil {
			t.Fatalf("add contact: %v", err)
		}
		if resp.GetContact() == nil {
			t.Fatal("expected contact in response")
		}
	}

	listResp, err := svc.ListContacts(context.Background(), &authv1.ListContactsRequest{
		OwnerUserId: "user-1",
		PageSize:    10,
	})
	if err != nil {
		t.Fatalf("list contacts: %v", err)
	}
	if len(listResp.GetContacts()) != 1 {
		t.Fatalf("contacts len = %d, want 1", len(listResp.GetContacts()))
	}
}

func TestAddContact_IdempotentResponsePreservesOriginalCreatedAt(t *testing.T) {
	store := openTempAuthStore(t)
	now := time.Date(2026, 2, 21, 20, 32, 0, 0, time.UTC)
	later := now.Add(5 * time.Minute)

	for _, u := range []user.User{
		{ID: "user-1", Email: "user1@example.com", CreatedAt: now, UpdatedAt: now},
		{ID: "user-2", Email: "user2@example.com", CreatedAt: now, UpdatedAt: now},
	} {
		if err := store.PutUser(context.Background(), u); err != nil {
			t.Fatalf("put user %s: %v", u.ID, err)
		}
	}

	svc := NewAuthService(store, store, nil)
	svc.clock = func() time.Time { return now }

	firstResp, err := svc.AddContact(context.Background(), &authv1.AddContactRequest{
		OwnerUserId:   "user-1",
		ContactUserId: "user-2",
	})
	if err != nil {
		t.Fatalf("first add contact: %v", err)
	}

	svc.clock = func() time.Time { return later }
	secondResp, err := svc.AddContact(context.Background(), &authv1.AddContactRequest{
		OwnerUserId:   "user-1",
		ContactUserId: "user-2",
	})
	if err != nil {
		t.Fatalf("second add contact: %v", err)
	}

	firstCreated := firstResp.GetContact().GetCreatedAt().AsTime()
	secondCreated := secondResp.GetContact().GetCreatedAt().AsTime()
	if !secondCreated.Equal(firstCreated) {
		t.Fatalf("created_at = %v, want %v", secondCreated, firstCreated)
	}
	if gotUpdated := secondResp.GetContact().GetUpdatedAt().AsTime(); !gotUpdated.Equal(later) {
		t.Fatalf("updated_at = %v, want %v", gotUpdated, later)
	}
}

func TestRemoveContact_Idempotent(t *testing.T) {
	store := newFakeUserStore()
	now := time.Date(2026, 2, 21, 20, 31, 0, 0, time.UTC)
	store.users["user-1"] = user.User{ID: "user-1", Email: "alpha@example.com", CreatedAt: now, UpdatedAt: now}
	store.users["user-2"] = user.User{ID: "user-2", Email: "beta@example.com", CreatedAt: now, UpdatedAt: now}
	svc := NewAuthService(store, nil, nil)
	svc.clock = func() time.Time { return now }

	_, err := svc.AddContact(context.Background(), &authv1.AddContactRequest{
		OwnerUserId:   "user-1",
		ContactUserId: "user-2",
	})
	if err != nil {
		t.Fatalf("seed contact: %v", err)
	}

	for range 2 {
		if _, err := svc.RemoveContact(context.Background(), &authv1.RemoveContactRequest{
			OwnerUserId:   "user-1",
			ContactUserId: "user-2",
		}); err != nil {
			t.Fatalf("remove contact: %v", err)
		}
	}

	listResp, err := svc.ListContacts(context.Background(), &authv1.ListContactsRequest{
		OwnerUserId: "user-1",
		PageSize:    10,
	})
	if err != nil {
		t.Fatalf("list contacts: %v", err)
	}
	if len(listResp.GetContacts()) != 0 {
		t.Fatalf("contacts len = %d, want 0", len(listResp.GetContacts()))
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
