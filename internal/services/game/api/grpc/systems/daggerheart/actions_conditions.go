package daggerheart

import (
	"context"
	"encoding/json"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *DaggerheartService) ApplyConditions(ctx context.Context, in *pb.DaggerheartApplyConditionsRequest) (*pb.DaggerheartApplyConditionsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply conditions request is required")
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
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart conditions")
	}

	sessionID := strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if err := s.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
		return nil, err
	}

	state, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	lifeStateProvided := in.GetLifeState() != pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED
	var lifeStateAfter string
	if lifeStateProvided {
		lifeStateAfter, err = daggerheartLifeStateFromProto(in.GetLifeState())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	lifeStateBefore := state.LifeState
	if lifeStateBefore == "" {
		lifeStateBefore = daggerheart.LifeStateAlive
	}
	lifeStateChanged := false
	if lifeStateProvided {
		beforeValue, err := daggerheart.NormalizeLifeState(lifeStateBefore)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "invalid stored life_state: %v", err)
		}
		afterValue, err := daggerheart.NormalizeLifeState(lifeStateAfter)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		lifeStateChanged = beforeValue != afterValue
	}

	addConditions, err := daggerheartConditionsFromProto(in.GetAdd())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	removeConditions, err := daggerheartConditionsFromProto(in.GetRemove())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if len(addConditions) == 0 && len(removeConditions) == 0 && !lifeStateProvided {
		return nil, status.Error(codes.InvalidArgument, "conditions or life_state are required")
	}

	normalizedAdd := []string{}
	if len(addConditions) > 0 {
		normalizedAdd, err = daggerheart.NormalizeConditions(addConditions)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	normalizedRemove := []string{}
	if len(removeConditions) > 0 {
		normalizedRemove, err = daggerheart.NormalizeConditions(removeConditions)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	removeSet := make(map[string]struct{}, len(normalizedRemove))
	for _, value := range normalizedRemove {
		removeSet[value] = struct{}{}
	}
	for _, value := range normalizedAdd {
		if _, ok := removeSet[value]; ok {
			return nil, status.Error(codes.InvalidArgument, "conditions cannot be both added and removed")
		}
	}

	var before []string
	var after []string
	var added []string
	var removed []string
	conditionChanged := false
	if len(normalizedAdd) > 0 || len(normalizedRemove) > 0 {
		before, err = daggerheart.NormalizeConditions(state.Conditions)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "invalid stored conditions: %v", err)
		}

		afterSet := make(map[string]struct{}, len(before)+len(normalizedAdd))
		for _, value := range before {
			afterSet[value] = struct{}{}
		}
		for _, value := range normalizedRemove {
			delete(afterSet, value)
		}
		for _, value := range normalizedAdd {
			afterSet[value] = struct{}{}
		}

		afterList := make([]string, 0, len(afterSet))
		for value := range afterSet {
			afterList = append(afterList, value)
		}
		after, err = daggerheart.NormalizeConditions(afterList)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "invalid condition set: %v", err)
		}

		added, removed = daggerheart.DiffConditions(before, after)
		conditionChanged = len(added) > 0 || len(removed) > 0
		if !conditionChanged && !lifeStateChanged {
			return nil, status.Error(codes.FailedPrecondition, "no condition or life_state changes to apply")
		}
	} else if !lifeStateChanged {
		return nil, status.Error(codes.FailedPrecondition, "no condition or life_state changes to apply")
	}

	source := strings.TrimSpace(in.GetSource())
	var rollSeq *uint64
	if in.RollSeq != nil {
		value := in.GetRollSeq()
		rollSeq = &value
		rollEvent, err := s.stores.Event.GetEventBySeq(ctx, campaignID, value)
		if err != nil {
			return nil, handleDomainError(err)
		}
		if sessionID != "" && rollEvent.SessionID != sessionID {
			return nil, status.Error(codes.InvalidArgument, "roll seq does not match session")
		}
	}

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	if conditionChanged {
		payload := daggerheart.ConditionChangePayload{
			CharacterID:      characterID,
			ConditionsBefore: before,
			ConditionsAfter:  after,
			Added:            added,
			Removed:          removed,
			Source:           source,
			RollSeq:          rollSeq,
		}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "encode condition payload: %v", err)
		}
		_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
			CampaignID:    campaignID,
			Type:          commandTypeDaggerheartConditionChange,
			ActorType:     command.ActorTypeSystem,
			SessionID:     sessionID,
			RequestID:     requestID,
			InvocationID:  invocationID,
			EntityType:    "character",
			EntityID:      characterID,
			SystemID:      daggerheart.SystemID,
			SystemVersion: daggerheart.SystemVersion,
			PayloadJSON:   payloadJSON,
		}, adapter, domainCommandApplyOptions{
			requireEvents:   true,
			missingEventMsg: "condition change did not emit an event",
			applyErrMessage: "apply condition event",
			executeErrMsg:   "execute domain command",
		})
		if err != nil {
			return nil, err
		}
	}

	if lifeStateChanged {
		payload := daggerheart.CharacterStatePatchedPayload{
			CharacterID:     characterID,
			LifeStateBefore: &lifeStateBefore,
			LifeStateAfter:  &lifeStateAfter,
		}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "encode character state payload: %v", err)
		}
		_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
			CampaignID:    campaignID,
			Type:          commandTypeDaggerheartCharacterStatePatch,
			ActorType:     command.ActorTypeSystem,
			SessionID:     sessionID,
			RequestID:     requestID,
			InvocationID:  invocationID,
			EntityType:    "character",
			EntityID:      characterID,
			SystemID:      daggerheart.SystemID,
			SystemVersion: daggerheart.SystemVersion,
			PayloadJSON:   payloadJSON,
		}, adapter, domainCommandApplyOptions{
			requireEvents:   true,
			missingEventMsg: "character state patch did not emit an event",
			applyErrMessage: "apply character state event",
			executeErrMsg:   "execute domain command",
		})
		if err != nil {
			return nil, err
		}
	}

	updated, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load daggerheart state: %v", err)
	}

	return &pb.DaggerheartApplyConditionsResponse{
		CharacterId: characterID,
		State:       daggerheartStateToProto(updated),
		Added:       daggerheartConditionsToProto(added),
		Removed:     daggerheartConditionsToProto(removed),
	}, nil
}

