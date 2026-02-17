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
)

// AuthService implements the auth.v1.AuthService gRPC API.
//
// It is the stable surface game, admin, and tooling call to perform identity
// actions without directly touching storage details.
type AuthService struct {
	authv1.UnimplementedAuthServiceServer
	store              storage.UserStore
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
	if store != nil {
		if typed, ok := store.(storage.EmailStore); ok {
			emailStore = typed
		}
		if typed, ok := store.(storage.MagicLinkStore); ok {
			magicLinkStore = typed
		}
	}
	return &AuthService{
		store:              store,
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

func userToProto(u user.User) *authv1.User {
	return &authv1.User{
		Id:        u.ID,
		Username:  u.Username,
		Locale:    u.Locale,
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}
}

// handleDomainError converts domain errors to gRPC status using the structured error system.
func handleDomainError(err error) error {
	return apperrors.HandleError(err, apperrors.DefaultLocale)
}
