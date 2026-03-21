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
	daggerheartcontent "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) TransformBeastform(ctx context.Context, in *pb.DaggerheartTransformBeastformRequest) (*pb.DaggerheartTransformBeastformResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "transform beastform request is required")
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
	beastformID := strings.TrimSpace(in.GetBeastformId())
	if beastformID == "" {
		return nil, status.Error(codes.InvalidArgument, "beastform id is required")
	}
	profile, err := h.validateCharacterPreconditions(ctx, campaignID, characterID, "beastform")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(profile.ClassID) != "class.druid" {
		return nil, status.Error(codes.FailedPrecondition, "beastform requires Druid")
	}
	beastform, err := h.deps.Content.GetDaggerheartBeastform(ctx, beastformID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	if beastform.Tier > tierForLevel(profile.Level) {
		return nil, status.Error(codes.FailedPrecondition, "beastform tier exceeds character tier")
	}
	state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	classState := classStateFromProjection(state.ClassState)
	payload := daggerheart.BeastformTransformPayload{
		ActorCharacterID: ids.CharacterID(characterID),
		CharacterID:      ids.CharacterID(characterID),
		BeastformID:      beastform.ID,
		ClassStateBefore: classStatePtr(classState),
	}
	evolutionTrait := ""
	if in.GetUseEvolution() {
		if state.Hope < 3 {
			return nil, status.Error(codes.FailedPrecondition, "insufficient hope")
		}
		evolutionTrait = strings.TrimSpace(in.GetEvolutionTrait())
		if evolutionTrait == "" {
			return nil, status.Error(codes.InvalidArgument, "evolution_trait is required")
		}
		payload.UseEvolution = true
		payload.EvolutionTrait = evolutionTrait
		payload.HopeBefore = intPtr(state.Hope)
		payload.HopeAfter = intPtr(state.Hope - 3)
	} else {
		if state.Stress >= profile.StressMax {
			return nil, status.Error(codes.FailedPrecondition, "insufficient stress capacity")
		}
		payload.StressBefore = intPtr(state.Stress)
		payload.StressAfter = intPtr(state.Stress + 1)
	}
	nextClassState := daggerheart.WithActiveBeastform(classState, resolvedBeastformState(beastform, evolutionTrait))
	payload.ClassStateAfter = classStatePtr(nextClassState)
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, grpcerror.Internal("encode beastform payload", err)
	}
	if err := h.executeCharacterCommand(ctx, CharacterCommandInput{
		CampaignID:      campaignID,
		CharacterID:     characterID,
		CommandType:     commandids.DaggerheartBeastformTransform,
		SessionID:       strings.TrimSpace(in.GetSessionId()),
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "beastform transform did not emit an event",
		ApplyErrMessage: "apply beastform transform event",
	}); err != nil {
		return nil, err
	}
	updatedState, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, grpcerror.Internal("load daggerheart state", err)
	}
	return &pb.DaggerheartTransformBeastformResponse{
		CharacterId: characterID,
		State:       statetransport.CharacterStateToProto(updatedState),
	}, nil
}

func (h *Handler) DropBeastform(ctx context.Context, in *pb.DaggerheartDropBeastformRequest) (*pb.DaggerheartDropBeastformResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "drop beastform request is required")
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
	if _, err := h.validateCharacterPreconditions(ctx, campaignID, characterID, "beastform"); err != nil {
		return nil, err
	}
	state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, grpcerror.HandleDomainError(err)
	}
	classState := classStateFromProjection(state.ClassState)
	if classState.ActiveBeastform == nil {
		return nil, status.Error(codes.FailedPrecondition, "character is not in beastform")
	}
	nextClassState := daggerheart.WithActiveBeastform(classState, nil)
	payload := daggerheart.BeastformDropPayload{
		ActorCharacterID: ids.CharacterID(characterID),
		CharacterID:      ids.CharacterID(characterID),
		BeastformID:      classState.ActiveBeastform.BeastformID,
		Source:           "beastform.drop",
		ClassStateBefore: classStatePtr(classState),
		ClassStateAfter:  classStatePtr(nextClassState),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, grpcerror.Internal("encode beastform drop payload", err)
	}
	if err := h.executeCharacterCommand(ctx, CharacterCommandInput{
		CampaignID:      campaignID,
		CharacterID:     characterID,
		CommandType:     commandids.DaggerheartBeastformDrop,
		SessionID:       strings.TrimSpace(in.GetSessionId()),
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "beastform drop did not emit an event",
		ApplyErrMessage: "apply beastform drop event",
	}); err != nil {
		return nil, err
	}
	updatedState, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, grpcerror.Internal("load daggerheart state", err)
	}
	return &pb.DaggerheartDropBeastformResponse{
		CharacterId: characterID,
		State:       statetransport.CharacterStateToProto(updatedState),
	}, nil
}

func resolvedBeastformState(beastform daggerheartcontent.DaggerheartBeastformEntry, evolutionTrait string) *daggerheart.CharacterActiveBeastformState {
	baseTrait := strings.TrimSpace(beastform.Trait)
	attackTrait := strings.TrimSpace(beastform.Attack.Trait)
	if attackTrait == "" {
		attackTrait = baseTrait
	}
	evolutionTrait = strings.TrimSpace(evolutionTrait)
	if evolutionTrait != "" {
		attackTrait = evolutionTrait
	}
	damageDice := make([]daggerheart.CharacterDamageDie, 0, len(beastform.Attack.DamageDice))
	for _, die := range beastform.Attack.DamageDice {
		damageDice = append(damageDice, daggerheart.CharacterDamageDie{
			Count: die.Count,
			Sides: die.Sides,
		})
	}
	return &daggerheart.CharacterActiveBeastformState{
		BeastformID:            strings.TrimSpace(beastform.ID),
		BaseTrait:              baseTrait,
		AttackTrait:            attackTrait,
		TraitBonus:             beastform.TraitBonus,
		EvasionBonus:           beastform.EvasionBonus,
		AttackRange:            strings.TrimSpace(beastform.Attack.Range),
		DamageDice:             damageDice,
		DamageBonus:            beastform.Attack.DamageBonus,
		DamageType:             strings.TrimSpace(beastform.Attack.DamageType),
		EvolutionTraitOverride: evolutionTrait,
		DropOnAnyHPMark:        beastformHasFragile(beastform.Features),
	}
}

func beastformHasFragile(features []daggerheartcontent.DaggerheartBeastformFeature) bool {
	for _, feature := range features {
		if strings.EqualFold(strings.TrimSpace(feature.ID), "fragile") ||
			strings.EqualFold(strings.TrimSpace(feature.Name), "fragile") {
			return true
		}
	}
	return false
}
