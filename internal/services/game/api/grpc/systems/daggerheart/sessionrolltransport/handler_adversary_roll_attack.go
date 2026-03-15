package sessionrolltransport

import (
	"context"
	"encoding/json"
	"errors"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/dice"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) sessionAdversaryAttackRoll(ctx context.Context, in *pb.SessionAdversaryAttackRollRequest) (*pb.SessionAdversaryAttackRollResponse, error) {
	if err := h.requireAdversaryRollDependencies(); err != nil {
		return nil, err
	}

	rollCtx, err := h.loadAdversaryRollContext(
		ctx,
		in.GetCampaignId(),
		in.GetSessionId(),
		in.GetSceneId(),
		in.GetAdversaryId(),
		"campaign system does not support daggerheart rolls",
	)
	if err != nil {
		return nil, err
	}

	if in.GetAdvantage() < 0 {
		return nil, status.Error(codes.InvalidArgument, "advantage cannot be negative")
	}
	if in.GetDisadvantage() < 0 {
		return nil, status.Error(codes.InvalidArgument, "disadvantage cannot be negative")
	}

	rollSeq, err := h.nextAdversaryRollSeq(ctx, rollCtx.CampaignID)
	if err != nil {
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

	modifier := int(in.GetAttackModifier())
	advantage := int(in.GetAdvantage())
	disadvantage := int(in.GetDisadvantage())
	roll, rolls, total, err := resolveD20RollWithAdvantage(seed, modifier, advantage, disadvantage)
	if err != nil {
		return nil, grpcerror.Internal("failed to resolve attack roll", err)
	}

	requestID := metadata.RequestIDFromContext(ctx)
	invocationID := metadata.InvocationIDFromContext(ctx)
	payloadJSON, err := json.Marshal(action.RollResolvePayload{
		RequestID: requestID,
		RollSeq:   rollSeq,
		Results: map[string]any{
			"rolls":                       rolls,
			workflowtransport.KeyRoll:     roll,
			workflowtransport.KeyModifier: modifier,
			workflowtransport.KeyTotal:    total,
			"advantage":                   advantage,
			"disadvantage":                disadvantage,
		},
		SystemData: workflowtransport.RollSystemMetadata{
			AdversaryID:  rollCtx.AdversaryID,
			CharacterID:  rollCtx.AdversaryID,
			RollKind:     "adversary_roll",
			Roll:         workflowtransport.IntPtr(roll),
			Modifier:     workflowtransport.IntPtr(modifier),
			Total:        workflowtransport.IntPtr(total),
			Advantage:    workflowtransport.IntPtr(advantage),
			Disadvantage: workflowtransport.IntPtr(disadvantage),
		}.MapValue(),
	})
	if err != nil {
		return nil, grpcerror.Internal("encode payload", err)
	}

	rollSeqValue, err := h.deps.ExecuteAdversaryRollResolve(ctx, RollResolveInput{
		CampaignID:      rollCtx.CampaignID,
		SessionID:       rollCtx.SessionID,
		SceneID:         rollCtx.SceneID,
		RequestID:       requestID,
		InvocationID:    invocationID,
		EntityType:      "adversary",
		EntityID:        rollCtx.AdversaryID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "adversary attack roll did not emit an event",
	})
	if err != nil {
		return nil, err
	}

	return &pb.SessionAdversaryAttackRollResponse{
		RollSeq: rollSeqValue,
		Roll:    int32(roll),
		Total:   int32(total),
		Rolls:   toInt32Slice(rolls),
		Rng: &commonv1.RngResponse{
			SeedUsed:   uint64(seed),
			RngAlgo:    random.RngAlgoMathRandV1,
			SeedSource: seedSource,
			RollMode:   rollMode,
		},
	}, nil
}

func toInt32Slice(values []int) []int32 {
	if len(values) == 0 {
		return nil
	}
	converted := make([]int32, 0, len(values))
	for _, value := range values {
		converted = append(converted, int32(value))
	}
	return converted
}

func resolveD20RollWithAdvantage(seed int64, modifier int, advantage int, disadvantage int) (int, []int, int, error) {
	rollCount := 1
	if advantage > 0 || disadvantage > 0 {
		rollCount = 2
	}

	result, err := dice.RollDice(dice.Request{
		Dice: []dice.Spec{{Sides: 20, Count: rollCount}},
		Seed: seed,
	})
	if err != nil {
		return 0, nil, 0, err
	}

	if len(result.Rolls) != 1 {
		return 0, nil, 0, status.Error(codes.Internal, "invalid d20 roll result")
	}
	rolls := append([]int(nil), result.Rolls[0].Results...)
	if len(rolls) == 0 {
		return 0, nil, 0, status.Error(codes.Internal, "empty d20 roll result")
	}

	roll := rolls[0]
	if len(rolls) > 1 {
		switch {
		case advantage > disadvantage:
			for _, candidate := range rolls {
				if candidate > roll {
					roll = candidate
				}
			}
		case disadvantage > advantage:
			for _, candidate := range rolls {
				if candidate < roll {
					roll = candidate
				}
			}
		}
	}

	return roll, rolls, roll + modifier, nil
}
