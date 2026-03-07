package daggerheart

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *DaggerheartService) runCreateCountdown(ctx context.Context, in *pb.DaggerheartCreateCountdownRequest) (*pb.DaggerheartCreateCountdownResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create countdown request is required")
	}
	if err := s.requireDependencies(dependencyCampaignStore, dependencySessionStore, dependencyDaggerheartStore, dependencyEventStore); err != nil {
		return nil, err
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	name := strings.TrimSpace(in.GetName())
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	kind, err := daggerheartCountdownKindFromProto(in.GetKind())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	direction, err := daggerheartCountdownDirectionFromProto(in.GetDirection())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	max := int(in.GetMax())
	if max <= 0 {
		return nil, status.Error(codes.InvalidArgument, "max must be positive")
	}
	current := int(in.GetCurrent())
	if current < 0 || current > max {
		return nil, status.Errorf(codes.InvalidArgument, "current must be in range 0..%d", max)
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart countdowns")
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

	countdownID := strings.TrimSpace(in.GetCountdownId())
	if countdownID == "" {
		countdownID, err = id.NewID()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "generate countdown id: %v", err)
		}
	}
	if _, err := s.stores.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID); err == nil {
		return nil, status.Error(codes.FailedPrecondition, "countdown already exists")
	} else if !errors.Is(err, storage.ErrNotFound) {
		return nil, handleDomainError(err)
	}

	payload := daggerheart.CountdownCreatePayload{
		CountdownID: countdownID,
		Name:        name,
		Kind:        kind,
		Current:     current,
		Max:         max,
		Direction:   direction,
		Looping:     in.GetLooping(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode countdown payload: %v", err)
	}

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:    campaignID,
		Type:          commandTypeDaggerheartCountdownCreate,
		ActorType:     command.ActorTypeSystem,
		SessionID:     sessionID,
		SceneID:       sceneID,
		RequestID:     requestID,
		InvocationID:  invocationID,
		EntityType:    "countdown",
		EntityID:      countdownID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}, adapter, domainwrite.RequireEventsWithDiagnostics("countdown create did not emit an event", "apply countdown created event"))
	if err != nil {
		return nil, err
	}

	countdown, err := s.stores.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load countdown: %v", err)
	}

	return &pb.DaggerheartCreateCountdownResponse{
		Countdown: daggerheartCountdownToProto(countdown),
	}, nil
}

func (s *DaggerheartService) runUpdateCountdown(ctx context.Context, in *pb.DaggerheartUpdateCountdownRequest) (*pb.DaggerheartUpdateCountdownResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update countdown request is required")
	}
	if err := s.requireDependencies(dependencyCampaignStore, dependencySessionStore, dependencyDaggerheartStore, dependencyEventStore); err != nil {
		return nil, err
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	countdownID := strings.TrimSpace(in.GetCountdownId())
	if countdownID == "" {
		return nil, status.Error(codes.InvalidArgument, "countdown id is required")
	}

	if in.Current == nil && in.GetDelta() == 0 {
		return nil, status.Error(codes.InvalidArgument, "delta or current is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart countdowns")
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

	storedCountdown, err := s.stores.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	var override *int
	if in.Current != nil {
		value := int(in.GetCurrent())
		override = &value
	}
	mutation, err := daggerheart.ResolveCountdownMutation(daggerheart.CountdownMutationInput{
		Countdown: daggerheartCountdownFromStorage(storedCountdown),
		Delta:     int(in.GetDelta()),
		Override:  override,
		Reason:    strings.TrimSpace(in.GetReason()),
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	payloadJSON, err := json.Marshal(mutation.Payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode countdown update payload: %v", err)
	}

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:    campaignID,
		Type:          commandTypeDaggerheartCountdownUpdate,
		ActorType:     command.ActorTypeSystem,
		SessionID:     sessionID,
		SceneID:       sceneID,
		RequestID:     requestID,
		InvocationID:  invocationID,
		EntityType:    "countdown",
		EntityID:      countdownID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}, adapter, domainwrite.RequireEventsWithDiagnostics("countdown update did not emit an event", "apply countdown update event"))
	if err != nil {
		return nil, err
	}

	updatedCountdown, err := s.stores.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load countdown: %v", err)
	}

	return &pb.DaggerheartUpdateCountdownResponse{
		Countdown: daggerheartCountdownToProto(updatedCountdown),
		Before:    int32(mutation.Update.Before),
		After:     int32(mutation.Update.After),
		Delta:     int32(mutation.Update.Delta),
	}, nil
}