func (s *DaggerheartService) ApplyAdversaryConditions(ctx context.Context, in *pb.DaggerheartApplyAdversaryConditionsRequest) (*pb.DaggerheartApplyAdversaryConditionsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply adversary conditions request is required")
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
	adversaryID := strings.TrimSpace(in.GetAdversaryId())
	if adversaryID == "" {
		return nil, status.Error(codes.InvalidArgument, "adversary id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart conditions")
	}

	sessionID := strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if err := s.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
		return nil, err
	}

	addConditions, err := daggerheartConditionsFromProto(in.GetAdd())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	removeConditions, err := daggerheartConditionsFromProto(in.GetRemove())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if len(addConditions) == 0 && len(removeConditions) == 0 {
		return nil, status.Error(codes.InvalidArgument, "conditions to add or remove are required")
	}

	normalizedAdd := []string{}
	if len(addConditions) > 0 {
		normalizedAdd, err = daggerheart.NormalizeConditions(addConditions)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	normalizedRemove := []string{}
	if len(removeConditions) > 0 {
		normalizedRemove, err = daggerheart.NormalizeConditions(removeConditions)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	removeSet := make(map[string]struct{}, len(normalizedRemove))
	for _, value := range normalizedRemove {
		removeSet[value] = struct{}{}
	}
	for _, value := range normalizedAdd {
		if _, ok := removeSet[value]; ok {
			return nil, status.Error(codes.InvalidArgument, "conditions cannot be both added and removed")
		}
	}

	adversary, err := s.loadAdversaryForSession(ctx, campaignID, sessionID, adversaryID)
	if err != nil {
		return nil, err
	}
	before, err := daggerheart.NormalizeConditions(adversary.Conditions)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "invalid stored conditions: %v", err)
	}

	afterSet := make(map[string]struct{}, len(before)+len(normalizedAdd))
	for _, value := range before {
		afterSet[value] = struct{}{}
	}
	for _, value := range normalizedRemove {
		delete(afterSet, value)
	}
	for _, value := range normalizedAdd {
		afterSet[value] = struct{}{}
	}

	afterList := make([]string, 0, len(afterSet))
	for value := range afterSet {
		afterList = append(afterList, value)
	}
	after, err := daggerheart.NormalizeConditions(afterList)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "invalid condition set: %v", err)
	}

	added, removed := daggerheart.DiffConditions(before, after)
	if len(added) == 0 && len(removed) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "no condition changes to apply")
	}

	source := strings.TrimSpace(in.GetSource())
	var rollSeq *uint64
	if in.RollSeq != nil {
		value := in.GetRollSeq()
		rollSeq = &value
		rollEvent, err := s.stores.Event.GetEventBySeq(ctx, campaignID, value)
		if err != nil {
			return nil, handleDomainError(err)
		}
		if sessionID != "" && rollEvent.SessionID != sessionID {
			return nil, status.Error(codes.InvalidArgument, "roll seq does not match session")
		}
	}

	payload := daggerheart.AdversaryConditionChangePayload{
		AdversaryID:      adversaryID,
		ConditionsBefore: before,
		ConditionsAfter:  after,
		Added:            added,
		Removed:          removed,
		Source:           source,
		RollSeq:          rollSeq,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode condition payload: %v", err)
	}

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	if s.stores.Domain == nil {
		return nil, status.Error(codes.Internal, "domain engine is not configured")
	}
	_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:    campaignID,
		Type:          commandTypeDaggerheartAdversaryCondition,
		ActorType:     command.ActorTypeSystem,
		SessionID:     sessionID,
		RequestID:     requestID,
		InvocationID:  invocationID,
		EntityType:    "adversary",
		EntityID:      adversaryID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}, adapter, domainCommandApplyOptions{
		requireEvents:   true,
		missingEventMsg: "adversary condition change did not emit an event",
		applyErrMessage: "apply adversary condition event",
		executeErrMsg:   "execute domain command",
	})
	if err != nil {
		return nil, err
	}

	updated, err := s.stores.Daggerheart.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load daggerheart adversary: %v", err)
	}

	return &pb.DaggerheartApplyAdversaryConditionsResponse{
		AdversaryId: adversaryID,
		Adversary:   daggerheartAdversaryToProto(updated),
		Added:       daggerheartConditionsToProto(added),
		Removed:     daggerheartConditionsToProto(removed),
	}, nil
}

