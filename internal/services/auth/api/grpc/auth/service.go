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

// NewAuthService creates an AuthService with default dependencies.
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

// CreateUser creates a new user record.
func (s *AuthService) CreateUser(ctx context.Context, in *authv1.CreateUserRequest) (*authv1.CreateUserResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create user request is required")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "user store is not configured")
	}

	created, err := user.CreateUser(user.CreateUserInput{
		DisplayName: in.GetDisplayName(),
		Locale:      in.GetLocale(),
	}, s.clock, s.idGenerator)
	if err != nil {
		return nil, handleDomainError(err)
	}

	if err := s.store.PutUser(ctx, created); err != nil {
		return nil, status.Errorf(codes.Internal, "put user: %v", err)
	}

	return &authv1.CreateUserResponse{User: userToProto(created)}, nil
}

// IssueJoinGrant issues a join grant for a campaign invite.
func (s *AuthService) IssueJoinGrant(ctx context.Context, in *authv1.IssueJoinGrantRequest) (*authv1.IssueJoinGrantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "issue join grant request is required")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "user store is not configured")
	}

	userID := strings.TrimSpace(in.GetUserId())
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	inviteID := strings.TrimSpace(in.GetInviteId())
	if inviteID == "" {
		return nil, status.Error(codes.InvalidArgument, "invite id is required")
	}
	participantID := strings.TrimSpace(in.GetParticipantId())

	if _, err := s.store.GetUser(ctx, userID); err != nil {
		return nil, handleDomainError(err)
	}

	config, err := loadJoinGrantConfigFromEnv()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "join grant config: %v", err)
	}

	issuedAt := s.clock().UTC()
	expiresAt := issuedAt.Add(config.ttl)
	jti, err := id.NewID()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate join grant id: %v", err)
	}

	payload := map[string]any{
		"iss":         config.issuer,
		"aud":         config.audience,
		"sub":         userID,
		"exp":         expiresAt.Unix(),
		"iat":         issuedAt.Unix(),
		"jti":         jti,
		"campaign_id": campaignID,
		"invite_id":   inviteID,
		"user_id":     userID,
	}
	if participantID != "" {
		payload["participant_id"] = participantID
	}

	grant, err := encodeJoinGrant(config, payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "sign join grant: %v", err)
	}

	return &authv1.IssueJoinGrantResponse{
		JoinGrant: grant,
		Jti:       jti,
		ExpiresAt: timestamppb.New(expiresAt),
	}, nil
}

// GetUser returns a user record by ID.
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

// ListUsers returns a page of user records.
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
		Id:          u.ID,
		DisplayName: u.DisplayName,
		Locale:      u.Locale,
		CreatedAt:   timestamppb.New(u.CreatedAt),
		UpdatedAt:   timestamppb.New(u.UpdatedAt),
	}
}

// handleDomainError converts domain errors to gRPC status using the structured error system.
func handleDomainError(err error) error {
	return apperrors.HandleError(err, apperrors.DefaultLocale)
}