func (s *DaggerheartService) runDeleteCountdown(ctx context.Context, in *pb.DaggerheartDeleteCountdownRequest) (*pb.DaggerheartDeleteCountdownResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "delete countdown request is required")
	}
	if err := s.requireDependencies(dependencyCampaignStore, dependencySessionStore, dependencyDaggerheartStore, dependencyEventStore); err != nil {
		return nil, err
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	countdownID := strings.TrimSpace(in.GetCountdownId())
	if countdownID == "" {
		return nil, status.Error(codes.InvalidArgument, "countdown id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart countdowns")
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

	if _, err := s.stores.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID); err != nil {
		return nil, handleDomainError(err)
	}

	payload := daggerheart.CountdownDeletePayload{
		CountdownID: countdownID,
		Reason:      strings.TrimSpace(in.GetReason()),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode countdown delete payload: %v", err)
	}

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:    campaignID,
		Type:          commandTypeDaggerheartCountdownDelete,
		ActorType:     command.ActorTypeSystem,
		SessionID:     sessionID,
		SceneID:       sceneID,
		RequestID:     requestID,
		InvocationID:  invocationID,
		EntityType:    "countdown",
		EntityID:      countdownID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}, adapter, domainwrite.RequireEventsWithDiagnostics("countdown delete did not emit an event", "apply countdown delete event"))
	if err != nil {
		return nil, err
	}

	return &pb.DaggerheartDeleteCountdownResponse{CountdownId: countdownID}, nil
}

func (s *DaggerheartService) runResolveBlazeOfGlory(ctx context.Context, in *pb.DaggerheartResolveBlazeOfGloryRequest) (*pb.DaggerheartResolveBlazeOfGloryResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "resolve blaze of glory request is required")
	}
	if err := s.requireDependencies(dependencyCampaignStore, dependencyDaggerheartStore, dependencyEventStore); err != nil {
		return nil, err
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
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart blaze of glory")
	}

	sessionID := strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	if err := s.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
		return nil, err
	}

	state, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if state.LifeState == "" {
		state.LifeState = daggerheart.LifeStateAlive
	}
	if state.LifeState == daggerheart.LifeStateDead {
		return nil, status.Error(codes.FailedPrecondition, "character is already dead")
	}
	if state.LifeState != daggerheart.LifeStateBlazeOfGlory {
		return nil, status.Error(codes.FailedPrecondition, "character is not in blaze of glory")
	}

	lifeStateBefore := state.LifeState
	lifeStateAfter := daggerheart.LifeStateDead
	patchPayload := daggerheart.CharacterStatePatchedPayload{
		CharacterID:     characterID,
		LifeStateBefore: &lifeStateBefore,
		LifeStateAfter:  &lifeStateAfter,
	}
	payloadJSON, err := json.Marshal(patchPayload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:    campaignID,
		Type:          commandTypeDaggerheartCharacterStatePatch,
		ActorType:     command.ActorTypeSystem,
		SessionID:     sessionID,
		SceneID:       sceneID,
		RequestID:     requestID,
		InvocationID:  invocationID,
		EntityType:    "character",
		EntityID:      characterID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   payloadJSON,
	}, adapter, domainwrite.RequireEventsWithDiagnostics("blaze of glory did not emit an event", "apply character state event"))
	if err != nil {
		return nil, err
	}
	updated, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load daggerheart state: %v", err)
	}
	if err := s.appendCharacterDeletedEvent(ctx, campaignID, characterID, daggerheart.LifeStateBlazeOfGlory); err != nil {
		return nil, err
	}

	return &pb.DaggerheartResolveBlazeOfGloryResponse{
		CharacterId: characterID,
		State:       daggerheartStateToProto(updated),
		Result: &pb.DaggerheartBlazeOfGloryResult{
			LifeState: daggerheartLifeStateToProto(lifeStateAfter),
		},
	}, nil
}
