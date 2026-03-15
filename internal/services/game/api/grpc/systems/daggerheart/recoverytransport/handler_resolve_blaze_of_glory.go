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

// ResolveBlazeOfGlory ends an in-progress blaze-of-glory state by patching the
// character to dead and appending the delete event owned by recovery transport.
func (h *Handler) ResolveBlazeOfGlory(ctx context.Context, in *pb.DaggerheartResolveBlazeOfGloryRequest) (BlazeResult, error) {
	if in == nil {
		return BlazeResult{}, status.Error(codes.InvalidArgument, "resolve blaze of glory request is required")
	}
	if err := h.requireDependencies(false); err != nil {
		return BlazeResult{}, err
	}
	if err := requireMutationPayload(in.GetCampaignId(), in.GetCharacterId()); err != nil {
		return BlazeResult{}, err
	}

	mutation, err := h.loadCharacterMutationContext(
		ctx,
		in.GetCampaignId(),
		in.GetCharacterId(),
		in.GetSceneId(),
		"campaign system does not support daggerheart blaze of glory",
	)
	if err != nil {
		return BlazeResult{}, err
	}
	state, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, mutation.campaignID, mutation.characterID)
	if err != nil {
		return BlazeResult{}, grpcerror.HandleDomainError(err)
	}
	if state.LifeState == "" {
		state.LifeState = daggerheart.LifeStateAlive
	}
	if state.LifeState == daggerheart.LifeStateDead {
		return BlazeResult{}, status.Error(codes.FailedPrecondition, "character is already dead")
	}
	if state.LifeState != daggerheart.LifeStateBlazeOfGlory {
		return BlazeResult{}, status.Error(codes.FailedPrecondition, "character is not in blaze of glory")
	}

	lifeStateBefore := state.LifeState
	lifeStateAfter := daggerheart.LifeStateDead
	payloadJSON, err := json.Marshal(daggerheart.CharacterStatePatchPayload{
		CharacterID:     ids.CharacterID(mutation.characterID),
		LifeStateBefore: &lifeStateBefore,
		LifeStateAfter:  &lifeStateAfter,
	})
	if err != nil {
		return BlazeResult{}, grpcerror.Internal("encode payload", err)
	}
	if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
		CampaignID:      mutation.campaignID,
		CommandType:     commandids.DaggerheartCharacterStatePatch,
		SessionID:       mutation.sessionID,
		SceneID:         mutation.sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "character",
		EntityID:        mutation.characterID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "blaze of glory did not emit an event",
		ApplyErrMessage: "apply character state event",
	}); err != nil {
		return BlazeResult{}, err
	}

	updated, err := h.loadUpdatedCharacterState(ctx, mutation.campaignID, mutation.characterID)
	if err != nil {
		return BlazeResult{}, err
	}
	if err := h.deps.AppendCharacterDeletedEvent(ctx, CharacterDeleteInput{
		CampaignID:  mutation.campaignID,
		CharacterID: mutation.characterID,
		Reason:      daggerheart.LifeStateBlazeOfGlory,
	}); err != nil {
		return BlazeResult{}, err
	}
	return BlazeResult{CharacterID: updated.CharacterID, State: updated.State, LifeState: lifeStateAfter}, nil
}
