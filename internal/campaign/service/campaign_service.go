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
	defaultListCampaignsPageSize = 10
	maxListCampaignsPageSize     = 10
)

// CampaignService implements the CampaignService gRPC API.
type CampaignService struct {
	campaignv1.UnimplementedCampaignServiceServer
	store            storage.CampaignStore
	participantStore storage.ParticipantStore
	clock            func() time.Time
	idGenerator      func() (string, error)
	participantIDGen func() (string, error)
}

// NewCampaignService creates a CampaignService with default dependencies.
func NewCampaignService(store storage.CampaignStore, participantStore storage.ParticipantStore) *CampaignService {
	return &CampaignService{
		store:            store,
		participantStore: participantStore,
		clock:             time.Now,
		idGenerator:       domain.NewID,
		participantIDGen: domain.NewID,
	}
}

// CreateCampaign creates a new campaign metadata record.
func (s *CampaignService) CreateCampaign(ctx context.Context, in *campaignv1.CreateCampaignRequest) (*campaignv1.CreateCampaignResponse, error) {
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

	response := &campaignv1.CreateCampaignResponse{
		Campaign: &campaignv1.Campaign{
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
func (s *CampaignService) ListCampaigns(ctx context.Context, in *campaignv1.ListCampaignsRequest) (*campaignv1.ListCampaignsResponse, error) {
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
			PlayerSlots: int32(campaign.PlayerSlots),
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

// RegisterParticipant registers a participant (GM or player) for a campaign.
func (s *CampaignService) RegisterParticipant(ctx context.Context, in *campaignv1.RegisterParticipantRequest) (*campaignv1.RegisterParticipantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "register participant request is required")
	}

	if s.store == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.participantStore == nil {
		return nil, status.Error(codes.Internal, "participant store is not configured")
	}

	// Validate campaign exists
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	_, err := s.store.Get(ctx, campaignID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		return nil, status.Errorf(codes.Internal, "check campaign: %v", err)
	}

	participant, err := domain.CreateParticipant(domain.CreateParticipantInput{
		CampaignID:  campaignID,
		DisplayName: in.GetDisplayName(),
		Role:        participantRoleFromProto(in.GetRole()),
		Controller:  controllerFromProto(in.GetController()),
	}, s.clock, s.participantIDGen)
	if err != nil {
		if errors.Is(err, domain.ErrEmptyDisplayName) || errors.Is(err, domain.ErrInvalidParticipantRole) || errors.Is(err, domain.ErrEmptyCampaignID) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "create participant: %v", err)
	}

	if err := s.participantStore.PutParticipant(ctx, participant); err != nil {
		return nil, status.Errorf(codes.Internal, "persist participant: %v", err)
	}

	response := &campaignv1.RegisterParticipantResponse{
		Participant: &campaignv1.Participant{
			Id:          participant.ID,
			CampaignId:  participant.CampaignID,
			DisplayName: participant.DisplayName,
			Role:        participantRoleToProto(participant.Role),
			Controller:  controllerToProto(participant.Controller),
			CreatedAt:   timestamppb.New(participant.CreatedAt),
			UpdatedAt:   timestamppb.New(participant.UpdatedAt),
		},
	}

	return response, nil
}

// participantRoleFromProto maps a protobuf participant role to the domain representation.
func participantRoleFromProto(role campaignv1.ParticipantRole) domain.ParticipantRole {
	switch role {
	case campaignv1.ParticipantRole_GM:
		return domain.ParticipantRoleGM
	case campaignv1.ParticipantRole_PLAYER:
		return domain.ParticipantRolePlayer
	default:
		return domain.ParticipantRoleUnspecified
	}
}

// participantRoleToProto maps a domain participant role to the protobuf representation.
func participantRoleToProto(role domain.ParticipantRole) campaignv1.ParticipantRole {
	switch role {
	case domain.ParticipantRoleGM:
		return campaignv1.ParticipantRole_GM
	case domain.ParticipantRolePlayer:
		return campaignv1.ParticipantRole_PLAYER
	default:
		return campaignv1.ParticipantRole_ROLE_UNSPECIFIED
	}
}

// controllerFromProto maps a protobuf controller to the domain representation.
func controllerFromProto(controller campaignv1.Controller) domain.Controller {
	switch controller {
	case campaignv1.Controller_CONTROLLER_HUMAN:
		return domain.ControllerHuman
	case campaignv1.Controller_CONTROLLER_AI:
		return domain.ControllerAI
	default:
		return domain.ControllerUnspecified
	}
}

// controllerToProto maps a domain controller to the protobuf representation.
func controllerToProto(controller domain.Controller) campaignv1.Controller {
	switch controller {
	case domain.ControllerHuman:
		return campaignv1.Controller_CONTROLLER_HUMAN
	case domain.ControllerAI:
		return campaignv1.Controller_CONTROLLER_AI
	default:
		return campaignv1.Controller_CONTROLLER_UNSPECIFIED
	}
}
