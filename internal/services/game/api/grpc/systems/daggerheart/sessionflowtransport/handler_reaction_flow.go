package sessionflowtransport

import (
	"context"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) SessionReactionFlow(ctx context.Context, in *pb.SessionReactionFlowRequest) (*pb.SessionReactionFlowResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session reaction flow request is required")
	}
	if h.deps.SessionActionRoll == nil || h.deps.ApplyRollOutcome == nil || h.deps.ApplyReactionOutcome == nil {
		return nil, status.Error(codes.Internal, "session workflow dependencies are not configured")
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return nil, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	actorID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}
	trait, err := validate.RequiredID(in.GetTrait(), "trait")
	if err != nil {
		return nil, err
	}
	modifiers := append([]*pb.ActionRollModifier{}, in.GetModifiers()...)
	advantage := in.GetAdvantage()
	if h.deps.LoadCharacterState != nil {
		state, err := h.deps.LoadCharacterState(ctx, campaignID, actorID)
		if err != nil {
			return nil, err
		}
		subclassState := subclassStateFromProjection(state.SubclassState)
		if subclassState.TranscendenceTraitBonusValue > 0 && strings.EqualFold(strings.TrimSpace(subclassState.TranscendenceTraitBonusTarget), strings.TrimSpace(trait)) {
			modifiers = append(modifiers, &pb.ActionRollModifier{
				Source: "subclass_transcendence_trait",
				Value:  int32(subclassState.TranscendenceTraitBonusValue),
			})
		}
		if subclassState.ElementalChannel == daggerheartstate.ElementalChannelAir && strings.EqualFold(strings.TrimSpace(trait), "agility") {
			advantage++
		}
	}

	rollResp, err := h.deps.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:           campaignID,
		SessionId:            sessionID,
		SceneId:              sceneID,
		CharacterId:          actorID,
		Trait:                trait,
		RollKind:             pb.RollKind_ROLL_KIND_REACTION,
		Difficulty:           in.GetDifficulty(),
		Modifiers:            modifiers,
		Advantage:            advantage,
		Disadvantage:         in.GetDisadvantage(),
		Rng:                  in.GetReactionRng(),
		ReplaceHopeWithArmor: in.GetReplaceHopeWithArmor(),
	})
	if err != nil {
		return nil, err
	}

	ctxWithMeta := workflowtransport.WithCampaignSessionMetadata(ctx, campaignID, sessionID)
	rollOutcome, err := h.deps.ApplyRollOutcome(ctxWithMeta, &pb.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		SceneId:   sceneID,
		RollSeq:   rollResp.GetRollSeq(),
	})
	if err != nil {
		return nil, err
	}
	reactionOutcome, err := h.deps.ApplyReactionOutcome(ctxWithMeta, &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: sessionID,
		SceneId:   sceneID,
		RollSeq:   rollResp.GetRollSeq(),
	})
	if err != nil {
		return nil, err
	}

	return &pb.SessionReactionFlowResponse{
		ActionRoll:      rollResp,
		RollOutcome:     rollOutcome,
		ReactionOutcome: reactionOutcome,
	}, nil
}

// SessionGroupActionFlow runs the group action orchestration by resolving each
