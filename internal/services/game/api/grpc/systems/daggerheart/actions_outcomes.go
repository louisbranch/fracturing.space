package daggerheart

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *DaggerheartService) runApplyRollOutcome(ctx context.Context, in *pb.ApplyRollOutcomeRequest) (*pb.ApplyRollOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply roll outcome request is required")
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
	campaignID := strings.TrimSpace(grpcmeta.CampaignIDFromContext(ctx))
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		sessionID = strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	}
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if in.GetRollSeq() == 0 {
		return nil, status.Error(codes.InvalidArgument, "roll seq is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart outcomes")
	}

	sess, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}

	rollEvent, err := s.stores.Event.GetEventBySeq(ctx, campaignID, in.GetRollSeq())
	if err != nil {
		return nil, handleDomainError(err)
	}
	if rollEvent.Type != eventTypeActionRollResolved {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not reference action.roll_resolved")
	}
	if rollEvent.SessionID != sessionID {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not match session")
	}

	var rollPayload action.RollResolvePayload
	if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
		return nil, status.Errorf(codes.Internal, "decode roll payload: %v", err)
	}

	rollRequestID := strings.TrimSpace(rollPayload.RequestID)
	if rollRequestID == "" {
		rollRequestID = strings.TrimSpace(rollEvent.RequestID)
	}
	if rollRequestID == "" {
		return nil, status.Error(codes.InvalidArgument, "roll request id is required")
	}
	invocationID := grpcmeta.InvocationIDFromContext(ctx)

	rollSystemData := rollPayload.SystemData
	rollKind := rollKindFromSystemData(rollSystemData)
	generateHopeFear := boolFromSystemData(rollSystemData, "hope_fear", rollKind != pb.RollKind_ROLL_KIND_REACTION)
	triggerGMMove := boolFromSystemData(rollSystemData, "gm_move", rollKind != pb.RollKind_ROLL_KIND_REACTION)
	rollOutcome := outcomeFromSystemData(rollSystemData, rollPayload.Outcome)
	if rollOutcome == "" {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is required")
	}
	flavor := outcomeFlavorFromCode(rollOutcome)
	if flavor == "" {
		return nil, status.Error(codes.InvalidArgument, "roll outcome flavor is required")
	}
	if !generateHopeFear {
		flavor = ""
	}
	crit := critFromSystemData(rollSystemData, rollOutcome)

	targets := normalizeTargets(in.GetTargets())
	if len(targets) == 0 {
		rollerID := stringFromSystemData(rollSystemData, "character_id")
		if strings.TrimSpace(rollerID) == "" {
			return nil, status.Error(codes.InvalidArgument, "targets are required")
		}
		targets = []string{rollerID}
	}

	gmFearDelta := 0
	if triggerGMMove && flavor == "FEAR" && !crit {
		gmFearDelta = len(targets)
	}
	requiresComplication := flavor == "FEAR" && !crit && triggerGMMove

	alreadyApplied, err := s.outcomeAlreadyAppliedForSessionRequest(ctx, campaignID, sessionID, in.GetRollSeq(), rollRequestID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "check outcome applied: %v", err)
	}
	if alreadyApplied {
		if requiresComplication {
			if err := s.openGMConsequenceGate(ctx, campaignID, sessionID, in.GetRollSeq(), rollRequestID); err != nil {
				return nil, err
			}
		}
		return s.buildApplyRollOutcomeIdempotentResponse(ctx, campaignID, in.GetRollSeq(), targets, requiresComplication, gmFearDelta > 0)
	}
	if err := s.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
		return nil, err
	}

	changes := make([]action.OutcomeAppliedChange, 0)
	postEffects := make([]action.OutcomeAppliedEffect, 0)
	updatedStates := make([]*pb.OutcomeCharacterState, 0, len(targets))

	gmFearAlreadyApplied := false
	if gmFearDelta > 0 {
		gmFearAlreadyApplied, err = s.sessionRequestEventExists(
			ctx,
			campaignID,
			sessionID,
			in.GetRollSeq(),
			rollRequestID,
			eventTypeDaggerheartGMFearChanged,
			campaignID,
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "check gm fear applied: %v", err)
		}
	}

	if gmFearDelta > 0 && !gmFearAlreadyApplied {
		currentSnap, err := s.stores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return nil, status.Errorf(codes.Internal, "load gm fear: %v", err)
		}
		beforeFear := currentSnap.GMFear
		before, after, err := applyGMFearGain(beforeFear, gmFearDelta)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "gm fear update invalid: %v", err)
		}

		payload := daggerheart.GMFearSetPayload{After: &after}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "encode gm fear payload: %v", err)
		}

		if err := s.executeAndApplyDaggerheartSystemCommand(ctx, daggerheartSystemCommandInput{
			campaignID:      campaignID,
			commandType:     commandTypeDaggerheartGMFearSet,
			sessionID:       sessionID,
			requestID:       rollRequestID,
			invocationID:    invocationID,
			entityType:      "campaign",
			entityID:        campaignID,
			payloadJSON:     payloadJSON,
			missingEventMsg: "gm fear update did not emit an event",
			applyErrMessage: "apply gm fear event",
		}); err != nil {
			return nil, err
		}

		changes = append(changes, action.OutcomeAppliedChange{Field: action.OutcomeFieldGMFear, Before: before, After: after})
	}

	for _, target := range targets {
		profile, err := s.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, target)
		if err != nil {
			return nil, handleDomainError(err)
		}
		state, err := s.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, target)
		if err != nil {
			return nil, handleDomainError(err)
		}

		hopeBefore := state.Hope
		stressBefore := state.Stress
		hopeMax := state.HopeMax
		if hopeMax == 0 {
			hopeMax = daggerheart.HopeMax
		}
		hopeAfter := hopeBefore
		stressAfter := stressBefore
		if generateHopeFear && flavor == "HOPE" {
			hopeAfter = clamp(hopeBefore+1, daggerheart.HopeMin, hopeMax)
		}
		if generateHopeFear && crit {
			stressAfter = clamp(stressBefore-1, daggerheart.StressMin, profile.StressMax)
		}

		if hopeAfter != hopeBefore || stressAfter != stressBefore {
			characterPatchAlreadyApplied, err := s.sessionRequestEventExists(
				ctx,
				campaignID,
				sessionID,
				in.GetRollSeq(),
				rollRequestID,
				eventTypeDaggerheartCharacterStatePatch,
				target,
			)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "check character state patch applied: %v", err)
			}
			if !characterPatchAlreadyApplied {
				payload := daggerheart.CharacterStatePatchedPayload{
					CharacterID:  target,
					HopeBefore:   &hopeBefore,
					HopeAfter:    &hopeAfter,
					StressBefore: &stressBefore,
					StressAfter:  &stressAfter,
				}
				payloadJSON, err := json.Marshal(payload)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "encode character state payload: %v", err)
				}
				if err := s.executeAndApplyDaggerheartSystemCommand(ctx, daggerheartSystemCommandInput{
					campaignID:      campaignID,
					commandType:     commandTypeDaggerheartCharacterStatePatch,
					sessionID:       sessionID,
					requestID:       rollRequestID,
					invocationID:    invocationID,
					entityType:      "character",
					entityID:        target,
					payloadJSON:     payloadJSON,
					missingEventMsg: "character state update did not emit an event",
					applyErrMessage: "apply character state event",
				}); err != nil {
					return nil, err
				}
			}
			rollSeq := in.GetRollSeq()
			err = s.applyStressVulnerableCondition(
				ctx,
				campaignID,
				sessionID,
				target,
				state.Conditions,
				stressBefore,
				stressAfter,
				profile.StressMax,
				&rollSeq,
				rollRequestID,
			)
			if err != nil {
				return nil, err
			}
		}

		if hopeAfter != hopeBefore {
			changes = append(changes, action.OutcomeAppliedChange{CharacterID: target, Field: action.OutcomeFieldHope, Before: hopeBefore, After: hopeAfter})
		}
		if stressAfter != stressBefore {
			changes = append(changes, action.OutcomeAppliedChange{CharacterID: target, Field: action.OutcomeFieldStress, Before: stressBefore, After: stressAfter})
		}
		updatedStates = append(updatedStates, &pb.OutcomeCharacterState{
			CharacterId: target,
			Hope:        int32(hopeAfter),
			Stress:      int32(stressAfter),
			Hp:          int32(state.Hp),
		})
	}

	if requiresComplication {
		postEffects, err = s.buildGMConsequenceOutcomeEffects(ctx, campaignID, sessionID, in.GetRollSeq(), rollRequestID)
		if err != nil {
			return nil, err
		}
	}

	payload := action.OutcomeApplyPayload{
		RequestID:            rollRequestID,
		RollSeq:              in.GetRollSeq(),
		Targets:              targets,
		RequiresComplication: requiresComplication,
		AppliedChanges:       changes,
		PostEffects:          postEffects,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode outcome payload: %v", err)
	}

	if s.stores.Domain == nil {
		return nil, status.Error(codes.Internal, "domain engine is not configured")
	}
	_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:   campaignID,
		Type:         commandTypeActionOutcomeApply,
		ActorType:    command.ActorTypeSystem,
		SessionID:    sessionID,
		RequestID:    rollRequestID,
		InvocationID: invocationID,
		EntityType:   "outcome",
		EntityID:     rollRequestID,
		PayloadJSON:  payloadJSON,
	}, s.stores.Applier(), domainCommandApplyOptions{
		requireEvents:   true,
		missingEventMsg: "outcome did not emit an event",
		executeErrMsg:   "execute domain command",
	})
	if err != nil {
		return nil, err
	}

	response := &pb.ApplyRollOutcomeResponse{
		RollSeq:              in.GetRollSeq(),
		RequiresComplication: requiresComplication,
		Updated: &pb.OutcomeUpdated{
			CharacterStates: updatedStates,
		},
	}
	if gmFearDelta > 0 {
		currentSnap, err := s.stores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "load gm fear snapshot: %v", err)
		}
		value := int32(currentSnap.GMFear)
		response.Updated.GmFear = &value
	}

	return response, nil
}

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

	filterClause := "session_id = ? AND request_id = ? AND event_type = ?"
	filterParams := []any{sessionID, requestID, string(eventType)}
	if entityID != "" {
		filterClause += " AND entity_id = ?"
		filterParams = append(filterParams, entityID)
	}

	result, err := s.stores.Event.ListEventsPage(ctx, storage.ListEventsPageRequest{
		CampaignID:   campaignID,
		AfterSeq:     rollSeq - 1,
		PageSize:     1,
		FilterClause: filterClause,
		FilterParams: filterParams,
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
	requestID       string
	invocationID    string
	entityType      string
	entityID        string
	payloadJSON     []byte
	missingEventMsg string
	applyErrMessage string
}

func (s *DaggerheartService) executeAndApplyDaggerheartSystemCommand(ctx context.Context, in daggerheartSystemCommandInput) error {
	adapter := daggerheart.NewAdapter(s.stores.Daggerheart)
	cmd := commandbuild.DaggerheartSystem(commandbuild.DaggerheartSystemInput{
		CoreInput: commandbuild.CoreInput{
			CampaignID:   in.campaignID,
			Type:         in.commandType,
			ActorType:    command.ActorTypeSystem,
			SessionID:    in.sessionID,
			RequestID:    in.requestID,
			InvocationID: in.invocationID,
			EntityType:   in.entityType,
			EntityID:     in.entityID,
			PayloadJSON:  in.payloadJSON,
		},
	})
	_, err := s.executeAndApplyDomainCommand(ctx, cmd, adapter, domainCommandApplyOptions{
		requireEvents:   true,
		missingEventMsg: in.missingEventMsg,
		applyErrMessage: in.applyErrMessage,
		executeErrMsg:   "execute domain command",
	})
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

func (s *DaggerheartService) buildGMConsequenceOutcomeEffects(
	ctx context.Context,
	campaignID string,
	sessionID string,
	rollSeq uint64,
	rollRequestID string,
) ([]action.OutcomeAppliedEffect, error) {
	if s.stores.SessionGate == nil {
		return nil, status.Error(codes.Internal, "session gate store is not configured")
	}
	if s.stores.SessionSpotlight == nil {
		return nil, status.Error(codes.Internal, "session spotlight store is not configured")
	}

	effects := make([]action.OutcomeAppliedEffect, 0, 2)

	gateOpen := false
	if _, err := s.stores.SessionGate.GetOpenSessionGate(ctx, campaignID, sessionID); err == nil {
		gateOpen = true
	} else if !errors.Is(err, storage.ErrNotFound) {
		return nil, status.Errorf(codes.Internal, "check session gate: %v", err)
	}
	if !gateOpen {
		gateID, err := id.NewID()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "generate gate id: %v", err)
		}
		gateType, err := session.NormalizeGateType("gm_consequence")
		if err != nil {
			return nil, status.Errorf(codes.Internal, "normalize gate type: %v", err)
		}
		gatePayload := session.GateOpenedPayload{
			GateID:   gateID,
			GateType: gateType,
			Reason:   "gm_consequence",
			Metadata: map[string]any{
				"roll_seq":   rollSeq,
				"request_id": rollRequestID,
			},
		}
		payloadJSON, err := json.Marshal(gatePayload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "encode session gate payload: %v", err)
		}
		effects = append(effects, action.OutcomeAppliedEffect{
			Type:        "session.gate_opened",
			EntityType:  "session_gate",
			EntityID:    gateID,
			PayloadJSON: payloadJSON,
		})
	}

	spotlight, err := s.stores.SessionSpotlight.GetSessionSpotlight(ctx, campaignID, sessionID)
	if err == nil {
		if spotlight.SpotlightType == session.SpotlightTypeGM && strings.TrimSpace(spotlight.CharacterID) == "" {
			return effects, nil
		}
	} else if !errors.Is(err, storage.ErrNotFound) {
		return nil, status.Errorf(codes.Internal, "check session spotlight: %v", err)
	}

	spotlightPayload := session.SpotlightSetPayload{
		SpotlightType: string(session.SpotlightTypeGM),
	}
	payloadJSON, err := json.Marshal(spotlightPayload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode spotlight payload: %v", err)
	}
	effects = append(effects, action.OutcomeAppliedEffect{
		Type:        "session.spotlight_set",
		EntityType:  "session_spotlight",
		EntityID:    sessionID,
		PayloadJSON: payloadJSON,
	})

	return effects, nil
}

