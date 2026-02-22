package auth

import (
	"context"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/auth/magiclink"
	"github.com/louisbranch/fracturing.space/internal/services/auth/oauth"
	"github.com/louisbranch/fracturing.space/internal/services/auth/passkey"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultListUsersPageSize = 10
	maxListUsersPageSize     = 50
	defaultListContactsSize  = 10
	maxListContactsSize      = 50
)

// AuthService implements the auth.v1.AuthService gRPC API.
//
// It is the stable surface game, admin, and tooling call to perform identity
// actions without directly touching storage details.
type AuthService struct {
	authv1.UnimplementedAuthServiceServer
	store              storage.UserStore
	contactStore       storage.ContactStore
	passkeyStore       storage.PasskeyStore
	emailStore         storage.EmailStore
	magicLinkStore     storage.MagicLinkStore
	oauthStore         *oauth.Store
	passkeyConfig      passkey.Config
	magicLinkConfig    magiclink.Config
	passkeyWebAuthn    passkeyProvider
	passkeyInitErr     error
	passkeyParser      passkeyParser
	clock              func() time.Time
	idGenerator        func() (string, error)
	passkeyIDGenerator func() (string, error)
}

// NewAuthService builds a service with defaults for the auth package.
//
// Defaults are intentionally assembled here so transport handlers can treat this
// as the canonical auth domain entrypoint.
func NewAuthService(store storage.UserStore, passkeyStore storage.PasskeyStore, oauthStore *oauth.Store) *AuthService {
	config := passkey.LoadConfigFromEnv()
	magicConfig := magiclink.LoadConfigFromEnv()
	webAuthn, err := webauthn.New(&webauthn.Config{
		RPDisplayName: config.RPDisplayName,
		RPID:          config.RPID,
		RPOrigins:     config.RPOrigins,
	})
	var emailStore storage.EmailStore
	var magicLinkStore storage.MagicLinkStore
	var contactStore storage.ContactStore
	if store != nil {
		if typed, ok := store.(storage.EmailStore); ok {
			emailStore = typed
		}
		if typed, ok := store.(storage.MagicLinkStore); ok {
			magicLinkStore = typed
		}
		if typed, ok := store.(storage.ContactStore); ok {
			contactStore = typed
		}
	}
	return &AuthService{
		store:              store,
		contactStore:       contactStore,
		passkeyStore:       passkeyStore,
		emailStore:         emailStore,
		magicLinkStore:     magicLinkStore,
		oauthStore:         oauthStore,
		passkeyConfig:      config,
		magicLinkConfig:    magicConfig,
		passkeyWebAuthn:    webAuthn,
		passkeyInitErr:     err,
		passkeyParser:      defaultPasskeyParser{},
		clock:              time.Now,
		idGenerator:        id.NewID,
		passkeyIDGenerator: id.NewID,
	}
}

// CreateUser creates a user and returns the canonical identity for campaign actions.
func (s *AuthService) CreateUser(ctx context.Context, in *authv1.CreateUserRequest) (*authv1.CreateUserResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create user request is required")
	}

	created, err := newUserCreator(s).create(ctx, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &authv1.CreateUserResponse{User: userToProto(created)}, nil
}

// IssueJoinGrant issues a one-time join grant for campaign invites.
func (s *AuthService) IssueJoinGrant(ctx context.Context, in *authv1.IssueJoinGrantRequest) (*authv1.IssueJoinGrantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "issue join grant request is required")
	}

	issued, err := newJoinGrantIssuer(s).issue(ctx, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &authv1.IssueJoinGrantResponse{
		JoinGrant: issued.grant,
		Jti:       issued.jti,
		ExpiresAt: timestamppb.New(issued.expiresAt),
	}, nil
}

// GetUser resolves a user ID to an identity record for cross-service lookups.
func (s *AuthService) GetUser(ctx context.Context, in *authv1.GetUserRequest) (*authv1.GetUserResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get user request is required")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "user store is not configured")
	}

	userID := strings.TrimSpace(in.GetUserId())
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}

	found, err := s.store.GetUser(ctx, userID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	return &authv1.GetUserResponse{User: userToProto(found)}, nil
}

// ListUsers returns a page of users for operator-facing views and audits.
func (s *AuthService) ListUsers(ctx context.Context, in *authv1.ListUsersRequest) (*authv1.ListUsersResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list users request is required")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "user store is not configured")
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListUsersPageSize,
		Max:     maxListUsersPageSize,
	})

	page, err := s.store.ListUsers(ctx, pageSize, in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list users: %v", err)
	}

	response := &authv1.ListUsersResponse{NextPageToken: page.NextPageToken}
	if len(page.Users) == 0 {
		return response, nil
	}

	response.Users = make([]*authv1.User, 0, len(page.Users))
	for _, u := range page.Users {
		response.Users = append(response.Users, userToProto(u))
	}

	return response, nil
}

