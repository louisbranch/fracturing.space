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

// UpdateGold applies a character gold mutation and reloads the profile so the
// transport response reflects the durable post-write state.
func (h *Handler) UpdateGold(ctx context.Context, in *pb.DaggerheartUpdateGoldRequest) (*pb.DaggerheartUpdateGoldResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update gold request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}
	if _, err := h.validateCharacterPreconditions(ctx, campaignID, characterID, "gold update"); err != nil {
		return nil, err
	}

	payloadJSON, err := json.Marshal(daggerheart.GoldUpdatePayload{
		CharacterID:    ids.CharacterID(characterID),
		HandfulsBefore: int(in.GetHandfulsBefore()),
		HandfulsAfter:  int(in.GetHandfulsAfter()),
		BagsBefore:     int(in.GetBagsBefore()),
		BagsAfter:      int(in.GetBagsAfter()),
		ChestsBefore:   int(in.GetChestsBefore()),
		ChestsAfter:    int(in.GetChestsAfter()),
		Reason:         strings.TrimSpace(in.GetReason()),
	})
	if err != nil {
		return nil, grpcerror.Internal("encode payload", err)
	}
	if err := h.executeCharacterCommand(ctx, CharacterCommandInput{
		CampaignID:      campaignID,
		CharacterID:     characterID,
		CommandType:     commandids.DaggerheartGoldUpdate,
		SessionID:       strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx)),
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "gold update did not emit an event",
		ApplyErrMessage: "apply gold update event",
	}); err != nil {
		return nil, err
	}

	updatedProfile, err := h.deps.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return nil, grpcerror.Internal("load daggerheart profile", err)
	}
	return &pb.DaggerheartUpdateGoldResponse{
		CharacterId: characterID,
		Handfuls:    int32(updatedProfile.GoldHandfuls),
		Bags:        int32(updatedProfile.GoldBags),
		Chests:      int32(updatedProfile.GoldChests),
	}, nil
}
