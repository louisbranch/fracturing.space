package outcometransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ApplyRollOutcome applies the durable side effects of a previously resolved
// roll and returns the updated projection state.
func (h *Handler) ApplyRollOutcome(ctx context.Context, in *pb.ApplyRollOutcomeRequest) (*pb.ApplyRollOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply roll outcome request is required")
	}
	if err := h.requireRollOutcomeDependencies(); err != nil {
		return nil, err
	}

	campaignID, err := validate.RequiredID(grpcmeta.CampaignIDFromContext(ctx), "campaign id")
	if err != nil {
		return nil, err
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		sessionID = strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx))
	}
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	if in.GetRollSeq() == 0 {
		return nil, status.Error(codes.InvalidArgument, "roll seq is required")
	}

	c, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(ctx, err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, handleDomainError(ctx, err)
	}
	if err := requireDaggerheartSystem(c, "campaign system does not support daggerheart outcomes"); err != nil {
		return nil, err
	}

	sess, err := h.deps.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, handleDomainError(ctx, err)
	}
	if sess.Status != session.StatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}

	rollEvent, err := h.deps.Event.GetEventBySeq(ctx, campaignID, in.GetRollSeq())
	if err != nil {
		return nil, handleDomainError(ctx, err)
	}
	if rollEvent.Type != eventTypeActionRollResolved {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not reference action.roll_resolved")
	}
	if rollEvent.SessionID.String() != sessionID {
		return nil, status.Error(codes.InvalidArgument, "roll seq does not match session")
	}

	var rollPayload action.RollResolvePayload
	if err := json.Unmarshal(rollEvent.PayloadJSON, &rollPayload); err != nil {
		return nil, grpcerror.Internal("decode roll payload", err)
	}
	rollMetadata, err := workflowtransport.DecodeRollSystemMetadata(rollPayload.SystemData)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid roll system_data: %v", err)
	}

	rollRequestID := strings.TrimSpace(rollPayload.RequestID)
	if rollRequestID == "" {
		rollRequestID = strings.TrimSpace(rollEvent.RequestID)
	}
	if rollRequestID == "" {
		return nil, status.Error(codes.InvalidArgument, "roll request id is required")
	}
	invocationID := grpcmeta.InvocationIDFromContext(ctx)

	rollKind := rollMetadata.RollKindOrDefault()
	generateHopeFear := workflowtransport.BoolValue(rollMetadata.HopeFear, rollKind != pb.RollKind_ROLL_KIND_REACTION)
	triggerGMMove := workflowtransport.BoolValue(rollMetadata.GMMove, rollKind != pb.RollKind_ROLL_KIND_REACTION)
	rollOutcome := rollMetadata.OutcomeOrFallback(rollPayload.Outcome)
	if rollOutcome == "" {
		return nil, status.Error(codes.InvalidArgument, "roll outcome is required")
	}
	if in.GetSwapHopeFear() {
		rollOutcome = swapHopeFearOutcomeCode(rollOutcome)
	}
	flavor := workflowtransport.OutcomeFlavorFromCode(rollOutcome)
	if flavor == "" {
		return nil, status.Error(codes.InvalidArgument, "roll outcome flavor is required")
	}
	if !generateHopeFear {
		flavor = ""
	}
	crit := workflowtransport.BoolValue(rollMetadata.Crit, strings.TrimSpace(rollOutcome) == pb.Outcome_CRITICAL_SUCCESS.String())

	targets := workflowtransport.NormalizeTargets(in.GetTargets())
	if len(targets) == 0 {
		rollerID := strings.TrimSpace(rollMetadata.CharacterID)
		if strings.TrimSpace(rollerID) == "" {
			return nil, status.Error(codes.InvalidArgument, "targets are required")
		}
		targets = []string{rollerID}
	}

	gmFearDelta := 0
	if triggerGMMove && flavor == outcomeFlavorFear && !crit {
		gmFearDelta = len(targets)
	}
	requiresComplication := flavor == outcomeFlavorFear && !crit && triggerGMMove

	alreadyApplied, err := h.outcomeAlreadyAppliedForSessionRequest(ctx, campaignID, sessionID, in.GetRollSeq(), rollRequestID)
	if err != nil {
		return nil, grpcerror.Internal("check outcome applied", err)
	}
	if alreadyApplied {
		if requiresComplication {
			if err := h.openGMConsequenceGate(ctx, campaignID, sessionID, sceneID, in.GetRollSeq(), rollRequestID); err != nil {
				return nil, err
			}
		}
		return h.buildApplyRollOutcomeIdempotentResponse(ctx, campaignID, in.GetRollSeq(), targets, requiresComplication, gmFearDelta > 0)
	}
	if err := h.ensureNoOpenSessionGate(ctx, campaignID, sessionID); err != nil {
		return nil, err
	}

	changes := make([]action.OutcomeAppliedChange, 0)
	postEffects := make([]action.OutcomeAppliedEffect, 0)
	updatedStates := make([]*pb.OutcomeCharacterState, 0, len(targets))

	gmFearAlreadyApplied := false
	if gmFearDelta > 0 {
		gmFearAlreadyApplied, err = h.sessionRequestEventExists(
			ctx,
			campaignID,
			sessionID,
			in.GetRollSeq(),
			rollRequestID,
			eventTypeDaggerheartGMFearChanged,
			campaignID,
		)
		if err != nil {
			return nil, grpcerror.Internal("check gm fear applied", err)
		}
	}

	if gmFearDelta > 0 && !gmFearAlreadyApplied {
		currentSnap, err := h.deps.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
		if err != nil {
			if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "load gm fear"); lookupErr != nil {
				return nil, lookupErr
			}
		}
		beforeFear := currentSnap.GMFear
		before, after, err := rules.ApplyGMFearGain(beforeFear, gmFearDelta)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "gm fear update invalid: %v", err)
		}

		payload := daggerheartpayload.GMFearSetPayload{After: &after}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return nil, grpcerror.Internal("encode gm fear payload", err)
		}

		if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
			CampaignID:      campaignID,
			CommandType:     commandTypeDaggerheartGMFearSet,
			SessionID:       sessionID,
			SceneID:         sceneID,
			RequestID:       rollRequestID,
			InvocationID:    invocationID,
			CorrelationID:   rollRequestID,
			EntityType:      "campaign",
			EntityID:        campaignID,
			PayloadJSON:     payloadJSON,
			MissingEventMsg: "gm fear update did not emit an event",
			ApplyErrMessage: "apply gm fear event",
		}); err != nil {
			return nil, err
		}

		changes = append(changes, action.OutcomeAppliedChange{Field: action.OutcomeFieldGMFear, Before: before, After: after})
	}

	for _, target := range targets {
		profile, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, target)
		if err != nil {
			return nil, handleDomainError(ctx, err)
		}
		subclassRules, err := h.activeSubclassRuleSummary(ctx, profile)
		if err != nil {
			return nil, err
		}
		state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, target)
		if err != nil {
			return nil, handleDomainError(ctx, err)
		}

		hopeBefore := state.Hope
		stressBefore := state.Stress
		hopeMax := state.HopeMax
		if hopeMax == 0 {
			hopeMax = mechanics.HopeMax
		}
		hopeAfter := hopeBefore
		stressAfter := stressBefore
		if generateHopeFear && flavor == outcomeFlavorHope {
			hopeAfter = clamp(hopeBefore+1, mechanics.HopeMin, hopeMax)
		}
		if generateHopeFear && flavor == outcomeFlavorFear && subclassRules.GainHopeOnFailureWithFearAmount > 0 {
			if success, known := workflowtransport.OutcomeSuccessFromCode(rollOutcome); known && !success {
				hopeAfter = clamp(hopeAfter+subclassRules.GainHopeOnFailureWithFearAmount, mechanics.HopeMin, hopeMax)
			}
		}
		if generateHopeFear && crit {
			stressAfter = clamp(stressBefore-1, mechanics.StressMin, profile.StressMax)
		}

		if hopeAfter != hopeBefore || stressAfter != stressBefore {
			characterPatchAlreadyApplied, err := h.sessionRequestEventExists(
				ctx,
				campaignID,
				sessionID,
				in.GetRollSeq(),
				rollRequestID,
				eventTypeDaggerheartCharacterStatePatch,
				target,
			)
			if err != nil {
				return nil, grpcerror.Internal("check character state patch applied", err)
			}
			if !characterPatchAlreadyApplied {
				payload := daggerheartpayload.CharacterStatePatchPayload{
					CharacterID:  ids.CharacterID(target),
					HopeBefore:   &hopeBefore,
					HopeAfter:    &hopeAfter,
					StressBefore: &stressBefore,
					StressAfter:  &stressAfter,
				}
				payloadJSON, err := json.Marshal(payload)
				if err != nil {
					return nil, grpcerror.Internal("encode character state payload", err)
				}
				if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
					CampaignID:      campaignID,
					CommandType:     commandTypeDaggerheartCharacterStatePatch,
					SessionID:       sessionID,
					SceneID:         sceneID,
					RequestID:       rollRequestID,
					InvocationID:    invocationID,
					CorrelationID:   rollRequestID,
					EntityType:      "character",
					EntityID:        target,
					PayloadJSON:     payloadJSON,
					MissingEventMsg: "character state update did not emit an event",
					ApplyErrMessage: "apply character state event",
				}); err != nil {
					return nil, err
				}
			}

			rollSeq := in.GetRollSeq()
			err = h.deps.ApplyStressVulnerableCondition(ctx, ApplyStressVulnerableConditionInput{
				CampaignID:    campaignID,
				SessionID:     sessionID,
				CharacterID:   target,
				Conditions:    state.Conditions,
				StressBefore:  stressBefore,
				StressAfter:   stressAfter,
				StressMax:     profile.StressMax,
				RollSeq:       &rollSeq,
				RequestID:     rollRequestID,
				CorrelationID: rollRequestID,
			})
			if err != nil {
				return nil, err
			}
		}

		if hopeAfter != hopeBefore {
			changes = append(changes, action.OutcomeAppliedChange{CharacterID: ids.CharacterID(target), Field: action.OutcomeFieldHope, Before: hopeBefore, After: hopeAfter})
		}
		if stressAfter != stressBefore {
			changes = append(changes, action.OutcomeAppliedChange{CharacterID: ids.CharacterID(target), Field: action.OutcomeFieldStress, Before: stressBefore, After: stressAfter})
		}
		updatedStates = append(updatedStates, &pb.OutcomeCharacterState{
			CharacterId: target,
			Hope:        int32(hopeAfter),
			Stress:      int32(stressAfter),
			Hp:          int32(state.Hp),
		})
	}

	if requiresComplication {
		postEffects, err = h.buildGMConsequenceOutcomeEffects(ctx, campaignID, sessionID, in.GetRollSeq(), rollRequestID)
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
		return nil, grpcerror.Internal("encode outcome payload", err)
	}

	if err := h.deps.ExecuteCoreCommand(ctx, CoreCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandTypeActionOutcomeApply,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       rollRequestID,
		InvocationID:    invocationID,
		CorrelationID:   rollRequestID,
		EntityType:      "outcome",
		EntityID:        rollRequestID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "outcome did not emit an event",
		ApplyErrMessage: "execute domain command",
	}); err != nil {
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
		currentSnap, err := h.deps.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
		if err != nil {
			return nil, grpcerror.Internal("load gm fear snapshot", err)
		}
		value := int32(currentSnap.GMFear)
		response.Updated.GmFear = &value
	}

	return response, nil
}
