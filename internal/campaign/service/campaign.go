package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/campaign/v1"
	"github.com/louisbranch/fracturing.space/internal/campaign/domain"
	"github.com/louisbranch/fracturing.space/internal/id"
	"github.com/louisbranch/fracturing.space/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultListCampaignsPageSize    = 10
	maxListCampaignsPageSize        = 10
	defaultListParticipantsPageSize = 10
	maxListParticipantsPageSize     = 10
	defaultListCharactersPageSize   = 10
	maxListCharactersPageSize       = 10
)

// Stores groups all campaign-related storage interfaces.
type Stores struct {
	Campaign         storage.CampaignStore
	Participant      storage.ParticipantStore
	Character        storage.CharacterStore
	CharacterProfile storage.CharacterProfileStore
	CharacterState   storage.CharacterStateStore
	ControlDefault   storage.ControlDefaultStore
}

// CampaignService implements the CampaignService gRPC API.
type CampaignService struct {
	campaignv1.UnimplementedCampaignServiceServer
	stores       Stores
	clock        func() time.Time
	idGenerator  func() (string, error)
	defaults     map[domain.CharacterKind]domain.CharacterProfileDefaults
	defaultsOnce sync.Once
	defaultsErr  error
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
			Id:               campaign.ID,
			Name:             campaign.Name,
			GmMode:           gmModeToProto(campaign.GmMode),
			ParticipantCount: int32(campaign.ParticipantCount),
			CharacterCount:   int32(campaign.CharacterCount),
			GmFear:           int32(campaign.GmFear),
			ThemePrompt:      campaign.ThemePrompt,
			CreatedAt:        timestamppb.New(campaign.CreatedAt),
			UpdatedAt:        timestamppb.New(campaign.UpdatedAt),
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
			Id:               campaign.ID,
			Name:             campaign.Name,
			GmMode:           gmModeToProto(campaign.GmMode),
			ParticipantCount: int32(campaign.ParticipantCount),
			CharacterCount:   int32(campaign.CharacterCount),
			GmFear:           int32(campaign.GmFear),
			ThemePrompt:      campaign.ThemePrompt,
			CreatedAt:        timestamppb.New(campaign.CreatedAt),
			UpdatedAt:        timestamppb.New(campaign.UpdatedAt),
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
			Id:               campaign.ID,
			Name:             campaign.Name,
			GmMode:           gmModeToProto(campaign.GmMode),
			ParticipantCount: int32(campaign.ParticipantCount),
			CharacterCount:   int32(campaign.CharacterCount),
			GmFear:           int32(campaign.GmFear),
			ThemePrompt:      campaign.ThemePrompt,
			CreatedAt:        timestamppb.New(campaign.CreatedAt),
			UpdatedAt:        timestamppb.New(campaign.UpdatedAt),
		},
	}

	return response, nil
}

// GMFearGain increases GM fear for a campaign and persists the change.
func (s *CampaignService) GMFearGain(ctx context.Context, campaignID string, amount int) (int, int, error) {
	if s == nil {
		return 0, 0, errors.New("campaign service is not configured")
	}
	if s.stores.Campaign == nil {
		return 0, 0, errors.New("campaign store is not configured")
	}

	trimmedCampaignID := strings.TrimSpace(campaignID)
	if trimmedCampaignID == "" {
		return 0, 0, errors.New("campaign id is required")
	}

	campaign, err := s.stores.Campaign.Get(ctx, trimmedCampaignID)
	if err != nil {
		return 0, 0, fmt.Errorf("get campaign: %w", err)
	}

	updated, before, after, err := domain.ApplyGMFearGain(campaign, amount)
	if err != nil {
		return 0, 0, err
	}

	updated.UpdatedAt = s.now().UTC()
	if err := s.stores.Campaign.Put(ctx, updated); err != nil {
		return 0, 0, fmt.Errorf("persist campaign: %w", err)
	}

	return before, after, nil
}

// GMFearSpend decreases GM fear for a campaign and persists the change.
func (s *CampaignService) GMFearSpend(ctx context.Context, campaignID string, amount int) (int, int, error) {
	if s == nil {
		return 0, 0, errors.New("campaign service is not configured")
	}
	if s.stores.Campaign == nil {
		return 0, 0, errors.New("campaign store is not configured")
	}

	trimmedCampaignID := strings.TrimSpace(campaignID)
	if trimmedCampaignID == "" {
		return 0, 0, errors.New("campaign id is required")
	}

	campaign, err := s.stores.Campaign.Get(ctx, trimmedCampaignID)
	if err != nil {
		return 0, 0, fmt.Errorf("get campaign: %w", err)
	}

	updated, before, after, err := domain.ApplyGMFearSpend(campaign, amount)
	if err != nil {
		return 0, 0, err
	}

	updated.UpdatedAt = s.now().UTC()
	if err := s.stores.Campaign.Put(ctx, updated); err != nil {
		return 0, 0, fmt.Errorf("persist campaign: %w", err)
	}

	return before, after, nil
}

