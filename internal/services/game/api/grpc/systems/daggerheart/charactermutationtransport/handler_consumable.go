package charactermutationtransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UseConsumable records consumable usage for one character via the shared
// mutation pipeline.
func (h *Handler) UseConsumable(ctx context.Context, in *pb.DaggerheartUseConsumableRequest) (*pb.DaggerheartUseConsumableResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "use consumable request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}
	consumableID, err := validate.RequiredID(in.GetConsumableId(), "consumable id")
	if err != nil {
		return nil, err
	}
	if _, err := h.validateCharacterPreconditions(ctx, campaignID, characterID, "consumable use"); err != nil {
		return nil, err
	}

	payloadJSON, err := json.Marshal(daggerheartpayload.ConsumableUsePayload{
		CharacterID:    ids.CharacterID(characterID),
		ConsumableID:   consumableID,
		QuantityBefore: int(in.GetQuantityBefore()),
		QuantityAfter:  int(in.GetQuantityAfter()),
	})
	if err != nil {
		return nil, grpcerror.Internal("encode payload", err)
	}
	if err := h.executeCharacterCommand(ctx, CharacterCommandInput{
		CampaignID:      campaignID,
		CharacterID:     characterID,
		CommandType:     commandids.DaggerheartConsumableUse,
		SessionID:       strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx)),
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "consumable use did not emit an event",
		ApplyErrMessage: "apply consumable use event",
	}); err != nil {
		return nil, err
	}

	return &pb.DaggerheartUseConsumableResponse{CharacterId: characterID}, nil
}

// AcquireConsumable records a consumable gain for one character through the
// shared character-command seam.
func (h *Handler) AcquireConsumable(ctx context.Context, in *pb.DaggerheartAcquireConsumableRequest) (*pb.DaggerheartAcquireConsumableResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "acquire consumable request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}
	consumableID, err := validate.RequiredID(in.GetConsumableId(), "consumable id")
	if err != nil {
		return nil, err
	}
	if _, err := h.validateCharacterPreconditions(ctx, campaignID, characterID, "consumable acquire"); err != nil {
		return nil, err
	}

	payloadJSON, err := json.Marshal(daggerheartpayload.ConsumableAcquirePayload{
		CharacterID:    ids.CharacterID(characterID),
		ConsumableID:   consumableID,
		QuantityBefore: int(in.GetQuantityBefore()),
		QuantityAfter:  int(in.GetQuantityAfter()),
	})
	if err != nil {
		return nil, grpcerror.Internal("encode payload", err)
	}
	if err := h.executeCharacterCommand(ctx, CharacterCommandInput{
		CampaignID:      campaignID,
		CharacterID:     characterID,
		CommandType:     commandids.DaggerheartConsumableAcquire,
		SessionID:       strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx)),
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "consumable acquire did not emit an event",
		ApplyErrMessage: "apply consumable acquire event",
	}); err != nil {
		return nil, err
	}

	return &pb.DaggerheartAcquireConsumableResponse{CharacterId: characterID}, nil
}
