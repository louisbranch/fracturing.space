package recoverytransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) ApplyTemporaryArmor(ctx context.Context, in *pb.DaggerheartApplyTemporaryArmorRequest) (CharacterStateResult, error) {
	if in == nil {
		return CharacterStateResult{}, status.Error(codes.InvalidArgument, "apply temporary armor request is required")
	}
	if err := h.requireDependencies(false); err != nil {
		return CharacterStateResult{}, err
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return CharacterStateResult{}, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return CharacterStateResult{}, err
	}
	record, err := h.deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return CharacterStateResult{}, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return CharacterStateResult{}, handleDomainError(err)
	}
	if err := requireDaggerheartSystem(record, "campaign system does not support daggerheart temporary armor"); err != nil {
		return CharacterStateResult{}, err
	}
	sessionID, err := validate.RequiredID(grpcmeta.SessionIDFromContext(ctx), "session id")
	if err != nil {
		return CharacterStateResult{}, err
	}
	sceneID := strings.TrimSpace(in.GetSceneId())
	if err := ensureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
		return CharacterStateResult{}, err
	}
	if in.Armor == nil {
		return CharacterStateResult{}, status.Error(codes.InvalidArgument, "armor is required")
	}
	if _, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID); err != nil {
		return CharacterStateResult{}, handleDomainError(err)
	}
	if _, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID); err != nil {
		return CharacterStateResult{}, handleDomainError(err)
	}
	payloadJSON, err := json.Marshal(daggerheartpayload.CharacterTemporaryArmorApplyPayload{
		CharacterID: ids.CharacterID(characterID),
		Source:      strings.TrimSpace(in.Armor.GetSource()),
		Duration:    strings.TrimSpace(in.Armor.GetDuration()),
		Amount:      int(in.Armor.GetAmount()),
		SourceID:    strings.TrimSpace(in.Armor.GetSourceId()),
	})
	if err != nil {
		return CharacterStateResult{}, grpcerror.Internal("encode payload", err)
	}
	if err := h.deps.ExecuteSystemCommand(ctx, SystemCommandInput{
		CampaignID:      campaignID,
		CommandType:     commandids.DaggerheartCharacterTemporaryArmorApply,
		SessionID:       sessionID,
		SceneID:         sceneID,
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		EntityType:      "character",
		EntityID:        characterID,
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "temporary armor apply did not emit an event",
		ApplyErrMessage: "apply temporary armor event",
	}); err != nil {
		return CharacterStateResult{}, err
	}
	updated, err := h.deps.Daggerheart.GetDaggerheartCharacterState(ctx, campaignID, characterID)
	if err != nil {
		return CharacterStateResult{}, grpcerror.Internal("load daggerheart state", err)
	}
	return CharacterStateResult{CharacterID: characterID, State: updated}, nil
}