// AddContact adds one owner-scoped user contact.
func (s *AuthService) AddContact(ctx context.Context, in *authv1.AddContactRequest) (*authv1.AddContactResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "add contact request is required")
	}
	if s.store == nil || s.contactStore == nil {
		return nil, status.Error(codes.Internal, "contact store is not configured")
	}

	ownerUserID := strings.TrimSpace(in.GetOwnerUserId())
	contactUserID := strings.TrimSpace(in.GetContactUserId())
	if ownerUserID == "" {
		return nil, status.Error(codes.InvalidArgument, "owner user id is required")
	}
	if contactUserID == "" {
		return nil, status.Error(codes.InvalidArgument, "contact user id is required")
	}
	if ownerUserID == contactUserID {
		return nil, status.Error(codes.InvalidArgument, "contact user id must differ from owner user id")
	}

	if _, err := s.store.GetUser(ctx, ownerUserID); err != nil {
		return nil, handleDomainError(err)
	}
	if _, err := s.store.GetUser(ctx, contactUserID); err != nil {
		return nil, handleDomainError(err)
	}

	now := time.Now()
	if s.clock != nil {
		now = s.clock()
	}

	contact := storage.Contact{
		OwnerUserID:   ownerUserID,
		ContactUserID: contactUserID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.contactStore.PutContact(ctx, contact); err != nil {
		return nil, status.Errorf(codes.Internal, "add contact: %v", err)
	}
	persisted, err := s.contactStore.GetContact(ctx, ownerUserID, contactUserID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "add contact: %v", err)
	}

	return &authv1.AddContactResponse{Contact: contactToProto(persisted)}, nil
}

// RemoveContact removes one owner-scoped user contact.
func (s *AuthService) RemoveContact(ctx context.Context, in *authv1.RemoveContactRequest) (*authv1.RemoveContactResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "remove contact request is required")
	}
	if s.store == nil || s.contactStore == nil {
		return nil, status.Error(codes.Internal, "contact store is not configured")
	}

	ownerUserID := strings.TrimSpace(in.GetOwnerUserId())
	contactUserID := strings.TrimSpace(in.GetContactUserId())
	if ownerUserID == "" {
		return nil, status.Error(codes.InvalidArgument, "owner user id is required")
	}
	if contactUserID == "" {
		return nil, status.Error(codes.InvalidArgument, "contact user id is required")
	}
	if ownerUserID == contactUserID {
		return nil, status.Error(codes.InvalidArgument, "contact user id must differ from owner user id")
	}

	if _, err := s.store.GetUser(ctx, ownerUserID); err != nil {
		return nil, handleDomainError(err)
	}

	if err := s.contactStore.DeleteContact(ctx, ownerUserID, contactUserID); err != nil {
		return nil, status.Errorf(codes.Internal, "remove contact: %v", err)
	}
	return &authv1.RemoveContactResponse{}, nil
}

// ListContacts returns owner-scoped contacts.
func (s *AuthService) ListContacts(ctx context.Context, in *authv1.ListContactsRequest) (*authv1.ListContactsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list contacts request is required")
	}
	if s.store == nil || s.contactStore == nil {
		return nil, status.Error(codes.Internal, "contact store is not configured")
	}

	ownerUserID := strings.TrimSpace(in.GetOwnerUserId())
	if ownerUserID == "" {
		return nil, status.Error(codes.InvalidArgument, "owner user id is required")
	}
	if _, err := s.store.GetUser(ctx, ownerUserID); err != nil {
		return nil, handleDomainError(err)
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListContactsSize,
		Max:     maxListContactsSize,
	})
	page, err := s.contactStore.ListContacts(ctx, ownerUserID, pageSize, in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list contacts: %v", err)
	}

	response := &authv1.ListContactsResponse{NextPageToken: page.NextPageToken}
	if len(page.Contacts) == 0 {
		return response, nil
	}
	response.Contacts = make([]*authv1.Contact, 0, len(page.Contacts))
	for _, contact := range page.Contacts {
		response.Contacts = append(response.Contacts, contactToProto(contact))
	}
	return response, nil
}

func userToProto(u user.User) *authv1.User {
	return &authv1.User{
		Id:        u.ID,
		Email:     u.Email,
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}
}

func contactToProto(contact storage.Contact) *authv1.Contact {
	return &authv1.Contact{
		OwnerUserId:   contact.OwnerUserID,
		ContactUserId: contact.ContactUserID,
		CreatedAt:     timestamppb.New(contact.CreatedAt),
		UpdatedAt:     timestamppb.New(contact.UpdatedAt),
	}
}

// handleDomainError converts domain errors to gRPC status using the structured error system.
func handleDomainError(err error) error {
	return apperrors.HandleError(err, apperrors.DefaultLocale)
}
