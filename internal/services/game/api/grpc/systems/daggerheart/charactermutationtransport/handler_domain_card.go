package charactermutationtransport

import (
	"context"
	"encoding/json"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AcquireDomainCard routes a domain card grant through the character command
// seam and preserves the package's default destination behavior.
func (h *Handler) AcquireDomainCard(ctx context.Context, in *pb.DaggerheartAcquireDomainCardRequest) (*pb.DaggerheartAcquireDomainCardResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "acquire domain card request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, err
	}
	cardID, err := validate.RequiredID(in.GetCardId(), "card id")
	if err != nil {
		return nil, err
	}
	if _, err := h.validateCharacterPreconditions(ctx, campaignID, characterID, "domain card acquire"); err != nil {
		return nil, err
	}

	payload := daggerheartpayload.DomainCardAcquirePayload{
		CharacterID: ids.CharacterID(characterID),
		CardID:      cardID,
		CardLevel:   int(in.GetCardLevel()),
		Destination: strings.TrimSpace(in.GetDestination()),
	}
	if payload.Destination == "" {
		payload.Destination = "vault"
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, grpcerror.Internal("encode payload", err)
	}
	if err := h.executeCharacterCommand(ctx, CharacterCommandInput{
		CampaignID:      campaignID,
		CharacterID:     characterID,
		CommandType:     commandids.DaggerheartDomainCardAcquire,
		SessionID:       strings.TrimSpace(grpcmeta.SessionIDFromContext(ctx)),
		RequestID:       grpcmeta.RequestIDFromContext(ctx),
		InvocationID:    grpcmeta.InvocationIDFromContext(ctx),
		PayloadJSON:     payloadJSON,
		MissingEventMsg: "domain card acquire did not emit an event",
		ApplyErrMessage: "apply domain card acquire event",
	}); err != nil {
		return nil, err
	}

	return &pb.DaggerheartAcquireDomainCardResponse{CharacterId: characterID}, nil
}
