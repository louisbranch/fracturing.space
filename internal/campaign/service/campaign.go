package service

import (
	"context"
	"errors"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/duality-engine/api/gen/go/campaign/v1"
	"github.com/louisbranch/duality-engine/internal/campaign/domain"
	"github.com/louisbranch/duality-engine/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultListCampaignsPageSize    = 10
	maxListCampaignsPageSize        = 10
	defaultListParticipantsPageSize = 10
	maxListParticipantsPageSize     = 10
	defaultListActorsPageSize       = 10
	maxListActorsPageSize           = 10
)

// Stores groups all campaign-related storage interfaces.
type Stores struct {
	Campaign    storage.CampaignStore
	Participant storage.ParticipantStore
	Actor       storage.ActorStore
}

// CampaignService implements the CampaignService gRPC API.
type CampaignService struct {
	campaignv1.UnimplementedCampaignServiceServer
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
}

// NewCampaignService creates a CampaignService with default dependencies.
func NewCampaignService(stores Stores) *CampaignService {
	return &CampaignService{
		stores:      stores,
		clock:       time.Now,
		idGenerator: domain.NewID,
	}
}

// CreateCampaign creates a new campaign metadata record.
func (s *CampaignService) CreateCampaign(ctx context.Context, in *campaignv1.CreateCampaignRequest) (*campaignv1.CreateCampaignResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create campaign request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}

	campaign, err := domain.CreateCampaign(domain.CreateCampaignInput{
		Name:        in.GetName(),
		GmMode:      gmModeFromProto(in.GetGmMode()),
		ThemePrompt: in.GetThemePrompt(),
	}, s.clock, s.idGenerator)
	if err != nil {
		if errors.Is(err, domain.ErrEmptyName) || errors.Is(err, domain.ErrInvalidGmMode) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "create campaign: %v", err)
	}
	if err := s.stores.Campaign.Put(ctx, campaign); err != nil {
		return nil, status.Errorf(codes.Internal, "persist campaign: %v", err)
	}

	// TODO: Persist session state to key "session/{campaign_id}/{session_id}" when sessions exist.
	// TODO: Persist GM state to key "gm/{campaign_id}/{session_id}" when GM state is added.
	// TODO: Consider removing warnings from the gRPC response when the API stabilizes.

	response := &campaignv1.CreateCampaignResponse{
		Campaign: &campaignv1.Campaign{
			Id:          campaign.ID,
			Name:        campaign.Name,
			GmMode:      gmModeToProto(campaign.GmMode),
			PlayerCount: int32(campaign.PlayerCount),
			ThemePrompt: campaign.ThemePrompt,
			CreatedAt:   timestamppb.New(campaign.CreatedAt),
			UpdatedAt:   timestamppb.New(campaign.UpdatedAt),
		},
	}

	return response, nil
}

// ListCampaigns returns a page of campaign metadata records.
func (s *CampaignService) ListCampaigns(ctx context.Context, in *campaignv1.ListCampaignsRequest) (*campaignv1.ListCampaignsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list campaigns request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}

	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = defaultListCampaignsPageSize
	}
	if pageSize > maxListCampaignsPageSize {
		pageSize = maxListCampaignsPageSize
	}

	page, err := s.stores.Campaign.List(ctx, pageSize, in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list campaigns: %v", err)
	}

	response := &campaignv1.ListCampaignsResponse{
		NextPageToken: page.NextPageToken,
	}
	if len(page.Campaigns) == 0 {
		return response, nil
	}

	response.Campaigns = make([]*campaignv1.Campaign, 0, len(page.Campaigns))
	for _, campaign := range page.Campaigns {
		response.Campaigns = append(response.Campaigns, &campaignv1.Campaign{
			Id:          campaign.ID,
			Name:        campaign.Name,
			GmMode:      gmModeToProto(campaign.GmMode),
			PlayerCount: int32(campaign.PlayerCount),
			ThemePrompt: campaign.ThemePrompt,
			CreatedAt:   timestamppb.New(campaign.CreatedAt),
			UpdatedAt:   timestamppb.New(campaign.UpdatedAt),
		})
	}

	return response, nil
}

// gmModeFromProto maps a protobuf GM mode to the domain representation.
func gmModeFromProto(mode campaignv1.GmMode) domain.GmMode {
	switch mode {
	case campaignv1.GmMode_HUMAN:
		return domain.GmModeHuman
	case campaignv1.GmMode_AI:
		return domain.GmModeAI
	case campaignv1.GmMode_HYBRID:
		return domain.GmModeHybrid
	default:
		return domain.GmModeUnspecified
	}
}

