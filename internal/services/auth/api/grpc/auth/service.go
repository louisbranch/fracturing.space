package auth

import (
	"context"
	"crypto/rand"
	"io"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/auth/oauth"
	"github.com/louisbranch/fracturing.space/internal/services/auth/passkey"
	"github.com/louisbranch/fracturing.space/internal/services/auth/recoverycode"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	authusername "github.com/louisbranch/fracturing.space/internal/services/auth/username"
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
	webSessionStore    storage.WebSessionStore
	oauthStore         *oauth.Store
	passkeyConfig      passkey.Config
	passkeyWebAuthn    passkeyProvider
	passkeyInitErr     error
	passkeyParser      passkeyParser
	clock              func() time.Time
	idGenerator        func() (string, error)
	passkeyIDGenerator func() (string, error)
	randReader         io.Reader
}

const defaultWebSessionTTL = 24 * time.Hour
const defaultRecoverySessionTTL = 10 * time.Minute
const defaultPendingSignupTTL = 30 * time.Minute

// NewAuthService builds a service with defaults for the auth package.
//
// Defaults are intentionally assembled here so transport handlers can treat this
// as the canonical auth domain entrypoint.
func NewAuthService(store storage.UserStore, passkeyStore storage.PasskeyStore, oauthStore *oauth.Store) *AuthService {
	config := passkey.LoadConfigFromEnv()
	webAuthn, err := webauthn.New(&webauthn.Config{
		RPDisplayName: config.RPDisplayName,
		RPID:          config.RPID,
		RPOrigins:     config.RPOrigins,
	})
	var webSessionStore storage.WebSessionStore
	if store != nil {
		if typed, ok := store.(storage.WebSessionStore); ok {
			webSessionStore = typed
		}
	}
	return &AuthService{
		store:              store,
		passkeyStore:       passkeyStore,
		webSessionStore:    webSessionStore,
		oauthStore:         oauthStore,
		passkeyConfig:      config,
		passkeyWebAuthn:    webAuthn,
		passkeyInitErr:     err,
		passkeyParser:      defaultPasskeyParser{},
		clock:              time.Now,
		idGenerator:        id.NewID,
		passkeyIDGenerator: id.NewID,
		randReader:         rand.Reader,
	}
}

// IssueJoinGrant issues a one-time join grant for campaign invites.
func (s *AuthService) IssueJoinGrant(ctx context.Context, in *authv1.IssueJoinGrantRequest) (*authv1.IssueJoinGrantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "Issue join grant request is required.")
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

// LookupUserByUsername resolves a username to its account record.
func (s *AuthService) LookupUserByUsername(ctx context.Context, in *authv1.LookupUserByUsernameRequest) (*authv1.LookupUserByUsernameResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "Lookup user by username request is required.")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "User store is not configured.")
	}
	username := strings.TrimSpace(in.GetUsername())
	if username == "" {
		return nil, status.Error(codes.InvalidArgument, "Username is required.")
	}
	found, err := s.store.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, handleDomainError(err)
	}
	return &authv1.LookupUserByUsernameResponse{User: userToProto(found)}, nil
}

// CheckUsernameAvailability reports whether a username is valid and available.
func (s *AuthService) CheckUsernameAvailability(ctx context.Context, in *authv1.CheckUsernameAvailabilityRequest) (*authv1.CheckUsernameAvailabilityResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "Check username availability request is required.")
	}
	if s.store == nil || s.passkeyStore == nil {
		return nil, status.Error(codes.Internal, "Auth stores are not configured.")
	}

	canonicalUsername, normalized := normalizeUsernameCandidate(in.GetUsername())
	if canonicalUsername == "" || !normalized {
		return &authv1.CheckUsernameAvailabilityResponse{
			State: authv1.UsernameAvailabilityState_USERNAME_AVAILABILITY_STATE_INVALID,
		}, nil
	}
	if _, err := authusername.Canonicalize(canonicalUsername); err != nil {
		return &authv1.CheckUsernameAvailabilityResponse{
			CanonicalUsername: canonicalUsername,
			State:             authv1.UsernameAvailabilityState_USERNAME_AVAILABILITY_STATE_INVALID,
		}, nil
	}

	_, err := s.store.GetUserByUsername(ctx, canonicalUsername)
	if err == nil {
		return &authv1.CheckUsernameAvailabilityResponse{
			CanonicalUsername: canonicalUsername,
			State:             authv1.UsernameAvailabilityState_USERNAME_AVAILABILITY_STATE_UNAVAILABLE,
		}, nil
	}
	if err == storage.ErrNotFound {
		if err := s.passkeyStore.DeleteExpiredRegistrationSessions(ctx, s.clock().UTC()); err != nil {
			return nil, status.Errorf(codes.Internal, "Delete expired registration sessions: %v", err)
		}
		if err := s.passkeyStore.DeleteExpiredPasskeySessions(ctx, s.clock().UTC()); err != nil {
			return nil, status.Errorf(codes.Internal, "Delete expired passkey sessions: %v", err)
		}
		if _, err := s.passkeyStore.GetRegistrationSessionByUsername(ctx, canonicalUsername); err == nil {
			return &authv1.CheckUsernameAvailabilityResponse{
				CanonicalUsername: canonicalUsername,
				State:             authv1.UsernameAvailabilityState_USERNAME_AVAILABILITY_STATE_UNAVAILABLE,
			}, nil
		} else if err != storage.ErrNotFound {
			return nil, status.Errorf(codes.Internal, "Get registration session by username: %v", err)
		}
		return &authv1.CheckUsernameAvailabilityResponse{
			CanonicalUsername: canonicalUsername,
			State:             authv1.UsernameAvailabilityState_USERNAME_AVAILABILITY_STATE_AVAILABLE,
		}, nil
	}
	return nil, handleDomainError(err)
}

// GetUser resolves a user ID to an identity record for cross-service lookups.
func (s *AuthService) GetUser(ctx context.Context, in *authv1.GetUserRequest) (*authv1.GetUserResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "Get user request is required.")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "User store is not configured.")
	}

	userID := strings.TrimSpace(in.GetUserId())
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "User ID is required.")
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
		return nil, status.Error(codes.InvalidArgument, "List users request is required.")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "User store is not configured.")
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListUsersPageSize,
		Max:     maxListUsersPageSize,
	})

	page, err := s.store.ListUsers(ctx, pageSize, in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "List users: %v", err)
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

func normalizeUsernameCandidate(input string) (string, bool) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", false
	}
	var builder strings.Builder
	builder.Grow(len(input))
	for i := 0; i < len(input); i++ {
		ch := input[i]
		if ch > 0x7f {
			return "", false
		}
		if ch >= 'A' && ch <= 'Z' {
			ch = ch - 'A' + 'a'
		}
		builder.WriteByte(ch)
	}
	return builder.String(), true
}

func (s *AuthService) generateRecoveryCode() (string, string, error) {
	if s == nil {
		return "", "", status.Error(codes.Internal, "Auth service is required.")
	}
	return recoverycode.Generate(s.randReader)
}

// handleDomainError converts domain errors to gRPC status using the structured error system.
func handleDomainError(err error) error {
	return apperrors.HandleError(err, apperrors.DefaultLocale)
}
