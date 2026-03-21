package charactermutationtransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/statetransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) BeginCompanionExperience(ctx context.Context, in *pb.DaggerheartBeginCompanionExperienceRequest) (*pb.DaggerheartBeginCompanionExperienceResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "begin companion experience request is required")
	}
	if err := h.requireDependencies(); err != nil {
		return nil, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}
	experienceID, err := validate.RequiredID(in.GetExperienceId(), "experience id")
	if err != nil {
		return nil, err
	}
	profile, err := h.validateCharacterPreconditions(ctx, campaignID, characterID, "companion")
	if err != nil {
		return nil, err
	}
	if profile.CompanionSheet == nil {
		return nil, status.Error(codes.FailedPrecondition, "companion requires a companion sheet")
	}
	if !profileHasCompanionExperience(profile, experienceID) {
		return nil, status.Error(codes.FailedPrecondition, "experience_id is not on the companion sheet")
	}
	state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	companionState := companionStateForCharacter(profile, state)
	if companionState.Status == daggerheart.CompanionStatusAway {
		return nil, status.Error(codes.FailedPrecondition, "companion is already away")
	}
	nextCompanionState := daggerheart.WithActiveCompanionExperience(companionState, experienceID)
	payload := daggerheart.CompanionExperienceBeginPayload{
		ActorCharacterID:     ids.CharacterID(characterID),
		CharacterID:          ids.CharacterID(characterID),
		ExperienceID:         experienceID,
		CompanionStateBefore: companionStatePtr(companionState),
		CompanionStateAfter:  companionStatePtr(nextCompanionState),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, grpcerror.Internal("encode companion begin payload", err)
	}
	if err := h.executeCharacterCommand(ctx, CharacterCommandInput{
		CampaignID:      campaignID,
		CharacterID:     characterID,
		CommandType:     commandids.DaggerheartCompanionExperienceBegin,
		SessionID:       strings.TrimSpace(in.GetSessionId()),
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "companion begin did not emit an event",
		ApplyErrMessage: "apply companion begin event",
	}); err != nil {
		return nil, err
	}
	updatedState, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, grpcerror.Internal("load daggerheart state", err)
	}
	return &pb.DaggerheartBeginCompanionExperienceResponse{
		CharacterId: characterID,
		State:       statetransport.CharacterStateToProto(updatedState),
	}, nil
}

func (h *Handler) ReturnCompanion(ctx context.Context, in *pb.DaggerheartReturnCompanionRequest) (*pb.DaggerheartReturnCompanionResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "return companion request is required")
	}
	if err := h.requireDependencies(); err != nil {
		return nil, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}
	resolution := companionReturnResolutionLabel(in.GetResolution())
	if resolution == "" {
		return nil, status.Error(codes.InvalidArgument, "resolution is required")
	}
	profile, err := h.validateCharacterPreconditions(ctx, campaignID, characterID, "companion")
	if err != nil {
		return nil, err
	}
	if profile.CompanionSheet == nil {
		return nil, status.Error(codes.FailedPrecondition, "companion requires a companion sheet")
	}
	state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	companionState := companionStateForCharacter(profile, state)
	if companionState.Status != daggerheart.CompanionStatusAway || strings.TrimSpace(companionState.ActiveExperienceID) == "" {
		return nil, status.Error(codes.FailedPrecondition, "companion is not away")
	}
	nextCompanionState := daggerheart.WithCompanionPresent(companionState)
	payload := daggerheart.CompanionReturnPayload{
		ActorCharacterID:     ids.CharacterID(characterID),
		CharacterID:          ids.CharacterID(characterID),
		Resolution:           resolution,
		CompanionStateBefore: companionStatePtr(companionState),
		CompanionStateAfter:  companionStatePtr(nextCompanionState),
	}
	if resolution == "experience_completed" && state.Stress > 0 {
		payload.StressBefore = intPtr(state.Stress)
		payload.StressAfter = intPtr(state.Stress - 1)
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, grpcerror.Internal("encode companion return payload", err)
	}
	if err := h.executeCharacterCommand(ctx, CharacterCommandInput{
		CampaignID:      campaignID,
		CharacterID:     characterID,
		CommandType:     commandids.DaggerheartCompanionReturn,
		SessionID:       strings.TrimSpace(in.GetSessionId()),
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "companion return did not emit an event",
		ApplyErrMessage: "apply companion return event",
	}); err != nil {
		return nil, err
	}
	updatedState, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, grpcerror.Internal("load daggerheart state", err)
	}
	return &pb.DaggerheartReturnCompanionResponse{
		CharacterId: characterID,
		State:       statetransport.CharacterStateToProto(updatedState),
	}, nil
}

func profileHasCompanionExperience(profile projectionstore.DaggerheartCharacterProfile, experienceID string) bool {
	if profile.CompanionSheet == nil {
		return false
	}
	for _, experience := range profile.CompanionSheet.Experiences {
		if strings.TrimSpace(experience.ExperienceID) == experienceID {
			return true
		}
	}
	return false
}

func companionStateForCharacter(profile projectionstore.DaggerheartCharacterProfile, state projectionstore.DaggerheartCharacterState) daggerheart.CharacterCompanionState {
	if profile.CompanionSheet == nil {
		return daggerheart.CharacterCompanionState{}
	}
	if state.CompanionState == nil {
		return daggerheart.CharacterCompanionState{Status: daggerheart.CompanionStatusPresent}
	}
	return daggerheart.CharacterCompanionState{
		Status:             state.CompanionState.Status,
		ActiveExperienceID: state.CompanionState.ActiveExperienceID,
	}.Normalized()
}

func companionReturnResolutionLabel(resolution pb.DaggerheartCompanionReturnResolution) string {
	switch resolution {
	case pb.DaggerheartCompanionReturnResolution_DAGGERHEART_COMPANION_RETURN_RESOLUTION_EXPERIENCE_COMPLETED:
		return "experience_completed"
	case pb.DaggerheartCompanionReturnResolution_DAGGERHEART_COMPANION_RETURN_RESOLUTION_EARLY_RETURN:
		return "early_return"
	default:
		return ""
	}
}

func companionStatePtr(value daggerheart.CharacterCompanionState) *daggerheart.CharacterCompanionState {
	normalized := value.Normalized()
	return &normalized
}
