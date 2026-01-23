package service

import (
	"context"
	"errors"
	"time"

	campaignpb "github.com/louisbranch/duality-engine/api/gen/go/campaign/v1"
	"github.com/louisbranch/duality-engine/internal/campaign/domain"
	"github.com/louisbranch/duality-engine/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultListCampaignsPageSize = 10
	maxListCampaignsPageSize     = 10
)

// CampaignService implements the CampaignService gRPC API.
type CampaignService struct {
	campaignpb.UnimplementedCampaignServiceServer
	store       storage.CampaignStore
	clock       func() time.Time
	idGenerator func() (string, error)
}

// NewCampaignService creates a CampaignService with default dependencies.
func NewCampaignService(store storage.CampaignStore) *CampaignService {
	return &CampaignService{
		store:       store,
		clock:       time.Now,
		idGenerator: domain.NewCampaignID,
	}
}

// CreateCampaign creates a new campaign metadata record.
func (s *CampaignService) CreateCampaign(ctx context.Context, in *campaignpb.CreateCampaignRequest) (*campaignpb.CreateCampaignResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create campaign request is required")
	}

	if s.store == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}

	campaign, err := domain.CreateCampaign(domain.CreateCampaignInput{
		Name:        in.GetName(),
		GmMode:      gmModeFromProto(in.GetGmMode()),
		PlayerSlots: int(in.GetPlayerSlots()),
		ThemePrompt: in.GetThemePrompt(),
	}, s.clock, s.idGenerator)
	if err != nil {
		if errors.Is(err, domain.ErrEmptyName) || errors.Is(err, domain.ErrInvalidGmMode) || errors.Is(err, domain.ErrInvalidPlayerSlots) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "create campaign: %v", err)
	}
	if err := s.store.Put(ctx, campaign); err != nil {
		return nil, status.Errorf(codes.Internal, "persist campaign: %v", err)
	}

	// TODO: Persist session state to key "session/{campaign_id}/{session_id}" when sessions exist.
	// TODO: Persist GM state to key "gm/{campaign_id}/{session_id}" when GM state is added.
	// TODO: Consider removing warnings from the gRPC response when the API stabilizes.

	response := &campaignpb.CreateCampaignResponse{
		Campaign: &campaignpb.Campaign{
			Id:          campaign.ID,
			Name:        campaign.Name,
			GmMode:      gmModeToProto(campaign.GmMode),
			PlayerSlots: int32(campaign.PlayerSlots),
			ThemePrompt: campaign.ThemePrompt,
			CreatedAt:   timestamppb.New(campaign.CreatedAt),
			UpdatedAt:   timestamppb.New(campaign.UpdatedAt),
		},
	}

	return response, nil
}

// ListCampaigns returns a page of campaign metadata records.
func (s *CampaignService) ListCampaigns(ctx context.Context, in *campaignpb.ListCampaignsRequest) (*campaignpb.ListCampaignsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list campaigns request is required")
	}

	if s.store == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}

	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = defaultListCampaignsPageSize
	}
	if pageSize > maxListCampaignsPageSize {
		pageSize = maxListCampaignsPageSize
	}

	page, err := s.store.List(ctx, pageSize, in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list campaigns: %v", err)
	}

	response := &campaignpb.ListCampaignsResponse{
		NextPageToken: page.NextPageToken,
	}
	if len(page.Campaigns) == 0 {
		return response, nil
	}

	response.Campaigns = make([]*campaignpb.Campaign, 0, len(page.Campaigns))
	for _, campaign := range page.Campaigns {
		response.Campaigns = append(response.Campaigns, &campaignpb.Campaign{
			Id:          campaign.ID,
			Name:        campaign.Name,
			GmMode:      gmModeToProto(campaign.GmMode),
			PlayerSlots: int32(campaign.PlayerSlots),
			ThemePrompt: campaign.ThemePrompt,
			CreatedAt:   timestamppb.New(campaign.CreatedAt),
			UpdatedAt:   timestamppb.New(campaign.UpdatedAt),
		})
	}

	return response, nil
}

// gmModeFromProto maps a protobuf GM mode to the domain representation.
func gmModeFromProto(mode campaignpb.GmMode) domain.GmMode {
	switch mode {
	case campaignpb.GmMode_HUMAN:
		return domain.GmModeHuman
	case campaignpb.GmMode_AI:
		return domain.GmModeAI
	case campaignpb.GmMode_HYBRID:
		return domain.GmModeHybrid
	default:
		return domain.GmModeUnspecified
	}
}

// gmModeToProto maps a domain GM mode to the protobuf representation.
func gmModeToProto(mode domain.GmMode) campaignpb.GmMode {
	switch mode {
	case domain.GmModeHuman:
		return campaignpb.GmMode_HUMAN
	case domain.GmModeAI:
		return campaignpb.GmMode_AI
	case domain.GmModeHybrid:
		return campaignpb.GmMode_HYBRID
	default:
		return campaignpb.GmMode_GM_MODE_UNSPECIFIED
	}
}
