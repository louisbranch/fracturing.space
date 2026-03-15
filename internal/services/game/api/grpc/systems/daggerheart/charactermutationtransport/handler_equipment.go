package charactermutationtransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SwapEquipment applies an equipment move through the shared character-command
// execution path.
func (h *Handler) SwapEquipment(ctx context.Context, in *pb.DaggerheartSwapEquipmentRequest) (*pb.DaggerheartSwapEquipmentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "swap equipment request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}
	itemID, err := validate.RequiredID(in.GetItemId(), "item id")
	if err != nil {
		return nil, err
	}
	if _, err := h.validateCharacterPreconditions(ctx, campaignID, characterID, "equipment swap"); err != nil {
		return nil, err
	}

	payloadJSON, err := json.Marshal(daggerheart.EquipmentSwapPayload{
		CharacterID: ids.CharacterID(characterID),
		ItemID:      itemID,
		ItemType:    strings.TrimSpace(in.GetItemType()),
		From:        strings.TrimSpace(in.GetFrom()),
		To:          strings.TrimSpace(in.GetTo()),
		StressCost:  int(in.GetStressCost()),
	})
	if err != nil {
		return nil, grpcerror.Internal("encode payload", err)
	}
	if err := h.executeCharacterCommand(ctx, CharacterCommandInput{
		CampaignID:      campaignID,
		CharacterID:     characterID,
		CommandType:     commandids.DaggerheartEquipmentSwap,
		SessionID:       strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx)),
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "equipment swap did not emit an event",
		ApplyErrMessage: "apply equipment swap event",
	}); err != nil {
		return nil, err
	}

	return &pb.DaggerheartSwapEquipmentResponse{CharacterId: characterID}, nil
}
