package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/auth/account"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AccountService exposes account profile operations separate from identity auth APIs.
type AccountService struct {
	authv1.UnimplementedAccountServiceServer
	profileStore storage.AccountProfileStore
	userStore    storage.UserStore
	clock        func() time.Time
}

// NewAccountService creates the account profile service.
func NewAccountService(profileStore storage.AccountProfileStore, userStore storage.UserStore) *AccountService {
	return &AccountService{
		profileStore: profileStore,
		userStore:    userStore,
		clock:        time.Now,
	}
}

// GetProfile returns account profile metadata for a specific user ID.
func (s *AccountService) GetProfile(ctx context.Context, in *authv1.GetProfileRequest) (*authv1.GetProfileResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get profile request is required")
	}
	if s.profileStore == nil || s.userStore == nil {
		return nil, status.Error(codes.Internal, "account store is not configured")
	}

	userID := strings.TrimSpace(in.GetUserId())
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}

	if _, err := s.userStore.GetUser(ctx, userID); err != nil {
		return nil, handleDomainError(err)
	}

	profile, err := s.profileStore.GetAccountProfile(ctx, userID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	return &authv1.GetProfileResponse{Profile: accountProfileToProto(profile)}, nil
}

// UpdateProfile creates or updates profile metadata for a user.
func (s *AccountService) UpdateProfile(ctx context.Context, in *authv1.UpdateProfileRequest) (*authv1.UpdateProfileResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update profile request is required")
	}
	if s.profileStore == nil || s.userStore == nil {
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

	if _, err := s.userStore.GetUser(ctx, userID); err != nil {
		return nil, handleDomainError(err)
	}

	existingProfile, err := s.profileStore.GetAccountProfile(ctx, userID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, handleDomainError(err)
	}
	existingProfileFound := err == nil

	avatarSetID := in.GetAvatarSetId()
	avatarAssetID := in.GetAvatarAssetId()
	if existingProfileFound {
		if strings.TrimSpace(avatarSetID) == "" {
			avatarSetID = existingProfile.AvatarSetID
		}
		if strings.TrimSpace(avatarAssetID) == "" {
			avatarAssetID = existingProfile.AvatarAssetID
		}
	}

	profile, err := account.NewProfile(account.ProfileInput{
		UserID:        userID,
		Name:          in.GetName(),
		Locale:        in.GetLocale(),
		AvatarSetID:   avatarSetID,
		AvatarAssetID: avatarAssetID,
	}, now)
	if err != nil {
		if errors.Is(err, account.ErrEmptyUserID) ||
			errors.Is(err, account.ErrAvatarSetInvalid) ||
			errors.Is(err, account.ErrAvatarAssetInvalid) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, apperrors.HandleError(err, apperrors.DefaultLocale)
	}
	storageProfile := storage.AccountProfile{
		UserID:        profile.UserID,
		Name:          profile.Name,
		Locale:        profile.Locale,
		AvatarSetID:   profile.AvatarSetID,
		AvatarAssetID: profile.AvatarAssetID,
		CreatedAt:     profile.CreatedAt,
		UpdatedAt:     profile.UpdatedAt,
	}
	storageProfile.CreatedAt = existingProfile.CreatedAt
	if existingProfile.CreatedAt.IsZero() {
		storageProfile.CreatedAt = now()
	}
	storageProfile.UpdatedAt = now()

	if err := s.profileStore.PutAccountProfile(ctx, storageProfile); err != nil {
		return nil, handleDomainError(err)
	}

	stored, err := s.profileStore.GetAccountProfile(ctx, userID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	return &authv1.UpdateProfileResponse{Profile: accountProfileToProto(stored)}, nil
}

func accountProfileToProto(profile storage.AccountProfile) *authv1.AccountProfile {
	return &authv1.AccountProfile{
		UserId:        profile.UserID,
		Name:          profile.Name,
		Locale:        profile.Locale,
		AvatarSetId:   profile.AvatarSetID,
		AvatarAssetId: profile.AvatarAssetID,
		CreatedAt:     timestamppb.New(profile.CreatedAt),
		UpdatedAt:     timestamppb.New(profile.UpdatedAt),
	}
}