func (s *CampaignService) now() time.Time {
	if s == nil || s.clock == nil {
		return time.Now()
	}
	return s.clock()
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

// CreateCharacter creates a character (PC/NPC/etc) for a campaign.
func (s *CampaignService) CreateCharacter(ctx context.Context, in *campaignv1.CreateCharacterRequest) (*campaignv1.CreateCharacterResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create character request is required")
	}

	if s.stores.Character == nil {
		return nil, status.Error(codes.Internal, "character store is not configured")
	}
	if s.stores.CharacterProfile == nil {
		return nil, status.Error(codes.Internal, "character profile store is not configured")
	}
	if s.stores.CharacterState == nil {
		return nil, status.Error(codes.Internal, "character state store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	character, err := domain.CreateCharacter(domain.CreateCharacterInput{
		CampaignID: campaignID,
		Name:       in.GetName(),
		Kind:       characterKindFromProto(in.GetKind()),
		Notes:      in.GetNotes(),
	}, s.clock, s.idGenerator)
	if err != nil {
		if errors.Is(err, domain.ErrEmptyCharacterName) || errors.Is(err, domain.ErrInvalidCharacterKind) || errors.Is(err, domain.ErrEmptyCampaignID) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "create character: %v", err)
	}

	if err := s.stores.Character.PutCharacter(ctx, character); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		return nil, status.Errorf(codes.Internal, "persist character: %v", err)
	}

	// Load defaults (cached after first load) and create profile
	s.defaultsOnce.Do(func() {
		s.defaults, s.defaultsErr = domain.LoadCharacterDefaults("")
	})
	if s.defaultsErr != nil {
		return nil, status.Errorf(codes.Internal, "load character defaults: %v", s.defaultsErr)
	}

	defaultProfile, err := domain.GetDefaultProfile(character.Kind, s.defaults)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get default profile: %v", err)
	}

	profile, err := domain.CreateCharacterProfile(domain.CreateCharacterProfileInput{
		CampaignID:      campaignID,
		CharacterID:     character.ID,
		Traits:          defaultProfile.Traits,
		HpMax:           defaultProfile.HpMax,
		StressMax:       defaultProfile.StressMax,
		Evasion:         defaultProfile.Evasion,
		MajorThreshold:  defaultProfile.MajorThreshold,
		SevereThreshold: defaultProfile.SevereThreshold,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create character profile: %v", err)
	}

	if err := s.stores.CharacterProfile.PutCharacterProfile(ctx, profile); err != nil {
		return nil, status.Errorf(codes.Internal, "persist character profile: %v", err)
	}

	// Create state with defaults
	state, err := domain.CreateCharacterState(domain.CreateCharacterStateInput{
		CampaignID:  campaignID,
		CharacterID: character.ID,
		Hope:        0,
		Stress:      0,
		Hp:          profile.HpMax,
	}, profile)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create character state: %v", err)
	}

	if err := s.stores.CharacterState.PutCharacterState(ctx, state); err != nil {
		return nil, status.Errorf(codes.Internal, "persist character state: %v", err)
	}

	response := &campaignv1.CreateCharacterResponse{
		Character: &campaignv1.Character{
			Id:         character.ID,
			CampaignId: character.CampaignID,
			Name:       character.Name,
			Kind:       characterKindToProto(character.Kind),
			Notes:      character.Notes,
			CreatedAt:  timestamppb.New(character.CreatedAt),
			UpdatedAt:  timestamppb.New(character.UpdatedAt),
		},
	}

	return response, nil
}

// characterKindFromProto maps a protobuf character kind to the domain representation.
func characterKindFromProto(kind campaignv1.CharacterKind) domain.CharacterKind {
	switch kind {
	case campaignv1.CharacterKind_PC:
		return domain.CharacterKindPC
	case campaignv1.CharacterKind_NPC:
		return domain.CharacterKindNPC
	default:
		return domain.CharacterKindUnspecified
	}
}

