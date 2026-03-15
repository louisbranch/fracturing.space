package recoverytransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ApplyTemporaryArmor records a temporary armor source for a character and
// returns the updated projected state.
func (h *Handler) ApplyTemporaryArmor(ctx context.Context, in *pb.DaggerheartApplyTemporaryArmorRequest) (CharacterStateResult, error) {
	if in == nil {
		return CharacterStateResult{}, status.Error(codes.InvalidArgument, "apply temporary armor request is required")
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
		in.GetSceneId(),
		"campaign system does not support daggerheart temporary armor",
	)
	if err != nil {
		return CharacterStateResult{}, err
	}
	if in.Armor == nil {
		return CharacterStateResult{}, status.Error(codes.InvalidArgument, "armor is required")
	}
	if _, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, mutation.campaignID, mutation.characterID); err != nil {
		return CharacterStateResult{}, grpcerror.HandleDomainError(err)
	}
	if _, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, mutation.campaignID, mutation.characterID); err != nil {
		return CharacterStateResult{}, grpcerror.HandleDomainError(err)
	}

	payloadJSON, err := json.Marshal(daggerheart.CharacterTemporaryArmorApplyPayload{
		CharacterID: ids.CharacterID(mutation.characterID),
		Source:      strings.TrimSpace(in.Armor.GetSource()),
		Duration:    strings.TrimSpace(in.Armor.GetDuration()),
		Amount:      int(in.Armor.GetAmount()),
		SourceID:    strings.TrimSpace(in.Armor.GetSourceId()),
	})
	if err != nil {
		return CharacterStateResult{}, grpcerror.Internal("encode payload", err)
	}
	if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
		CampaignID:      mutation.campaignID,
		CommandType:     commandids.DaggerheartCharacterTemporaryArmorApply,
		SessionID:       mutation.sessionID,
		SceneID:         mutation.sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "character",
		EntityID:        mutation.characterID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "temporary armor apply did not emit an event",
		ApplyErrMessage: "apply temporary armor event",
	}); err != nil {
		return CharacterStateResult{}, err
	}

	return h.loadUpdatedCharacterState(ctx, mutation.campaignID, mutation.characterID)
}
