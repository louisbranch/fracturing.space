package campaign

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/campaign/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/campaign"
	"github.com/louisbranch/fracturing.space/internal/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/campaign/projection"
	"github.com/louisbranch/fracturing.space/internal/id"
	"github.com/louisbranch/fracturing.space/internal/storage"
	"github.com/louisbranch/fracturing.space/internal/systems/daggerheart"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	defaultListCharactersPageSize = 10
	maxListCharactersPageSize     = 10
)

// CharacterService implements the campaign.v1.CharacterService gRPC API.
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

	if s.stores.Character == nil {
		return nil, status.Error(codes.Internal, "character store is not configured")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
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

	input := character.CreateCharacterInput{
		CampaignID: campaignID,
		Name:       in.GetName(),
		Kind:       characterKindFromProto(in.GetKind()),
		Notes:      in.GetNotes(),
	}
	normalized, err := character.NormalizeCreateCharacterInput(input)
	if err != nil {
		return nil, handleDomainError(err)
	}

	characterID, err := s.idGenerator()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate character id: %v", err)
	}

	payload := event.CharacterCreatedPayload{
		CharacterID: characterID,
		Name:        normalized.Name,
		Kind:        in.GetKind().String(),
		Notes:       normalized.Notes,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    s.clock().UTC(),
		Type:         event.TypeCharacterCreated,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "character",
		EntityID:     characterID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := projection.Applier{Campaign: s.stores.Campaign, Participant: s.stores.Participant, Character: s.stores.Character}
	if err := applier.Apply(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	created, err := s.stores.Character.GetCharacter(ctx, campaignID, characterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load character: %v", err)
	}

	// Get Daggerheart defaults for the character kind
	kindStr := "PC"
	if created.Kind == character.CharacterKindNPC {
		kindStr = "NPC"
	}
	dhDefaults := daggerheart.GetProfileDefaults(kindStr)

	reqID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	profileActorType := event.ActorTypeSystem
	if actorID != "" {
		profileActorType = event.ActorTypeGM
	}

	profilePayload := event.ProfileUpdatedPayload{
		CharacterID: created.ID,
		SystemProfile: map[string]any{
			"daggerheart": map[string]any{
				"hp_max":           dhDefaults.HpMax,
				"stress_max":       dhDefaults.StressMax,
				"evasion":          dhDefaults.Evasion,
				"major_threshold":  dhDefaults.MajorThreshold,
				"severe_threshold": dhDefaults.SevereThreshold,
				"agility":          dhDefaults.Traits.Agility,
				"strength":         dhDefaults.Traits.Strength,
				"finesse":          dhDefaults.Traits.Finesse,
				"instinct":         dhDefaults.Traits.Instinct,
				"presence":         dhDefaults.Traits.Presence,
				"knowledge":        dhDefaults.Traits.Knowledge,
			},
		},
	}
	profileJSON, err := json.Marshal(profilePayload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode profile payload: %v", err)
	}
	profileEvent, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    s.clock().UTC(),
		Type:         event.TypeProfileUpdated,
		RequestID:    reqID,
		InvocationID: invocationID,
		ActorType:    profileActorType,
		ActorID:      actorID,
		EntityType:   "character",
		EntityID:     created.ID,
		PayloadJSON:  profileJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append profile event: %v", err)
	}

	hpAfter := dhDefaults.HpMax
	statePayload := event.CharacterStateChangedPayload{
		CharacterID: created.ID,
		HpAfter:     &hpAfter,
		SystemState: map[string]any{
			"daggerheart": map[string]any{
				"hope_after":   daggerheart.HopeDefault,
				"stress_after": daggerheart.StressDefault,
			},
		},
	}
	stateJSON, err := json.Marshal(statePayload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode state payload: %v", err)
	}
	stateEvent, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    s.clock().UTC(),
		Type:         event.TypeCharacterStateChanged,
		SessionID:    grpcmeta.SessionIDFromContext(ctx),
		RequestID:    reqID,
		InvocationID: invocationID,
		ActorType:    profileActorType,
		ActorID:      actorID,
		EntityType:   "character",
		EntityID:     created.ID,
		PayloadJSON:  stateJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append state event: %v", err)
	}

	projectionApplier := projection.Applier{
		Campaign:    s.stores.Campaign,
		Character:   s.stores.Character,
		Daggerheart: s.stores.Daggerheart,
	}
	if err := projectionApplier.Apply(ctx, profileEvent); err != nil {
		return nil, status.Errorf(codes.Internal, "apply profile event: %v", err)
	}
	if err := projectionApplier.Apply(ctx, stateEvent); err != nil {
		return nil, status.Errorf(codes.Internal, "apply state event: %v", err)
	}

	return &campaignv1.CreateCharacterResponse{
		Character: characterToProto(created),
	}, nil
}