func (s *DaggerheartService) openGMConsequenceGate(ctx context.Context, campaignID, sessionID string, rollSeq uint64, rollRequestID string) error {
	if s.stores.SessionGate == nil {
		return status.Error(codes.Internal, "session gate store is not configured")
	}
	if s.stores.SessionSpotlight == nil {
		return status.Error(codes.Internal, "session spotlight store is not configured")
	}
	if s.stores.Domain == nil {
		return status.Error(codes.Internal, "domain engine is not configured")
	}

	gateOpen := false
	if _, err := s.stores.SessionGate.GetOpenSessionGate(ctx, campaignID, sessionID); err == nil {
		gateOpen = true
	} else if !errors.Is(err, storage.ErrNotFound) {
		return status.Errorf(codes.Internal, "check session gate: %v", err)
	}

	if !gateOpen {
		gateID, err := id.NewID()
		if err != nil {
			return status.Errorf(codes.Internal, "generate gate id: %v", err)
		}
		gateType, err := session.NormalizeGateType("gm_consequence")
		if err != nil {
			return status.Errorf(codes.Internal, "normalize gate type: %v", err)
		}
		metadata := map[string]any{
			"roll_seq":   rollSeq,
			"request_id": rollRequestID,
		}
		payload := session.GateOpenedPayload{
			GateID:   gateID,
			GateType: gateType,
			Reason:   "gm_consequence",
			Metadata: metadata,
		}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return status.Errorf(codes.Internal, "encode session gate payload: %v", err)
		}
		gateApplier := s.stores.Applier()
		_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
			CampaignID:   campaignID,
			Type:         commandTypeSessionGateOpen,
			ActorType:    command.ActorTypeSystem,
			SessionID:    sessionID,
			RequestID:    rollRequestID,
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "session_gate",
			EntityID:     gateID,
			PayloadJSON:  payloadJSON,
		}, gateApplier, domainCommandApplyOptions{
			requireEvents:   true,
			missingEventMsg: "session gate open did not emit an event",
			applyErrMessage: "apply session gate event",
			executeErrMsg:   "execute domain command",
		})
		if err != nil {
			return err
		}
	}

	spotlight, err := s.stores.SessionSpotlight.GetSessionSpotlight(ctx, campaignID, sessionID)
	if err == nil {
		if spotlight.SpotlightType == session.SpotlightTypeGM && strings.TrimSpace(spotlight.CharacterID) == "" {
			return nil
		}
	} else if !errors.Is(err, storage.ErrNotFound) {
		return status.Errorf(codes.Internal, "check session spotlight: %v", err)
	}

	spotlightPayload := session.SpotlightSetPayload{
		SpotlightType: string(session.SpotlightTypeGM),
	}
	spotlightPayloadJSON, err := json.Marshal(spotlightPayload)
	if err != nil {
		return status.Errorf(codes.Internal, "encode spotlight payload: %v", err)
	}
	spotlightApplier := s.stores.Applier()
	_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:   campaignID,
		Type:         commandTypeSessionSpotlightSet,
		ActorType:    command.ActorTypeSystem,
		SessionID:    sessionID,
		RequestID:    rollRequestID,
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		EntityType:   "session_spotlight",
		EntityID:     sessionID,
		PayloadJSON:  spotlightPayloadJSON,
	}, spotlightApplier, domainCommandApplyOptions{
		requireEvents:   true,
		missingEventMsg: "session spotlight set did not emit an event",
		applyErrMessage: "apply spotlight event",
		executeErrMsg:   "execute domain command",
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *DaggerheartService) runApplyAttackOutcome(ctx context.Context, in *pb.DaggerheartApplyAttackOutcomeRequest) (*pb.DaggerheartApplyAttackOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply attack outcome request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(grpcmeta.CampaignIDFromContext(ctx))
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		sessionID = strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	}
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if in.GetRollSeq() == 0 {
		return nil, status.Error(codes.InvalidArgument, "roll seq is required")
	}
	if len(in.GetTargets()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "targets are required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart attack outcomes")
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

	rollEvent, err := s.stores.Event.GetEventBySeq(ctx, campaignID, in.GetRollSeq())
	if err != nil {
		return nil, handleDomainError(err)
	}
	if rollEvent.Type != eventTypeActionRollResolved {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not reference action.roll_resolved")
	}
	if rollEvent.SessionID != sessionID {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not match session")
	}

	var rollPayload action.RollResolvePayload
	if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
		return nil, status.Errorf(codes.Internal, "decode roll payload: %v", err)
	}

	rollRequestID := strings.TrimSpace(rollPayload.RequestID)
	if rollRequestID == "" {
		rollRequestID = strings.TrimSpace(rollEvent.RequestID)
	}
	if rollRequestID == "" {
		return nil, status.Error(codes.InvalidArgument, "roll request id is required")
	}

	rollKind := rollKindFromSystemData(rollPayload.SystemData)
	if rollKind == pb.RollKind_ROLL_KIND_REACTION {
		return nil, status.Error(codes.FailedPrecondition, "roll seq references a reaction roll")
	}
	rollOutcome := outcomeFromSystemData(rollPayload.SystemData, rollPayload.Outcome)
	if rollOutcome == "" {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is required")
	}
	crit := critFromSystemData(rollPayload.SystemData, rollOutcome)
	flavor := outcomeFlavorFromCode(rollOutcome)
	if !boolFromSystemData(rollPayload.SystemData, "hope_fear", true) {
		flavor = ""
	}
	rollSuccess, ok := outcomeSuccessFromCode(rollOutcome)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is invalid")
	}
	attackerID := stringFromSystemData(rollPayload.SystemData, "character_id")
	if attackerID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}
	targets := normalizeTargets(in.GetTargets())
	if len(targets) == 0 {
		return nil, status.Error(codes.InvalidArgument, "targets are required")
	}

	return &pb.DaggerheartApplyAttackOutcomeResponse{
		RollSeq:     in.GetRollSeq(),
		CharacterId: attackerID,
		Targets:     targets,
		Result: &pb.DaggerheartAttackOutcomeResult{
			Outcome: outcomeCodeToProto(rollOutcome),
			Success: rollSuccess,
			Crit:    crit,
			Flavor:  flavor,
		},
	}, nil
}

