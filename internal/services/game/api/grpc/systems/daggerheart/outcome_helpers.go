package daggerheart

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *DaggerheartService) outcomeAlreadyAppliedForSessionRequest(ctx context.Context, campaignID, sessionID string, rollSeq uint64, requestID string) (bool, error) {
	return s.sessionRequestEventExists(
		ctx,
		campaignID,
		sessionID,
		rollSeq,
		requestID,
		eventTypeActionOutcomeApplied,
		requestID,
	)
}

func (s *DaggerheartService) sessionRequestEventExists(
	ctx context.Context,
	campaignID string,
	sessionID string,
	rollSeq uint64,
	requestID string,
	eventType event.Type,
	entityID string,
) (bool, error) {
	requestID = strings.TrimSpace(requestID)
	entityID = strings.TrimSpace(entityID)
	if rollSeq == 0 || requestID == "" {
		return false, nil
	}

	result, err := s.stores.Event.ListEventsPage(ctx, storage.ListEventsPageRequest{
		CampaignID: campaignID,
		AfterSeq:   rollSeq - 1,
		PageSize:   1,
		Filter: storage.EventQueryFilter{
			SessionID: sessionID,
			RequestID: requestID,
			EventType: string(eventType),
			EntityID:  entityID,
		},
	})
	if err != nil {
		return false, err
	}

	return len(result.Events) > 0, nil
}

type daggerheartSystemCommandInput struct {
	campaignID      string
	commandType     command.Type
	sessionID       string
	sceneID         string
	requestID       string
	invocationID    string
	correlationID   string
	entityType      string
	entityID        string
	payloadJSON     []byte
	missingEventMsg string
	applyErrMessage string
}

func (s *DaggerheartService) executeAndApplyDaggerheartSystemCommand(ctx context.Context, in daggerheartSystemCommandInput) error {
	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	cmd := commandbuild.SystemCommand(commandbuild.SystemCommandInput{
		CampaignID:    in.campaignID,
		Type:          in.commandType,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		SessionID:     in.sessionID,
		SceneID:       in.sceneID,
		RequestID:     in.requestID,
		InvocationID:  in.invocationID,
		CorrelationID: in.correlationID,
		EntityType:    in.entityType,
		EntityID:      in.entityID,
		PayloadJSON:   in.payloadJSON,
	})
	_, err := s.executeAndApplyDomainCommand(ctx, cmd, adapter, domainwrite.RequireEventsWithDiagnostics(in.missingEventMsg, in.applyErrMessage))
	if err != nil {
		return err
	}
	return nil
}

func (s *DaggerheartService) buildApplyRollOutcomeIdempotentResponse(
	ctx context.Context,
	campaignID string,
	rollSeq uint64,
	targets []string,
	requiresComplication bool,
	includeGMFear bool,
) (*pb.ApplyRollOutcomeResponse, error) {
	updatedStates := make([]*pb.OutcomeCharacterState, 0, len(targets))
	for _, target := range targets {
		state, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, target)
		if err != nil {
			return nil, handleDomainError(err)
		}
		updatedStates = append(updatedStates, &pb.OutcomeCharacterState{
			CharacterId: target,
			Hope:        int32(state.Hope),
			Stress:      int32(state.Stress),
			Hp:          int32(state.Hp),
		})
	}

	response := &pb.ApplyRollOutcomeResponse{
		RollSeq:              rollSeq,
		RequiresComplication: requiresComplication,
		Updated: &pb.OutcomeUpdated{
			CharacterStates: updatedStates,
		},
	}
	if !includeGMFear {
		return response, nil
	}

	currentSnap, err := s.stores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load gm fear snapshot: %v", err)
	}
	value := int32(currentSnap.GMFear)
	response.Updated.GmFear = &value
	return response, nil
}

// gmConsequenceResolution describes what gate/spotlight changes are needed
// when a GM consequence triggers. Both buildGMConsequenceOutcomeEffects and
// openGMConsequenceGate share this resolution to avoid duplicate logic.
type gmConsequenceResolution struct {
	needsGate            bool
	gateID               string
	gatePayloadJSON      []byte
	needsSpotlight       bool
	spotlightPayloadJSON []byte
}