// UpdateCharacter updates a character's metadata.
func (s *CharacterService) UpdateCharacter(ctx context.Context, in *campaignv1.UpdateCharacterRequest) (*campaignv1.UpdateCharacterResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update character request is required")
	}

	if s.stores.Character == nil {
		return nil, status.Error(codes.Internal, "character store is not configured")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
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

	fields := make(map[string]any)
	if name := in.GetName(); name != nil {
		trimmed := strings.TrimSpace(name.GetValue())
		if trimmed == "" {
			return nil, status.Error(codes.InvalidArgument, "name must not be empty")
		}
		ch.Name = trimmed
		fields["name"] = trimmed
	}
	if in.GetKind() != campaignv1.CharacterKind_CHARACTER_KIND_UNSPECIFIED {
		kind := characterKindFromProto(in.GetKind())
		if kind == character.CharacterKindUnspecified {
			return nil, status.Error(codes.InvalidArgument, "character kind is invalid")
		}
		ch.Kind = kind
		fields["kind"] = in.GetKind().String()
	}
	if notes := in.GetNotes(); notes != nil {
		ch.Notes = strings.TrimSpace(notes.GetValue())
		fields["notes"] = ch.Notes
	}
	if len(fields) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one field must be provided")
	}

	payload := event.CharacterUpdatedPayload{
		CharacterID: characterID,
		Fields:      fields,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    s.clock().UTC(),
		Type:         event.TypeCharacterUpdated,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "character",
		EntityID:     characterID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := projection.Applier{
		Campaign:    s.stores.Campaign,
		Character:   s.stores.Character,
		Control:     s.stores.ControlDefault,
		Daggerheart: s.stores.Daggerheart,
		Participant: s.stores.Participant,
	}
	if err := applier.Apply(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	updated, err := s.stores.Character.GetCharacter(ctx, campaignID, characterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load character: %v", err)
	}

	return &campaignv1.UpdateCharacterResponse{Character: characterToProto(updated)}, nil
}

// DeleteCharacter deletes a character.
func (s *CharacterService) DeleteCharacter(ctx context.Context, in *campaignv1.DeleteCharacterRequest) (*campaignv1.DeleteCharacterResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "delete character request is required")
	}

	if s.stores.Character == nil {
		return nil, status.Error(codes.Internal, "character store is not configured")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
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

	payload := event.CharacterDeletedPayload{
		CharacterID: characterID,
		Reason:      strings.TrimSpace(in.GetReason()),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    s.clock().UTC(),
		Type:         event.TypeCharacterDeleted,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "character",
		EntityID:     characterID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := projection.Applier{
		Campaign:    s.stores.Campaign,
		Character:   s.stores.Character,
		Control:     s.stores.ControlDefault,
		Daggerheart: s.stores.Daggerheart,
		Participant: s.stores.Participant,
	}
	if err := applier.Apply(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	return &campaignv1.DeleteCharacterResponse{Character: characterToProto(ch)}, nil
}

// ListCharacters returns a page of character records for a campaign.
func (s *CharacterService) ListCharacters(ctx context.Context, in *campaignv1.ListCharactersRequest) (*campaignv1.ListCharactersResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list characters request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Character == nil {
		return nil, status.Error(codes.Internal, "character store is not configured")
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

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Character == nil {
		return nil, status.Error(codes.Internal, "character store is not configured")
	}
	if s.stores.ControlDefault == nil {
		return nil, status.Error(codes.Internal, "control default store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	_, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}
	_, err = s.stores.Character.GetCharacter(ctx, campaignID, characterID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	if in.GetController() == nil {
		return nil, status.Error(codes.InvalidArgument, "controller is required")
	}
	controller, err := characterControllerFromProto(in.GetController())
	if err != nil {
		return nil, handleDomainError(err)
	}

	// If participant controller, validate participant exists
	if !controller.IsGM {
		if s.stores.Participant == nil {
			return nil, status.Error(codes.Internal, "participant store is not configured")
		}
		_, err = s.stores.Participant.GetParticipant(ctx, campaignID, controller.ParticipantID)
		if err != nil {
			return nil, handleDomainError(err)
		}
	}

	payload := event.ControllerAssignedPayload{
		CharacterID:   characterID,
		IsGM:          controller.IsGM,
		ParticipantID: controller.ParticipantID,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    s.clock().UTC(),
		Type:         event.TypeControllerAssigned,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "character",
		EntityID:     characterID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := projection.Applier{
		Campaign:    s.stores.Campaign,
		Character:   s.stores.Character,
		Control:     s.stores.ControlDefault,
		Daggerheart: s.stores.Daggerheart,
		Participant: s.stores.Participant,
	}
	if err := applier.Apply(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	return &campaignv1.SetDefaultControlResponse{
		CampaignId:  campaignID,
		CharacterId: characterID,
		Controller:  characterControllerToProto(controller),
	}, nil
}

// GetCharacterSheet returns a character sheet (character, profile, and state).
func (s *CharacterService) GetCharacterSheet(ctx context.Context, in *campaignv1.GetCharacterSheetRequest) (*campaignv1.GetCharacterSheetResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get character sheet request is required")
	}

	if s.stores.Character == nil {
		return nil, status.Error(codes.Internal, "character store is not configured")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
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

	if s.stores.Daggerheart == nil {
		return nil, status.Error(codes.Internal, "daggerheart store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}

	dhProfile, err := s.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	// Apply Daggerheart-specific patches (including hp_max)
	if dhPatch := in.GetDaggerheart(); dhPatch != nil {
		// Validate hp_max (plain int32: 0 is not valid)
		if dhPatch.HpMax < 0 {
			return nil, status.Error(codes.InvalidArgument, "hp_max must be non-negative")
		}
		if dhPatch.HpMax > 0 {
			dhProfile.HpMax = int(dhPatch.HpMax)
		}

		// Validate stress_max (wrapper type: nil means not provided)
		if dhPatch.GetStressMax() != nil {
			val := int(dhPatch.GetStressMax().GetValue())
			if val < 0 {
				return nil, status.Error(codes.InvalidArgument, "stress_max must be non-negative")
			}
			dhProfile.StressMax = val
		}

		// Validate evasion (wrapper type: nil means not provided)
		if dhPatch.GetEvasion() != nil {
			val := int(dhPatch.GetEvasion().GetValue())
			if val < 0 {
				return nil, status.Error(codes.InvalidArgument, "evasion must be non-negative")
			}
			dhProfile.Evasion = val
		}

		// Validate major_threshold (wrapper type: nil means not provided)
		if dhPatch.GetMajorThreshold() != nil {
			val := int(dhPatch.GetMajorThreshold().GetValue())
			if val < 0 {
				return nil, status.Error(codes.InvalidArgument, "major_threshold must be non-negative")
			}
			dhProfile.MajorThreshold = val
		}

		// Validate severe_threshold (wrapper type: nil means not provided)
		if dhPatch.GetSevereThreshold() != nil {
			val := int(dhPatch.GetSevereThreshold().GetValue())
			if val < 0 {
				return nil, status.Error(codes.InvalidArgument, "severe_threshold must be non-negative")
			}
			dhProfile.SevereThreshold = val
		}

		// Validate and apply traits (wrapper types allow nil-checking)
		if dhPatch.GetAgility() != nil {
			if err := daggerheart.ValidateTrait("agility", int(dhPatch.GetAgility().GetValue())); err != nil {
				return nil, handleDomainError(err)
			}
			dhProfile.Agility = int(dhPatch.GetAgility().GetValue())
		}
		if dhPatch.GetStrength() != nil {
			if err := daggerheart.ValidateTrait("strength", int(dhPatch.GetStrength().GetValue())); err != nil {
				return nil, handleDomainError(err)
			}
			dhProfile.Strength = int(dhPatch.GetStrength().GetValue())
		}
		if dhPatch.GetFinesse() != nil {
			if err := daggerheart.ValidateTrait("finesse", int(dhPatch.GetFinesse().GetValue())); err != nil {
				return nil, handleDomainError(err)
			}
			dhProfile.Finesse = int(dhPatch.GetFinesse().GetValue())
		}
		if dhPatch.GetInstinct() != nil {
			if err := daggerheart.ValidateTrait("instinct", int(dhPatch.GetInstinct().GetValue())); err != nil {
				return nil, handleDomainError(err)
			}
			dhProfile.Instinct = int(dhPatch.GetInstinct().GetValue())
		}
		if dhPatch.GetPresence() != nil {
			if err := daggerheart.ValidateTrait("presence", int(dhPatch.GetPresence().GetValue())); err != nil {
				return nil, handleDomainError(err)
			}
			dhProfile.Presence = int(dhPatch.GetPresence().GetValue())
		}
		if dhPatch.GetKnowledge() != nil {
			if err := daggerheart.ValidateTrait("knowledge", int(dhPatch.GetKnowledge().GetValue())); err != nil {
				return nil, handleDomainError(err)
			}
			dhProfile.Knowledge = int(dhPatch.GetKnowledge().GetValue())
		}

	}

	profilePayload := event.ProfileUpdatedPayload{
		CharacterID: characterID,
		SystemProfile: map[string]any{
			"daggerheart": map[string]any{
				"hp_max":           dhProfile.HpMax,
				"stress_max":       dhProfile.StressMax,
				"evasion":          dhProfile.Evasion,
				"major_threshold":  dhProfile.MajorThreshold,
				"severe_threshold": dhProfile.SevereThreshold,
				"agility":          dhProfile.Agility,
				"strength":         dhProfile.Strength,
				"finesse":          dhProfile.Finesse,
				"instinct":         dhProfile.Instinct,
				"presence":         dhProfile.Presence,
				"knowledge":        dhProfile.Knowledge,
			},
		},
	}
	profilePayloadJSON, err := json.Marshal(profilePayload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    s.clock().UTC(),
		Type:         event.TypeProfileUpdated,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "character",
		EntityID:     characterID,
		PayloadJSON:  profilePayloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := projection.Applier{
		Campaign:    s.stores.Campaign,
		Character:   s.stores.Character,
		Control:     s.stores.ControlDefault,
		Daggerheart: s.stores.Daggerheart,
		Participant: s.stores.Participant,
	}
	if err := applier.Apply(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
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
				HpMax:           int32(dh.HpMax),
				StressMax:       wrapperspb.Int32(int32(dh.StressMax)),
				Evasion:         wrapperspb.Int32(int32(dh.Evasion)),
				MajorThreshold:  wrapperspb.Int32(int32(dh.MajorThreshold)),
				SevereThreshold: wrapperspb.Int32(int32(dh.SevereThreshold)),
				Agility:         wrapperspb.Int32(int32(dh.Agility)),
				Strength:        wrapperspb.Int32(int32(dh.Strength)),
				Finesse:         wrapperspb.Int32(int32(dh.Finesse)),
				Instinct:        wrapperspb.Int32(int32(dh.Instinct)),
				Presence:        wrapperspb.Int32(int32(dh.Presence)),
				Knowledge:       wrapperspb.Int32(int32(dh.Knowledge)),
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
				Hp:     int32(dh.Hp),
				Hope:   int32(dh.Hope),
				Stress: int32(dh.Stress),
			},
		},
	}
}
