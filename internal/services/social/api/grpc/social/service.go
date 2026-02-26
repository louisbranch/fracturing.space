package social

import (
	"context"
	"errors"
	"strings"
	"time"

	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	profileutil "github.com/louisbranch/fracturing.space/internal/services/social/profile"
	"github.com/louisbranch/fracturing.space/internal/services/social/storage"
	usernameutil "github.com/louisbranch/fracturing.space/internal/services/social/username"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultListContactsPageSize = 10
	maxListContactsPageSize     = 50
)

type contactAndUserProfileStore interface {
	storage.ContactStore
	storage.UserProfileStore
}

// Service exposes social.v1 gRPC operations.
type Service struct {
	socialv1.UnimplementedSocialServiceServer
	store contactAndUserProfileStore
	clock func() time.Time
}

// NewService creates a social service backed by contact storage.
func NewService(store contactAndUserProfileStore) *Service {
	return &Service{
		store: store,
		clock: time.Now,
	}
}

// AddContact adds one owner-scoped directed contact relationship.
func (s *Service) AddContact(ctx context.Context, in *socialv1.AddContactRequest) (*socialv1.AddContactResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "add contact request is required")
	}
	if s == nil || s.store == nil {
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
	if err := s.store.PutContact(ctx, contact); err != nil {
		return nil, status.Errorf(codes.Internal, "add contact: %v", err)
	}
	persisted, err := s.store.GetContact(ctx, ownerUserID, contactUserID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "add contact: %v", err)
	}
	return &socialv1.AddContactResponse{
		Contact: contactToProto(persisted),
	}, nil
}

// RemoveContact removes one owner-scoped directed contact relationship.
func (s *Service) RemoveContact(ctx context.Context, in *socialv1.RemoveContactRequest) (*socialv1.RemoveContactResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "remove contact request is required")
	}
	if s == nil || s.store == nil {
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

	if err := s.store.DeleteContact(ctx, ownerUserID, contactUserID); err != nil {
		return nil, status.Errorf(codes.Internal, "remove contact: %v", err)
	}
	return &socialv1.RemoveContactResponse{}, nil
}

// ListContacts returns one page of owner-scoped directed contacts.
func (s *Service) ListContacts(ctx context.Context, in *socialv1.ListContactsRequest) (*socialv1.ListContactsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list contacts request is required")
	}
	if s == nil || s.store == nil {
		return nil, status.Error(codes.Internal, "contact store is not configured")
	}

	ownerUserID := strings.TrimSpace(in.GetOwnerUserId())
	if ownerUserID == "" {
		return nil, status.Error(codes.InvalidArgument, "owner user id is required")
	}
	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListContactsPageSize,
		Max:     maxListContactsPageSize,
	})
	page, err := s.store.ListContacts(ctx, ownerUserID, pageSize, in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list contacts: %v", err)
	}

	resp := &socialv1.ListContactsResponse{
		Contacts:      make([]*socialv1.Contact, 0, len(page.Contacts)),
		NextPageToken: page.NextPageToken,
	}
	for _, contact := range page.Contacts {
		resp.Contacts = append(resp.Contacts, contactToProto(contact))
	}
	return resp, nil
}

// SetUserProfile claims or updates one social/discovery profile for a user.
func (s *Service) SetUserProfile(ctx context.Context, in *socialv1.SetUserProfileRequest) (*socialv1.SetUserProfileResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "set user profile request is required")
	}
	if s == nil || s.store == nil {
		return nil, status.Error(codes.Internal, "contact store is not configured")
	}
	userID := strings.TrimSpace(in.GetUserId())
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}

	canonicalUsername, err := canonicalizeOptionalUsername(in.GetUsername())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "username is invalid: %v", err)
	}
	normalized, err := profileutil.Normalize(userID, in.GetName(), in.GetAvatarSetId(), in.GetAvatarAssetId(), in.GetBio(), in.GetPronouns())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "user profile is invalid: %v", err)
	}

	now := time.Now()
	if s.clock != nil {
		now = s.clock()
	}
	if err := s.store.PutUserProfile(ctx, storage.UserProfile{
		UserID:        userID,
		Username:      canonicalUsername,
		Name:          normalized.Name,
		AvatarSetID:   normalized.AvatarSetID,
		AvatarAssetID: normalized.AvatarAssetID,
		Bio:           normalized.Bio,
		Pronouns:      normalized.Pronouns,
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		if errors.Is(err, storage.ErrAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "username is already claimed")
		}
		return nil, status.Errorf(codes.Internal, "set user profile: %v", err)
	}
	record, err := s.store.GetUserProfileByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "user profile not found")
		}
		return nil, status.Errorf(codes.Internal, "set user profile: %v", err)
	}
	return &socialv1.SetUserProfileResponse{
		UserProfile: userProfileToProto(record),
	}, nil
}

// GetUserProfile fetches one social/discovery profile by owner user ID.
func (s *Service) GetUserProfile(ctx context.Context, in *socialv1.GetUserProfileRequest) (*socialv1.GetUserProfileResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get user profile request is required")
	}
	if s == nil || s.store == nil {
		return nil, status.Error(codes.Internal, "contact store is not configured")
	}
	userID := strings.TrimSpace(in.GetUserId())
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}
	record, err := s.store.GetUserProfileByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "user profile not found")
		}
		return nil, status.Errorf(codes.Internal, "get user profile: %v", err)
	}
	return &socialv1.GetUserProfileResponse{
		UserProfile: userProfileToProto(record),
	}, nil
}

// LookupUserProfile resolves one canonical username to its profile record.
func (s *Service) LookupUserProfile(ctx context.Context, in *socialv1.LookupUserProfileRequest) (*socialv1.LookupUserProfileResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "lookup user profile request is required")
	}
	if s == nil || s.store == nil {
		return nil, status.Error(codes.Internal, "contact store is not configured")
	}
	canonicalUsername, err := usernameutil.Canonicalize(in.GetUsername())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "username is invalid: %v", err)
	}
	record, err := s.store.GetUserProfileByUsername(ctx, canonicalUsername)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "username not found")
		}
		return nil, status.Errorf(codes.Internal, "lookup user profile: %v", err)
	}
	return &socialv1.LookupUserProfileResponse{
		UserProfile: userProfileToProto(record),
	}, nil
}

func contactToProto(contact storage.Contact) *socialv1.Contact {
	return &socialv1.Contact{
		OwnerUserId:   contact.OwnerUserID,
		ContactUserId: contact.ContactUserID,
		CreatedAt:     timestamppb.New(contact.CreatedAt),
		UpdatedAt:     timestamppb.New(contact.UpdatedAt),
	}
}

func userProfileToProto(profile storage.UserProfile) *socialv1.UserProfile {
	return &socialv1.UserProfile{
		UserId:        profile.UserID,
		Username:      profile.Username,
		Name:          profile.Name,
		AvatarSetId:   profile.AvatarSetID,
		AvatarAssetId: profile.AvatarAssetID,
		Bio:           profile.Bio,
		Pronouns:      profile.Pronouns,
		CreatedAt:     timestamppb.New(profile.CreatedAt),
		UpdatedAt:     timestamppb.New(profile.UpdatedAt),
	}
}

func canonicalizeOptionalUsername(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	return usernameutil.Canonicalize(value)
}
