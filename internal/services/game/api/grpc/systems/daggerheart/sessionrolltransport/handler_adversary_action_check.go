package sessionrolltransport

import (
	"context"
	"errors"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) SessionAdversaryActionCheck(ctx context.Context, in *pb.SessionAdversaryActionCheckRequest) (*pb.SessionAdversaryActionCheckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session adversary action check request is required")
	}
	if err := h.requireAdversaryActionCheckDependencies(); err != nil {
		return nil, err
	}

	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return nil, err
	}
	adversaryID, err := validate.RequiredID(in.GetAdversaryId(), "adversary id")
	if err != nil {
		return nil, err
	}
	if in.GetDifficulty() < 0 {
		return nil, status.Error(codes.InvalidArgument, "difficulty must be non-negative")
	}

	c, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(c, "campaign system does not support daggerheart adversary checks"); err != nil {
		return nil, err
	}

	sess, err := h.deps.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return nil, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if sess.Status != session.StatusActive {
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}
	if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return nil, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	adversary, err := h.deps.LoadAdversaryForSession(ctx, campaignID, sessionID, adversaryID)
	if err != nil {
		return nil, err
	}

	latestSeq, err := h.deps.Event.GetLatestEventSeq(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.Internal("load latest event seq", err)
	}
	rollSeq := latestSeq + 1

	difficulty := int(in.GetDifficulty())
	modifier, _ := normalizeActionModifiers(in.GetModifiers())
	if adversary.PendingExperience != nil {
		modifier += adversary.PendingExperience.Modifier
	}
	autoSuccess := !in.GetDramatic()
	success := true
	roll := 0
	total := 0
	requestID := grpcmeta.RequestIDFromContext(ctx)
	var rngResp *commonv1.RngResponse

	if !autoSuccess {
		seed, seedSource, rollMode, err := random.ResolveSeed(
			in.GetRng(),
			h.deps.SeedFunc,
			func(mode commonv1.RollMode) bool { return mode == commonv1.RollMode_REPLAY },
		)
		if err != nil {
			if errors.Is(err, random.ErrSeedOutOfRange()) {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
			return nil, grpcerror.Internal("failed to resolve seed", err)
		}
		result, err := dice.RollDice(dice.Request{
			Dice: []dice.Spec{{Sides: 20, Count: 1}},
			Seed: seed,
		})
		if err != nil {
			return nil, grpcerror.Internal("failed to roll adversary action die", err)
		}
		roll = result.Rolls[0].Results[0]
		total = roll + modifier
		success = total >= difficulty
		rngResp = &commonv1.RngResponse{
			SeedUsed:   uint64(seed),
			RngAlgo:    random.RngAlgoMathRandV1,
			SeedSource: seedSource,
			RollMode:   rollMode,
		}
	} else {
		total = modifier
	}
	if adversary.PendingExperience != nil && h.deps.ExecuteAdversaryFeatureApply != nil {
		if err := h.deps.ExecuteAdversaryFeatureApply(ctx, AdversaryFeatureApplyInput{
			CampaignID:              campaignID,
			SessionID:               sessionID,
			SceneID:                 sceneID,
			RequestID:               requestID,
			InvocationID:            grpcmeta.InvocationIDFromContext(ctx),
			Adversary:               adversary,
			FeatureID:               "experience:" + adversary.PendingExperience.Name,
			PendingExperienceBefore: adversary.PendingExperience,
			PendingExperienceAfter:  nil,
		}); err != nil {
			return nil, err
		}
	}

	return &pb.SessionAdversaryActionCheckResponse{
		RollSeq:     rollSeq,
		AutoSuccess: autoSuccess,
		Success:     success,
		Roll:        int32(roll),
		Modifier:    int32(modifier),
		Total:       int32(total),
		Rng:         rngResp,
	}, nil
}