// characterKindToProto maps a domain character kind to the protobuf representation.
func characterKindToProto(kind domain.CharacterKind) campaignv1.CharacterKind {
	switch kind {
	case domain.CharacterKindPC:
		return campaignv1.CharacterKind_PC
	case domain.CharacterKindNPC:
		return campaignv1.CharacterKind_NPC
	default:
		return campaignv1.CharacterKind_CHARACTER_KIND_UNSPECIFIED
	}
}

// ListCharacters returns a page of character records for a campaign.
func (s *CampaignService) ListCharacters(ctx context.Context, in *campaignv1.ListCharactersRequest) (*campaignv1.ListCharactersResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list characters request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Character == nil {
		return nil, status.Error(codes.Internal, "character store is not configured")
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
		pageSize = defaultListCharactersPageSize
	}
	if pageSize > maxListCharactersPageSize {
		pageSize = maxListCharactersPageSize
	}

	page, err := s.stores.Character.ListCharacters(ctx, campaignID, pageSize, in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list characters: %v", err)
	}

	response := &campaignv1.ListCharactersResponse{
		NextPageToken: page.NextPageToken,
	}
	if len(page.Characters) == 0 {
		return response, nil
	}

	response.Characters = make([]*campaignv1.Character, 0, len(page.Characters))
	for _, character := range page.Characters {
		response.Characters = append(response.Characters, &campaignv1.Character{
			Id:         character.ID,
			CampaignId: character.CampaignID,
			Name:       character.Name,
			Kind:       characterKindToProto(character.Kind),
			Notes:      character.Notes,
			CreatedAt:  timestamppb.New(character.CreatedAt),
			UpdatedAt:  timestamppb.New(character.UpdatedAt),
		})
	}

	return response, nil
}

// SetDefaultControl assigns a campaign-scoped default controller for a character.
func (s *CampaignService) SetDefaultControl(ctx context.Context, in *campaignv1.SetDefaultControlRequest) (*campaignv1.SetDefaultControlResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "set default control request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Character == nil {
		return nil, status.Error(codes.Internal, "character store is not configured")
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

	// Validate character exists
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}
	_, err = s.stores.Character.GetCharacter(ctx, campaignID, characterID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "character not found")
		}
		return nil, status.Errorf(codes.Internal, "check character: %v", err)
	}

	// Validate and convert controller
	if in.GetController() == nil {
		return nil, status.Error(codes.InvalidArgument, "controller is required")
	}
	controller, err := characterControllerFromProto(in.GetController())
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
	if err := s.stores.ControlDefault.PutControlDefault(ctx, campaignID, characterID, controller); err != nil {
		return nil, status.Errorf(codes.Internal, "persist control default: %v", err)
	}

	response := &campaignv1.SetDefaultControlResponse{
		CampaignId:  campaignID,
		CharacterId: characterID,
		Controller:  characterControllerToProto(controller),
	}

	return response, nil
}

// characterControllerFromProto converts a protobuf CharacterController to the domain representation.
func characterControllerFromProto(pb *campaignv1.CharacterController) (domain.CharacterController, error) {
	if pb == nil {
		return domain.CharacterController{}, domain.ErrInvalidCharacterController
	}

	switch c := pb.GetController().(type) {
	case *campaignv1.CharacterController_Gm:
		if c.Gm == nil {
			return domain.CharacterController{}, domain.ErrInvalidCharacterController
		}
		return domain.NewGmController(), nil
	case *campaignv1.CharacterController_Participant:
		if c.Participant == nil {
			return domain.CharacterController{}, domain.ErrInvalidCharacterController
		}
		return domain.NewParticipantController(c.Participant.GetParticipantId())
	default:
		return domain.CharacterController{}, domain.ErrInvalidCharacterController
	}
}

// characterControllerToProto converts a domain CharacterController to the protobuf representation.
// The controller must be valid (exactly one of IsGM or ParticipantID set).
func characterControllerToProto(ctrl domain.CharacterController) *campaignv1.CharacterController {
	if ctrl.IsGM {
		return &campaignv1.CharacterController{
			Controller: &campaignv1.CharacterController_Gm{
				Gm: &campaignv1.GmController{},
			},
		}
	}
	// If not GM, assume participant controller (validation should ensure this is valid).
	return &campaignv1.CharacterController{
		Controller: &campaignv1.CharacterController_Participant{
			Participant: &campaignv1.ParticipantController{
				ParticipantId: ctrl.ParticipantID,
			},
		},
	}
}
