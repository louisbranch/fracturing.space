package service

import (
	"context"
	"errors"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/duality-engine/api/gen/go/campaign/v1"
	"github.com/louisbranch/duality-engine/internal/campaign/domain"
	"github.com/louisbranch/duality-engine/internal/id"
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
	Campaign       storage.CampaignStore
	Participant    storage.ParticipantStore
	Actor          storage.ActorStore
	ControlDefault storage.ControlDefaultStore
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
		idGenerator: id.NewID,
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

// GetCampaign returns a campaign metadata record by ID.
func (s *CampaignService) GetCampaign(ctx context.Context, in *campaignv1.GetCampaignRequest) (*campaignv1.GetCampaignResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get campaign request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	campaign, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		return nil, status.Errorf(codes.Internal, "get campaign: %v", err)
	}

	response := &campaignv1.GetCampaignResponse{
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

// SetDefaultControl assigns a campaign-scoped default controller for an actor.
func (s *CampaignService) SetDefaultControl(ctx context.Context, in *campaignv1.SetDefaultControlRequest) (*campaignv1.SetDefaultControlResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "set default control request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Actor == nil {
		return nil, status.Error(codes.Internal, "actor store is not configured")
	}
	if s.stores.ControlDefault == nil {
		return nil, status.Error(codes.Internal, "control default store is not configured")
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

	// Validate actor exists
	actorID := strings.TrimSpace(in.GetActorId())
	if actorID == "" {
		return nil, status.Error(codes.InvalidArgument, "actor id is required")
	}
	_, err = s.stores.Actor.GetActor(ctx, campaignID, actorID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "actor not found")
		}
		return nil, status.Errorf(codes.Internal, "check actor: %v", err)
	}

	// Validate and convert controller
	if in.GetController() == nil {
		return nil, status.Error(codes.InvalidArgument, "controller is required")
	}
	controller, err := actorControllerFromProto(in.GetController())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// If participant controller, validate participant exists
	if !controller.IsGM {
		if s.stores.Participant == nil {
			return nil, status.Error(codes.Internal, "participant store is not configured")
		}
		_, err = s.stores.Participant.GetParticipant(ctx, campaignID, controller.ParticipantID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return nil, status.Error(codes.NotFound, "participant not found")
			}
			return nil, status.Errorf(codes.Internal, "check participant: %v", err)
		}
	}

	// Persist controller
	if err := s.stores.ControlDefault.PutControlDefault(ctx, campaignID, actorID, controller); err != nil {
		return nil, status.Errorf(codes.Internal, "persist control default: %v", err)
	}

	response := &campaignv1.SetDefaultControlResponse{
		CampaignId: campaignID,
		ActorId:    actorID,
		Controller: actorControllerToProto(controller),
	}

	return response, nil
}

// actorControllerFromProto converts a protobuf ActorController to the domain representation.
func actorControllerFromProto(pb *campaignv1.ActorController) (domain.ActorController, error) {
	if pb == nil {
		return domain.ActorController{}, domain.ErrInvalidActorController
	}

	switch c := pb.GetController().(type) {
	case *campaignv1.ActorController_Gm:
		if c.Gm == nil {
			return domain.ActorController{}, domain.ErrInvalidActorController
		}
		return domain.NewGmController(), nil
	case *campaignv1.ActorController_Participant:
		if c.Participant == nil {
			return domain.ActorController{}, domain.ErrInvalidActorController
		}
		return domain.NewParticipantController(c.Participant.GetParticipantId())
	default:
		return domain.ActorController{}, domain.ErrInvalidActorController
	}
}

// actorControllerToProto converts a domain ActorController to the protobuf representation.
// The controller must be valid (exactly one of IsGM or ParticipantID set).
func actorControllerToProto(ctrl domain.ActorController) *campaignv1.ActorController {
	if ctrl.IsGM {
		return &campaignv1.ActorController{
			Controller: &campaignv1.ActorController_Gm{
				Gm: &campaignv1.GmController{},
			},
		}
	}
	// If not GM, assume participant controller (validation should ensure this is valid).
	return &campaignv1.ActorController{
		Controller: &campaignv1.ActorController_Participant{
			Participant: &campaignv1.ParticipantController{
				ParticipantId: ctrl.ParticipantID,
			},
		},
	}
}
