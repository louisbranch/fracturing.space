package gateway

import (
	"context"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// SetCharacterOwner applies an explicit owner update via gRPC.
func (g characterOwnershipMutationGateway) SetCharacterOwner(ctx context.Context, campaignID string, characterID string, participantID string) error {
	if g.mutation.Character == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}

	_, err := g.mutation.Character.UpdateCharacter(ctx, &statev1.UpdateCharacterRequest{
		CampaignId:         campaignID,
		CharacterId:        characterID,
		OwnerParticipantId: wrapperspb.String(strings.TrimSpace(participantID)),
	})
	if err != nil {
		return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_set_character_owner",
			FallbackMessage: "failed to set character owner",
		})
	}
	return nil
}
