package game

import (
	"context"
	"errors"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	defaultListCharactersPageSize = 10
	maxListCharactersPageSize     = 10
)

// CharacterService implements the game.v1.CharacterService gRPC API.
type CharacterService struct {
	campaignv1.UnimplementedCharacterServiceServer
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
}

// NewCharacterService creates a CharacterService with default dependencies.
func NewCharacterService(stores Stores) *CharacterService {
	return &CharacterService{
		stores:      stores,
		clock:       time.Now,
		idGenerator: id.NewID,
	}
}

// CreateCharacter creates a character (PC/NPC/etc) for a campaign.
func (s *CharacterService) CreateCharacter(ctx context.Context, in *campaignv1.CreateCharacterRequest) (*campaignv1.CreateCharacterResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create character request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	created, err := newCharacterApplication(s).CreateCharacter(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.CreateCharacterResponse{Character: characterToProto(created)}, nil
}

// UpdateCharacter updates a character's metadata.
func (s *CharacterService) UpdateCharacter(ctx context.Context, in *campaignv1.UpdateCharacterRequest) (*campaignv1.UpdateCharacterResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update character request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	updated, err := newCharacterApplication(s).UpdateCharacter(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.UpdateCharacterResponse{Character: characterToProto(updated)}, nil
}

// DeleteCharacter deletes a character.
func (s *CharacterService) DeleteCharacter(ctx context.Context, in *campaignv1.DeleteCharacterRequest) (*campaignv1.DeleteCharacterResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "delete character request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	ch, err := newCharacterApplication(s).DeleteCharacter(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.DeleteCharacterResponse{Character: characterToProto(ch)}, nil
}

// ListCharacters returns a page of character records for a campaign.
func (s *CharacterService) ListCharacters(ctx context.Context, in *campaignv1.ListCharactersRequest) (*campaignv1.ListCharactersResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list characters request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpRead); err != nil {
		return nil, handleDomainError(err)
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListCharactersPageSize,
		Max:     maxListCharactersPageSize,
	})

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
	for _, ch := range page.Characters {
		response.Characters = append(response.Characters, characterToProto(ch))
	}

	return response, nil
}

// SetDefaultControl assigns a campaign-scoped default controller for a character.
func (s *CharacterService) SetDefaultControl(ctx context.Context, in *campaignv1.SetDefaultControlRequest) (*campaignv1.SetDefaultControlResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "set default control request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	characterID, participantID, err := newCharacterApplication(s).SetDefaultControl(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	var participantIDValue *wrapperspb.StringValue
	if participantID != "" {
		participantIDValue = wrapperspb.String(participantID)
	}
	return &campaignv1.SetDefaultControlResponse{
		CampaignId:    campaignID,
		CharacterId:   characterID,
		ParticipantId: participantIDValue,
	}, nil
}

// GetCharacterSheet returns a character sheet (character, profile, and state).
func (s *CharacterService) GetCharacterSheet(ctx context.Context, in *campaignv1.GetCharacterSheetRequest) (*campaignv1.GetCharacterSheetResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get character sheet request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpRead); err != nil {
		return nil, handleDomainError(err)
	}

	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}

	ch, err := s.stores.Character.GetCharacter(ctx, campaignID, characterID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	dhProfile, err := s.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, status.Errorf(codes.Internal, "get daggerheart profile: %v", err)
	}

	dhState, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, status.Errorf(codes.Internal, "get daggerheart state: %v", err)
	}

	return &campaignv1.GetCharacterSheetResponse{
		Character: characterToProto(ch),
		Profile:   daggerheartProfileToProto(campaignID, characterID, dhProfile),
		State:     daggerheartStateToProto(campaignID, characterID, dhState),
	}, nil
}

// PatchCharacterProfile patches a character profile (all fields optional).
func (s *CharacterService) PatchCharacterProfile(ctx context.Context, in *campaignv1.PatchCharacterProfileRequest) (*campaignv1.PatchCharacterProfileResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "patch character profile request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	characterID, dhProfile, err := newCharacterApplication(s).PatchCharacterProfile(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.PatchCharacterProfileResponse{
		Profile: daggerheartProfileToProto(campaignID, characterID, dhProfile),
	}, nil
}

// daggerheartProfileToProto converts a Daggerheart profile to proto.
func daggerheartProfileToProto(campaignID, characterID string, dh storage.DaggerheartCharacterProfile) *campaignv1.CharacterProfile {
	return &campaignv1.CharacterProfile{
		CampaignId:  campaignID,
		CharacterId: characterID,
		SystemProfile: &campaignv1.CharacterProfile_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartProfile{
				Level:           int32(dh.Level),
				HpMax:           int32(dh.HpMax),
				StressMax:       wrapperspb.Int32(int32(dh.StressMax)),
				Evasion:         wrapperspb.Int32(int32(dh.Evasion)),
				MajorThreshold:  wrapperspb.Int32(int32(dh.MajorThreshold)),
				SevereThreshold: wrapperspb.Int32(int32(dh.SevereThreshold)),
				Proficiency:     wrapperspb.Int32(int32(dh.Proficiency)),
				ArmorScore:      wrapperspb.Int32(int32(dh.ArmorScore)),
				ArmorMax:        wrapperspb.Int32(int32(dh.ArmorMax)),
				Agility:         wrapperspb.Int32(int32(dh.Agility)),
				Strength:        wrapperspb.Int32(int32(dh.Strength)),
				Finesse:         wrapperspb.Int32(int32(dh.Finesse)),
				Instinct:        wrapperspb.Int32(int32(dh.Instinct)),
				Presence:        wrapperspb.Int32(int32(dh.Presence)),
				Knowledge:       wrapperspb.Int32(int32(dh.Knowledge)),
				Experiences:     daggerheartExperiencesToProto(dh.Experiences),
			},
		},
	}
}

// daggerheartStateToProto converts a Daggerheart state to proto.
func daggerheartStateToProto(campaignID, characterID string, dh storage.DaggerheartCharacterState) *campaignv1.CharacterState {
	return &campaignv1.CharacterState{
		CampaignId:  campaignID,
		CharacterId: characterID,
		SystemState: &campaignv1.CharacterState_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCharacterState{
				Hp:         int32(dh.Hp),
				Hope:       int32(dh.Hope),
				HopeMax:    int32(dh.HopeMax),
				Stress:     int32(dh.Stress),
				Armor:      int32(dh.Armor),
				Conditions: daggerheartConditionsToProto(dh.Conditions),
				LifeState:  daggerheartLifeStateToProto(dh.LifeState),
			},
		},
	}
}

func daggerheartExperiencesToProto(experiences []storage.DaggerheartExperience) []*daggerheartv1.DaggerheartExperience {
	if len(experiences) == 0 {
		return nil
	}
	result := make([]*daggerheartv1.DaggerheartExperience, 0, len(experiences))
	for _, experience := range experiences {
		result = append(result, &daggerheartv1.DaggerheartExperience{
			Name:     experience.Name,
			Modifier: int32(experience.Modifier),
		})
	}
	return result
}