func (s *DaggerheartService) resolveGMConsequence(
	ctx context.Context,
	campaignID, sessionID string,
	rollSeq uint64,
	rollRequestID string,
) (gmConsequenceResolution, error) {
	if err := s.requireDependencies(dependencySessionGateStore, dependencySessionSpotlightStore); err != nil {
		return gmConsequenceResolution{}, err
	}

	var res gmConsequenceResolution

	// Check whether a session gate is already open.
	gateOpen := false
	if _, err := s.stores.SessionGate.GetOpenSessionGate(ctx, campaignID, sessionID); err == nil {
		gateOpen = true
	} else if !errors.Is(err, storage.ErrNotFound) {
		return res, status.Errorf(codes.Internal, "check session gate: %v", err)
	}
	if !gateOpen {
		gateID, err := id.NewID()
		if err != nil {
			return res, status.Errorf(codes.Internal, "generate gate id: %v", err)
		}
		gateType, err := session.NormalizeGateType("gm_consequence")
		if err != nil {
			return res, status.Errorf(codes.Internal, "normalize gate type: %v", err)
		}
		payload := session.GateOpenedPayload{
			GateID:   gateID,
			GateType: gateType,
			Reason:   "gm_consequence",
			Metadata: map[string]any{
				"roll_seq":   rollSeq,
				"request_id": rollRequestID,
			},
		}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return res, status.Errorf(codes.Internal, "encode session gate payload: %v", err)
		}
		res.needsGate = true
		res.gateID = gateID
		res.gatePayloadJSON = payloadJSON
	}

	// Check whether the spotlight is already GM-focused.
	spotlight, err := s.stores.SessionSpotlight.GetSessionSpotlight(ctx, campaignID, sessionID)
	if err == nil {
		if spotlight.SpotlightType == session.SpotlightTypeGM && strings.TrimSpace(spotlight.CharacterID) == "" {
			return res, nil
		}
	} else if !errors.Is(err, storage.ErrNotFound) {
		return res, status.Errorf(codes.Internal, "check session spotlight: %v", err)
	}

	spotlightPayload := session.SpotlightSetPayload{
		SpotlightType: string(session.SpotlightTypeGM),
	}
	payloadJSON, err := json.Marshal(spotlightPayload)
	if err != nil {
		return res, status.Errorf(codes.Internal, "encode spotlight payload: %v", err)
	}
	res.needsSpotlight = true
	res.spotlightPayloadJSON = payloadJSON
	return res, nil
}

func (s *DaggerheartService) buildGMConsequenceOutcomeEffects(
	ctx context.Context,
	campaignID string,
	sessionID string,
	rollSeq uint64,
	rollRequestID string,
) ([]action.OutcomeAppliedEffect, error) {
	res, err := s.resolveGMConsequence(ctx, campaignID, sessionID, rollSeq, rollRequestID)
	if err != nil {
		return nil, err
	}

	effects := make([]action.OutcomeAppliedEffect, 0, 2)
	if res.needsGate {
		effects = append(effects, action.OutcomeAppliedEffect{
			Type:        "session.gate_opened",
			EntityType:  "session_gate",
			EntityID:    res.gateID,
			PayloadJSON: res.gatePayloadJSON,
		})
	}
	if res.needsSpotlight {
		effects = append(effects, action.OutcomeAppliedEffect{
			Type:        "session.spotlight_set",
			EntityType:  "session_spotlight",
			EntityID:    sessionID,
			PayloadJSON: res.spotlightPayloadJSON,
		})
	}
	return effects, nil
}

func (s *DaggerheartService) openGMConsequenceGate(ctx context.Context, campaignID, sessionID, sceneID string, rollSeq uint64, rollRequestID string) error {
	res, err := s.resolveGMConsequence(ctx, campaignID, sessionID, rollSeq, rollRequestID)
	if err != nil {
		return err
	}

	if res.needsGate {
		gateApplier := s.stores.Applier()
		gateCmd := commandbuild.CoreSystem(commandbuild.CoreSystemInput{
			CampaignID:   campaignID,
			Type:         commandTypeSessionGateOpen,
			SessionID:    sessionID,
			SceneID:      sceneID,
			RequestID:    rollRequestID,
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "session_gate",
			EntityID:     res.gateID,
			PayloadJSON:  res.gatePayloadJSON,
		})
		if _, err := s.executeAndApplyDomainCommand(ctx, gateCmd, gateApplier, domainwrite.RequireEventsWithDiagnostics("session gate open did not emit an event", "apply session gate event")); err != nil {
			return err
		}
	}

	if res.needsSpotlight {
		spotlightApplier := s.stores.Applier()
		spotlightCmd := commandbuild.CoreSystem(commandbuild.CoreSystemInput{
			CampaignID:   campaignID,
			Type:         commandTypeSessionSpotlightSet,
			SessionID:    sessionID,
			SceneID:      sceneID,
			RequestID:    rollRequestID,
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "session_spotlight",
			EntityID:     sessionID,
			PayloadJSON:  res.spotlightPayloadJSON,
		})
		if _, err := s.executeAndApplyDomainCommand(ctx, spotlightCmd, spotlightApplier, domainwrite.RequireEventsWithDiagnostics("session spotlight set did not emit an event", "apply spotlight event")); err != nil {
			return err
		}
	}

	return nil
}