// gmModeToProto maps a domain GM mode to the protobuf representation.
func gmModeToProto(mode domain.GmMode) campaignv1.GmMode {
	switch mode {
	case domain.GmModeHuman:
		return campaignv1.GmMode_HUMAN
	case domain.GmModeAI:
		return campaignv1.GmMode_AI
	case domain.GmModeHybrid:
		return campaignv1.GmMode_HYBRID
	default:
		return campaignv1.GmMode_GM_MODE_UNSPECIFIED
	}
}

// CreateActor creates an actor (PC/NPC/etc) for a campaign.
func (s *CampaignService) CreateActor(ctx context.Context, in *campaignv1.CreateActorRequest) (*campaignv1.CreateActorResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create actor request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Actor == nil {
		return nil, status.Error(codes.Internal, "actor store is not configured")
	}

	// Validate campaign exists
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	_, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		return nil, status.Errorf(codes.Internal, "check campaign: %v", err)
	}

	actor, err := domain.CreateActor(domain.CreateActorInput{
		CampaignID: campaignID,
		Name:       in.GetName(),
		Kind:       actorKindFromProto(in.GetKind()),
		Notes:      in.GetNotes(),
	}, s.clock, s.idGenerator)
	if err != nil {
		if errors.Is(err, domain.ErrEmptyActorName) || errors.Is(err, domain.ErrInvalidActorKind) || errors.Is(err, domain.ErrEmptyCampaignID) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "create actor: %v", err)
	}

	if err := s.stores.Actor.PutActor(ctx, actor); err != nil {
		return nil, status.Errorf(codes.Internal, "persist actor: %v", err)
	}

	response := &campaignv1.CreateActorResponse{
		Actor: &campaignv1.Actor{
			Id:         actor.ID,
			CampaignId: actor.CampaignID,
			Name:       actor.Name,
			Kind:       actorKindToProto(actor.Kind),
			Notes:      actor.Notes,
			CreatedAt:  timestamppb.New(actor.CreatedAt),
			UpdatedAt:  timestamppb.New(actor.UpdatedAt),
		},
	}

	return response, nil
}

// actorKindFromProto maps a protobuf actor kind to the domain representation.
func actorKindFromProto(kind campaignv1.ActorKind) domain.ActorKind {
	switch kind {
	case campaignv1.ActorKind_PC:
		return domain.ActorKindPC
	case campaignv1.ActorKind_NPC:
		return domain.ActorKindNPC
	default:
		return domain.ActorKindUnspecified
	}
}

// actorKindToProto maps a domain actor kind to the protobuf representation.
func actorKindToProto(kind domain.ActorKind) campaignv1.ActorKind {
	switch kind {
	case domain.ActorKindPC:
		return campaignv1.ActorKind_PC
	case domain.ActorKindNPC:
		return campaignv1.ActorKind_NPC
	default:
		return campaignv1.ActorKind_ACTOR_KIND_UNSPECIFIED
	}
}

// ListActors returns a page of actor records for a campaign.
func (s *CampaignService) ListActors(ctx context.Context, in *campaignv1.ListActorsRequest) (*campaignv1.ListActorsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list actors request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Actor == nil {
		return nil, status.Error(codes.Internal, "actor store is not configured")
	}

	// Validate campaign exists
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	_, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		return nil, status.Errorf(codes.Internal, "check campaign: %v", err)
	}

	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = defaultListActorsPageSize
	}
	if pageSize > maxListActorsPageSize {
		pageSize = maxListActorsPageSize
	}

	page, err := s.stores.Actor.ListActors(ctx, campaignID, pageSize, in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list actors: %v", err)
	}

	response := &campaignv1.ListActorsResponse{
		NextPageToken: page.NextPageToken,
	}
	if len(page.Actors) == 0 {
		return response, nil
	}

	response.Actors = make([]*campaignv1.Actor, 0, len(page.Actors))
	for _, actor := range page.Actors {
		response.Actors = append(response.Actors, &campaignv1.Actor{
			Id:         actor.ID,
			CampaignId: actor.CampaignID,
			Name:       actor.Name,
			Kind:       actorKindToProto(actor.Kind),
			Notes:      actor.Notes,
			CreatedAt:  timestamppb.New(actor.CreatedAt),
			UpdatedAt:  timestamppb.New(actor.UpdatedAt),
		})
	}

	return response, nil
}
