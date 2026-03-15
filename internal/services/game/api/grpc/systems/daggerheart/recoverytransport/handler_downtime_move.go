package recoverytransport

import (
	"context"
	"encoding/json"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ApplyDowntimeMove applies a downtime move against a character and returns the
// updated projected state after stress-condition side effects run.
func (h *Handler) ApplyDowntimeMove(ctx context.Context, in *pb.DaggerheartApplyDowntimeMoveRequest) (CharacterStateResult, error) {
	if in == nil {
		return CharacterStateResult{}, status.Error(codes.InvalidArgument, "apply downtime request is required")
	}
	if err := h.requireDependencies(false); err != nil {
		return CharacterStateResult{}, err
	}
	if err := requireMutationPayload(in.GetCampaignId(), in.GetCharacterId()); err != nil {
		return CharacterStateResult{}, err
	}

	mutation, err := h.loadCharacterMutationContext(
		ctx,
		in.GetCampaignId(),
		in.GetCharacterId(),
		"",
		"campaign system does not support daggerheart downtime",
	)
	if err != nil {
		return CharacterStateResult{}, err
	}
	if in.Move == nil {
		return CharacterStateResult{}, status.Error(codes.InvalidArgument, "move is required")
	}
	if in.Move.Move == pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_UNSPECIFIED {
		return CharacterStateResult{}, status.Error(codes.InvalidArgument, "downtime move is required")
	}

	profile, current, state, err := h.loadMutableCharacterState(ctx, mutation.campaignID, mutation.characterID)
	if err != nil {
		return CharacterStateResult{}, err
	}
	move, err := downtimeMoveFromProto(in.Move.Move)
	if err != nil {
		return CharacterStateResult{}, status.Error(codes.InvalidArgument, err.Error())
	}
	result := daggerheart.ApplyDowntimeMove(
		state,
		move,
		daggerheart.DowntimeOptions{PrepareWithGroup: in.Move.PrepareWithGroup},
	)
	hopeBefore, hopeAfter := result.HopeBefore, result.HopeAfter
	stressBefore, stressAfter := result.StressBefore, result.StressAfter
	armorBefore, armorAfter := result.ArmorBefore, result.ArmorAfter
	payloadJSON, err := json.Marshal(daggerheart.DowntimeMoveApplyPayload{
		CharacterID:  ids.CharacterID(mutation.characterID),
		Move:         downtimeMoveToString(move),
		HopeBefore:   &hopeBefore,
		HopeAfter:    &hopeAfter,
		StressBefore: &stressBefore,
		StressAfter:  &stressAfter,
		ArmorBefore:  &armorBefore,
		ArmorAfter:   &armorAfter,
	})
	if err != nil {
		return CharacterStateResult{}, grpcerror.Internal("encode payload", err)
	}
	if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
		CampaignID:      mutation.campaignID,
		CommandType:     commandids.DaggerheartDowntimeMoveApply,
		SessionID:       mutation.sessionID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "character",
		EntityID:        mutation.characterID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "downtime move did not emit an event",
		ApplyErrMessage: "apply downtime move event",
	}); err != nil {
		return CharacterStateResult{}, err
	}
	if err := h.deps.ApplyStressConditionChange(ctx, StressConditionInput{
		CampaignID:   mutation.campaignID,
		SessionID:    mutation.sessionID,
		CharacterID:  mutation.characterID,
		Conditions:   current.Conditions,
		StressBefore: stressBefore,
		StressAfter:  stressAfter,
		StressMax:    profile.StressMax,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
	}); err != nil {
		return CharacterStateResult{}, err
	}

	return h.loadUpdatedCharacterState(ctx, mutation.campaignID, mutation.characterID)
}
