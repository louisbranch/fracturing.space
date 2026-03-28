package sessionrolltransport

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) SessionAdversaryAttackRoll(ctx context.Context, in *pb.SessionAdversaryAttackRollRequest) (*pb.SessionAdversaryAttackRollResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session adversary attack roll request is required")
	}
	if err := h.requireAdversaryRollDependencies(); err != nil {
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
	sceneID := strings.TrimSpace(in.GetSceneId())
	adversaryID, err := validate.RequiredID(in.GetAdversaryId(), "adversary id")
	if err != nil {
		return nil, err
	}

	c, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return nil, grpcerror.HandleDomainErrorContext(ctx, err)
	}
	if err := daggerheartguard.RequireDaggerheartSystem(c, "campaign system does not support daggerheart adversary rolls"); err != nil {
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
	if _, err := h.deps.LoadAdversaryForSession(ctx, campaignID, sessionID, adversaryID); err != nil {
		return nil, err
	}

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

	advantage := int(in.GetAdvantage())
	disadvantage := int(in.GetDisadvantage())
	modifier, modifierList := normalizeActionModifiers(in.GetModifiers())
	if advantage > 0 && disadvantage > 0 {
		advantage = 0
		disadvantage = 0
	}
	rollCount := 1
	if advantage > 0 || disadvantage > 0 {
		rollCount = 2
	}

	result, err := dice.RollDice(dice.Request{
		Dice: []dice.Spec{{Sides: 20, Count: rollCount}},
		Seed: seed,
	})
	if err != nil {
		return nil, grpcerror.Internal("failed to roll adversary die", err)
	}
	rolls := result.Rolls[0].Results
	selected := rolls[0]
	if rollCount == 2 {
		if advantage > 0 && rolls[1] > selected {
			selected = rolls[1]
		}
		if disadvantage > 0 && rolls[1] < selected {
			selected = rolls[1]
		}
	}
	total := selected + modifier

	latestSeq, err := h.deps.Event.GetLatestEventSeq(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.Internal("load latest event seq", err)
	}
	rollSeq := latestSeq + 1

	requestID := grpcmeta.RequestIDFromContext(ctx)
	payloadJSON, err := json.Marshal(action.RollResolvePayload{
		RequestID: requestID,
		RollSeq:   rollSeq,
		Results: map[string]any{
			"rolls":                       rolls,
			workflowtransport.KeyRoll:     selected,
			workflowtransport.KeyModifier: modifier,
			workflowtransport.KeyTotal:    total,
			"advantage":                   advantage,
			"disadvantage":                disadvantage,
			"modifiers":                   modifierList,
		},
		SystemData: workflowtransport.RollSystemMetadata{
			CharacterID:  adversaryID,
			AdversaryID:  adversaryID,
			RollKind:     "adversary_roll",
			Roll:         workflowtransport.IntPtr(selected),
			Modifier:     workflowtransport.IntPtr(modifier),
			Total:        workflowtransport.IntPtr(total),
			Advantage:    workflowtransport.IntPtr(advantage),
			Disadvantage: workflowtransport.IntPtr(disadvantage),
			Modifiers:    modifierList,
		}.MapValue(),
	})
	if err != nil {
		return nil, grpcerror.Internal("encode adversary roll payload", err)
	}

	rollSeqValue, err := h.deps.ExecuteAdversaryRollResolve(ctx, RollResolveInput{
		CampaignID:      campaignID,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       requestID,
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "adversary",
		EntityID:        adversaryID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "adversary roll did not emit an event",
	})
	if err != nil {
		return nil, err
	}

	rollValues := make([]int32, 0, len(rolls))
	for _, roll := range rolls {
		rollValues = append(rollValues, int32(roll))
	}

	return &pb.SessionAdversaryAttackRollResponse{
		RollSeq: rollSeqValue,
		Roll:    int32(selected),
		Total:   int32(total),
		Rolls:   rollValues,
		Rng: &commonv1.RngResponse{
			SeedUsed:   uint64(seed),
			RngAlgo:    random.RngAlgoMathRandV1,
			SeedSource: seedSource,
			RollMode:   rollMode,
		},
	}, nil
}