func (s *DaggerheartService) ApplyGmMove(ctx context.Context, in *pb.DaggerheartApplyGmMoveRequest) (*pb.DaggerheartApplyGmMoveResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply gm move request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
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
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	move := strings.TrimSpace(in.GetMove())
	if move == "" {
		return nil, status.Error(codes.InvalidArgument, "move is required")
	}
	if in.GetFearSpent() < 0 {
		return nil, status.Error(codes.InvalidArgument, "fear_spent must be non-negative")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart gm moves")
	}

	sess, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}
	if err := s.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
		return nil, err
	}

	gmFearBefore := 0
	gmFearAfter := 0
	if snap, err := s.stores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID); err == nil {
		gmFearBefore = snap.GMFear
		gmFearAfter = snap.GMFear
	}

	fearSpent := int(in.GetFearSpent())
	if fearSpent > 0 {
		before, after, err := applyGMFearSpend(gmFearBefore, fearSpent)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		gmFearBefore = before
		gmFearAfter = after

		adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
		requestID := grpcmeta.RequestIDFromContext(ctx)
		invocationID := grpcmeta.InvocationIDFromContext(ctx)
		if s.stores.Domain == nil {
			return nil, status.Error(codes.Internal, "domain engine is not configured")
		}
		payload := daggerheart.GMFearSetPayload{After: &gmFearAfter, Reason: "gm_move"}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "encode gm fear payload: %v", err)
		}
		_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
			CampaignID:    campaignID,
			Type:          commandTypeDaggerheartGMFearSet,
			ActorType:     command.ActorTypeSystem,
			SessionID:     sessionID,
			RequestID:     requestID,
			InvocationID:  invocationID,
			EntityType:    "campaign",
			EntityID:      campaignID,
			SystemID:      daggerheart.SystemID,
			SystemVersion: daggerheart.SystemVersion,
			PayloadJSON:   payloadJSON,
		}, adapter, domainCommandApplyOptions{
			requireEvents:   true,
			missingEventMsg: "gm fear update did not emit an event",
			applyErrMessage: "apply gm fear event",
			executeErrMsg:   "execute domain command",
		})
		if err != nil {
			return nil, err
		}
	}

	return &pb.DaggerheartApplyGmMoveResponse{
		CampaignId:   campaignID,
		GmFearBefore: int32(gmFearBefore),
		GmFearAfter:  int32(gmFearAfter),
	}, nil
}