func (s *DaggerheartService) runApplyAdversaryAttackOutcome(ctx context.Context, in *pb.DaggerheartApplyAdversaryAttackOutcomeRequest) (*pb.DaggerheartApplyAdversaryAttackOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply adversary attack outcome request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(grpcmeta.CampaignIDFromContext(ctx))
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		sessionID = strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	}
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if in.GetRollSeq() == 0 {
		return nil, status.Error(codes.InvalidArgument, "roll seq is required")
	}
	if len(in.GetTargets()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "targets are required")
	}
	if in.GetDifficulty() < 0 {
		return nil, status.Error(codes.InvalidArgument, "difficulty must be non-negative")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart adversary attack outcomes")
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

	rollEvent, err := s.stores.Event.GetEventBySeq(ctx, campaignID, in.GetRollSeq())
	if err != nil {
		return nil, handleDomainError(err)
	}
	if rollEvent.Type != eventTypeActionRollResolved {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not reference action.roll_resolved")
	}
	if rollEvent.SessionID != sessionID {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not match session")
	}

	var rollPayload action.RollResolvePayload
	if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
		return nil, status.Errorf(codes.Internal, "decode roll payload: %v", err)
	}
	rollKind := strings.TrimSpace(stringFromSystemData(rollPayload.SystemData, "roll_kind"))
	if rollKind != "adversary_roll" {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not reference an adversary roll")
	}
	adversaryID := strings.TrimSpace(stringFromSystemData(rollPayload.SystemData, "character_id"))
	if adversaryID == "" {
		adversaryID = strings.TrimSpace(stringFromSystemData(rollPayload.SystemData, "adversary_id"))
	}
	if adversaryID == "" {
		return nil, status.Error(codes.InvalidArgument, "adversary id is required")
	}

	rollRequestID := strings.TrimSpace(rollEvent.RequestID)
	if rollRequestID == "" {
		return nil, status.Error(codes.InvalidArgument, "roll request id is required")
	}

	targets := normalizeTargets(in.GetTargets())
	if len(targets) == 0 {
		return nil, status.Error(codes.InvalidArgument, "targets are required")
	}

	roll, rollHasValue := intFromSystemData(rollPayload.SystemData, "roll")
	if !rollHasValue {
		return nil, status.Error(codes.InvalidArgument, "roll payload missing roll")
	}
	_, hasModifier := intFromSystemData(rollPayload.SystemData, "modifier")
	if !hasModifier {
		return nil, status.Error(codes.InvalidArgument, "roll payload missing modifier")
	}
	total, hasTotal := intFromSystemData(rollPayload.SystemData, "total")
	if !hasTotal {
		return nil, status.Error(codes.InvalidArgument, "roll payload missing total")
	}
	difficulty := int(in.GetDifficulty())
	success := total >= difficulty
	crit := roll == 20

	return &pb.DaggerheartApplyAdversaryAttackOutcomeResponse{
		RollSeq:     in.GetRollSeq(),
		AdversaryId: adversaryID,
		Targets:     targets,
		Result: &pb.DaggerheartAdversaryAttackOutcomeResult{
			Success:    success,
			Crit:       crit,
			Roll:       int32(roll),
			Total:      int32(total),
			Difficulty: int32(difficulty),
		},
	}, nil
}

