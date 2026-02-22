package listing

import (
	"context"
	"errors"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	listingv1 "github.com/louisbranch/fracturing.space/api/gen/go/listing/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/listing/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultListCampaignListingsPageSize = 10
	maxListCampaignListingsPageSize     = 50
)

// Service exposes listing.v1 gRPC operations.
type Service struct {
	listingv1.UnimplementedCampaignListingServiceServer
	store storage.CampaignListingStore
	clock func() time.Time
}

// NewService creates a listing service backed by campaign listing storage.
func NewService(store storage.CampaignListingStore) *Service {
	return &Service{
		store: store,
		clock: time.Now,
	}
}

// CreateCampaignListing creates one campaign listing record.
func (s *Service) CreateCampaignListing(ctx context.Context, in *listingv1.CreateCampaignListingRequest) (*listingv1.CreateCampaignListingResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create campaign listing request is required")
	}
	if s == nil || s.store == nil {
		return nil, status.Error(codes.Internal, "campaign listing store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	title := strings.TrimSpace(in.GetTitle())
	description := strings.TrimSpace(in.GetDescription())
	expectedDurationLabel := strings.TrimSpace(in.GetExpectedDurationLabel())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	if title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	if description == "" {
		return nil, status.Error(codes.InvalidArgument, "description is required")
	}
	if expectedDurationLabel == "" {
		return nil, status.Error(codes.InvalidArgument, "expected duration label is required")
	}
	if in.GetDifficultyTier() == listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "difficulty tier is required")
	}
	if in.GetSystem() == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "game system is required")
	}
	if in.GetRecommendedParticipantsMin() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "recommended participants min must be greater than zero")
	}
	if in.GetRecommendedParticipantsMax() < in.GetRecommendedParticipantsMin() {
		return nil, status.Error(codes.InvalidArgument, "recommended participants max must be greater than or equal to min")
	}

	now := time.Now().UTC()
	if s.clock != nil {
		now = s.clock().UTC()
	}
	record := storage.CampaignListing{
		CampaignID:                 campaignID,
		Title:                      title,
		Description:                description,
		RecommendedParticipantsMin: int(in.GetRecommendedParticipantsMin()),
		RecommendedParticipantsMax: int(in.GetRecommendedParticipantsMax()),
		DifficultyTier:             in.GetDifficultyTier(),
		ExpectedDurationLabel:      expectedDurationLabel,
		System:                     in.GetSystem(),
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}
	if err := s.store.CreateCampaignListing(ctx, record); err != nil {
		if errors.Is(err, storage.ErrAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "campaign listing already exists")
		}
		return nil, status.Errorf(codes.Internal, "create campaign listing: %v", err)
	}
	return &listingv1.CreateCampaignListingResponse{
		Listing: campaignListingToProto(record),
	}, nil
}

// GetCampaignListing returns one campaign listing record by campaign ID.
func (s *Service) GetCampaignListing(ctx context.Context, in *listingv1.GetCampaignListingRequest) (*listingv1.GetCampaignListingResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get campaign listing request is required")
	}
	if s == nil || s.store == nil {
		return nil, status.Error(codes.Internal, "campaign listing store is not configured")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	record, err := s.store.GetCampaignListing(ctx, campaignID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "campaign listing not found")
		}
		return nil, status.Errorf(codes.Internal, "get campaign listing: %v", err)
	}
	return &listingv1.GetCampaignListingResponse{
		Listing: campaignListingToProto(record),
	}, nil
}

// ListCampaignListings returns a page of campaign listing records.
func (s *Service) ListCampaignListings(ctx context.Context, in *listingv1.ListCampaignListingsRequest) (*listingv1.ListCampaignListingsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list campaign listings request is required")
	}
	if s == nil || s.store == nil {
		return nil, status.Error(codes.Internal, "campaign listing store is not configured")
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListCampaignListingsPageSize,
		Max:     maxListCampaignListingsPageSize,
	})
	page, err := s.store.ListCampaignListings(ctx, pageSize, in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list campaign listings: %v", err)
	}

	resp := &listingv1.ListCampaignListingsResponse{
		Listings:      make([]*listingv1.CampaignListing, 0, len(page.Listings)),
		NextPageToken: page.NextPageToken,
	}
	for _, listing := range page.Listings {
		resp.Listings = append(resp.Listings, campaignListingToProto(listing))
	}
	return resp, nil
}

func campaignListingToProto(record storage.CampaignListing) *listingv1.CampaignListing {
	return &listingv1.CampaignListing{
		CampaignId:                 record.CampaignID,
		Title:                      record.Title,
		Description:                record.Description,
		RecommendedParticipantsMin: int32(record.RecommendedParticipantsMin),
		RecommendedParticipantsMax: int32(record.RecommendedParticipantsMax),
		DifficultyTier:             record.DifficultyTier,
		ExpectedDurationLabel:      record.ExpectedDurationLabel,
		System:                     record.System,
		CreatedAt:                  timestamppb.New(record.CreatedAt),
		UpdatedAt:                  timestamppb.New(record.UpdatedAt),
	}
}
