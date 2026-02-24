package daggerheart

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type outcomeApplication struct {
	service *DaggerheartService
}

func newOutcomeApplication(service *DaggerheartService) outcomeApplication {
	return outcomeApplication{service: service}
}

func (a outcomeApplication) runApplyRollOutcome(ctx context.Context, in *pb.ApplyRollOutcomeRequest) (*pb.ApplyRollOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply roll outcome request is required")
	}
	if err := a.service.requireDependencies(
		dependencyCampaignStore,
		dependencySessionStore,
		dependencyDaggerheartStore,
		dependencyEventStore,
		dependencyDomainEngine,
	); err != nil {
		return nil, err
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

	c, err := a.service.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(err)
	}
	if c.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		return nil, status.Error(codes.FailedPrecondition, "campaign system does not support daggerheart outcomes")
	}

	sess, err := a.service.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if sess.Status != session.StatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}

	rollEvent, err := a.service.stores.Event.GetEventBySeq(ctx, campaignID, in.GetRollSeq())
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

	alreadyApplied, err := a.service.outcomeAlreadyAppliedForSessionRequest(ctx, campaignID, sessionID, in.GetRollSeq(), rollRequestID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "check outcome applied: %v", err)
	}
	if alreadyApplied {
		if requiresComplication {
			if err := a.service.openGMConsequenceGate(ctx, campaignID, sessionID, in.GetRollSeq(), rollRequestID); err != nil {
				return nil, err
			}
		}
		return a.service.buildApplyRollOutcomeIdempotentResponse(ctx, campaignID, in.GetRollSeq(), targets, requiresComplication, gmFearDelta > 0)
	}
	if err := a.service.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
		return nil, err
	}

	changes := make([]action.OutcomeAppliedChange, 0)
	postEffects := make([]action.OutcomeAppliedEffect, 0)
	updatedStates := make([]*pb.OutcomeCharacterState, 0, len(targets))

	gmFearAlreadyApplied := false
	if gmFearDelta > 0 {
		gmFearAlreadyApplied, err = a.service.sessionRequestEventExists(
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
		currentSnap, err := a.service.stores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
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

		if err := a.service.executeAndApplyDaggerheartSystemCommand(ctx, daggerheartSystemCommandInput{
			campaignID:      campaignID,
			commandType:     commandTypeDaggerheartGMFearSet,
			sessionID:       sessionID,
			requestID:       rollRequestID,
			invocationID:    invocationID,
			correlationID:   rollRequestID,
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
		profile, err := a.service.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, target)
		if err != nil {
			return nil, handleDomainError(err)
		}
		state, err := a.service.stores.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, target)
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
			characterPatchAlreadyApplied, err := a.service.sessionRequestEventExists(
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
				if err := a.service.executeAndApplyDaggerheartSystemCommand(ctx, daggerheartSystemCommandInput{
					campaignID:      campaignID,
					commandType:     commandTypeDaggerheartCharacterStatePatch,
					sessionID:       sessionID,
					requestID:       rollRequestID,
					invocationID:    invocationID,
					correlationID:   rollRequestID,
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
			err = a.service.applyStressVulnerableCondition(ctx, applyStressVulnerableConditionInput{
				campaignID:    campaignID,
				sessionID:     sessionID,
				characterID:   target,
				conditions:    state.Conditions,
				stressBefore:  stressBefore,
				stressAfter:   stressAfter,
				stressMax:     profile.StressMax,
				rollSeq:       &rollSeq,
				requestID:     rollRequestID,
				correlationID: rollRequestID,
			})
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
		postEffects, err = a.service.buildGMConsequenceOutcomeEffects(ctx, campaignID, sessionID, in.GetRollSeq(), rollRequestID)
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

	cmd := commandbuild.CoreSystem(commandbuild.CoreSystemInput{
		CampaignID:    campaignID,
		Type:          commandTypeActionOutcomeApply,
		SessionID:     sessionID,
		RequestID:     rollRequestID,
		InvocationID:  invocationID,
		CorrelationID: rollRequestID,
		EntityType:    "outcome",
		EntityID:      rollRequestID,
		PayloadJSON:   payloadJSON,
	})
	_, err = a.service.executeAndApplyDomainCommand(ctx, cmd, a.service.stores.Applier(), domainCommandApplyOptions{
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
		currentSnap, err := a.service.stores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "load gm fear snapshot: %v", err)
		}
		value := int32(currentSnap.GMFear)
		response.Updated.GmFear = &value
	}

	return response, nil
}
