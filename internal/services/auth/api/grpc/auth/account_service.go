package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
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

	profile := storage.AccountProfile{
		UserID:    userID,
		Locale:    platformi18n.NormalizeLocale(baseUser.Locale),
		CreatedAt: baseUser.CreatedAt,
		UpdatedAt: baseUser.UpdatedAt,
	}

	if s.profileStore != nil {
		stored, getErr := s.profileStore.GetAccountProfile(ctx, userID)
		switch {
		case getErr == nil:
			profile.Name = stored.Name
			profile.AvatarSetID = stored.AvatarSetID
			profile.AvatarAssetID = stored.AvatarAssetID
			profile.CreatedAt = stored.CreatedAt
			profile.UpdatedAt = stored.UpdatedAt
		case !errors.Is(getErr, storage.ErrNotFound):
			return nil, handleDomainError(getErr)
		}
	}

	return &authv1.GetProfileResponse{Profile: accountProfileToProto(profile)}, nil
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

	profile := storage.AccountProfile{
		UserID:    userID,
		Locale:    normalizedLocale,
		CreatedAt: baseUser.CreatedAt,
		UpdatedAt: baseUser.UpdatedAt,
	}

	if s.profileStore != nil {
		existingProfile, getErr := s.profileStore.GetAccountProfile(ctx, userID)
		if getErr != nil && !errors.Is(getErr, storage.ErrNotFound) {
			return nil, handleDomainError(getErr)
		}
		existingProfileFound := getErr == nil

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

		nextProfile, profileErr := account.NewProfile(account.ProfileInput{
			UserID:        userID,
			Name:          in.GetName(),
			Locale:        normalizedLocale,
			AvatarSetID:   avatarSetID,
			AvatarAssetID: avatarAssetID,
		}, now)
		if profileErr != nil {
			if errors.Is(profileErr, account.ErrEmptyUserID) ||
				errors.Is(profileErr, account.ErrAvatarSetInvalid) ||
				errors.Is(profileErr, account.ErrAvatarAssetInvalid) {
				return nil, status.Error(codes.InvalidArgument, profileErr.Error())
			}
			return nil, apperrors.HandleError(profileErr, apperrors.DefaultLocale)
		}

		storageProfile := storage.AccountProfile{
			UserID:        nextProfile.UserID,
			Name:          nextProfile.Name,
			Locale:        normalizedLocale,
			AvatarSetID:   nextProfile.AvatarSetID,
			AvatarAssetID: nextProfile.AvatarAssetID,
			CreatedAt:     nextProfile.CreatedAt,
			UpdatedAt:     nextProfile.UpdatedAt,
		}
		storageProfile.CreatedAt = existingProfile.CreatedAt
		if existingProfile.CreatedAt.IsZero() {
			storageProfile.CreatedAt = now()
		}
		storageProfile.UpdatedAt = now()

		if err := s.profileStore.PutAccountProfile(ctx, storageProfile); err != nil {
			return nil, handleDomainError(err)
		}

		stored, getErr := s.profileStore.GetAccountProfile(ctx, userID)
		if getErr != nil {
			return nil, handleDomainError(getErr)
		}
		profile.Name = stored.Name
		profile.AvatarSetID = stored.AvatarSetID
		profile.AvatarAssetID = stored.AvatarAssetID
		profile.CreatedAt = stored.CreatedAt
		profile.UpdatedAt = stored.UpdatedAt
	}

	return &authv1.UpdateProfileResponse{Profile: accountProfileToProto(profile)}, nil
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
