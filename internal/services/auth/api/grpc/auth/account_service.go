package auth

import (
	"context"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AccountService exposes account profile operations separate from identity auth APIs.
type AccountService struct {
	authv1.UnimplementedAccountServiceServer
	userStore storage.UserStore
	clock     func() time.Time
}

// NewAccountService creates the account profile service.
func NewAccountService(userStore storage.UserStore) *AccountService {
	return &AccountService{
		userStore: userStore,
		clock:     time.Now,
	}
}

// GetProfile returns account profile metadata for a specific user ID.
func (s *AccountService) GetProfile(ctx context.Context, in *authv1.GetProfileRequest) (*authv1.GetProfileResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get profile request is required")
	}
	if s.userStore == nil {
		return nil, status.Error(codes.Internal, "account store is not configured")
	}

	userID := strings.TrimSpace(in.GetUserId())
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}

	baseUser, err := s.userStore.GetUser(ctx, userID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	return &authv1.GetProfileResponse{Profile: accountProfileToProto(baseUser)}, nil
}

// UpdateProfile creates or updates profile metadata for a user.
func (s *AccountService) UpdateProfile(ctx context.Context, in *authv1.UpdateProfileRequest) (*authv1.UpdateProfileResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update profile request is required")
	}
	if s.userStore == nil {
		return nil, status.Error(codes.Internal, "account store is not configured")
	}
	now := time.Now
	if s.clock != nil {
		now = s.clock
	}

	userID := strings.TrimSpace(in.GetUserId())
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}

	baseUser, err := s.userStore.GetUser(ctx, userID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	normalizedLocale := baseUser.Locale
	if in.GetLocale() != 0 || normalizedLocale == 0 {
		normalizedLocale = platformi18n.NormalizeLocale(in.GetLocale())
	}
	baseUser.Locale = normalizedLocale
	baseUser.UpdatedAt = now()
	if err := s.userStore.PutUser(ctx, baseUser); err != nil {
		return nil, handleDomainError(err)
	}

	return &authv1.UpdateProfileResponse{Profile: accountProfileToProto(baseUser)}, nil
}

func accountProfileToProto(profile user.User) *authv1.AccountProfile {
	return &authv1.AccountProfile{
		UserId:    profile.ID,
		Locale:    platformi18n.NormalizeLocale(profile.Locale),
		CreatedAt: timestamppb.New(profile.CreatedAt),
		UpdatedAt: timestamppb.New(profile.UpdatedAt),
	}
}