func (s *DaggerheartService) runApplyReactionOutcome(ctx context.Context, in *pb.DaggerheartApplyReactionOutcomeRequest) (*pb.DaggerheartApplyReactionOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply reaction outcome request is required")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}
	if s.stores.Domain == nil {
		return nil, status.Error(codes.Internal, "domain engine is not configured")
	}

	campaignID := strings.TrimSpace(grpcmeta.CampaignIDFromContext(ctx))
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		sessionID = strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	}
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if in.GetRollSeq() == 0 {
		return nil, status.Error(codes.InvalidArgument, "roll seq is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart reaction outcomes")
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

	rollEvent, err := s.stores.Event.GetEventBySeq(ctx, campaignID, in.GetRollSeq())
	if err != nil {
		return nil, handleDomainError(err)
	}
	if rollEvent.Type != eventTypeActionRollResolved {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not reference action.roll_resolved")
	}
	if rollEvent.SessionID != sessionID {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not match session")
	}

	var rollPayload action.RollResolvePayload
	if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
		return nil, status.Errorf(codes.Internal, "decode roll payload: %v", err)
	}

	rollRequestID := strings.TrimSpace(rollPayload.RequestID)
	if rollRequestID == "" {
		rollRequestID = strings.TrimSpace(rollEvent.RequestID)
	}
	if rollRequestID == "" {
		return nil, status.Error(codes.InvalidArgument, "roll request id is required")
	}

	rollKind := rollKindFromSystemData(rollPayload.SystemData)
	if rollKind != pb.RollKind_ROLL_KIND_REACTION {
		return nil, status.Error(codes.FailedPrecondition, "roll seq does not reference a reaction roll")
	}
	rollOutcome := outcomeFromSystemData(rollPayload.SystemData, rollPayload.Outcome)
	if rollOutcome == "" {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is required")
	}
	crit := critFromSystemData(rollPayload.SystemData, rollOutcome)
	rollSuccess, ok := outcomeSuccessFromCode(rollOutcome)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is invalid")
	}
	critNegates := boolFromSystemData(rollPayload.SystemData, "crit_negates", crit)
	effectsNegated := crit && critNegates
	actorID := stringFromSystemData(rollPayload.SystemData, "character_id")
	if actorID == "" {
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}

	return &pb.DaggerheartApplyReactionOutcomeResponse{
		RollSeq:     in.GetRollSeq(),
		CharacterId: actorID,
		Result: &pb.DaggerheartReactionOutcomeResult{
			Outcome:            outcomeCodeToProto(rollOutcome),
			Success:            rollSuccess,
			Crit:               crit,
			CritNegatesEffects: critNegates,
			EffectsNegated:     effectsNegated,
		},
	}, nil
}
