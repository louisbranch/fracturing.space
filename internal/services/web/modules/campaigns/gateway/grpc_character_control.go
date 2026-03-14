package gateway

import (
	"context"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// SetCharacterController applies an explicit controller update via gRPC.
func (g characterControlMutationGateway) SetCharacterController(ctx context.Context, campaignID string, characterID string, participantID string) error {
	if g.mutation.Character == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}

	_, err := g.mutation.Character.SetDefaultControl(ctx, &statev1.SetDefaultControlRequest{
		CampaignId:    campaignID,
		CharacterId:   characterID,
		ParticipantId: wrapperspb.String(strings.TrimSpace(participantID)),
	})
	if err != nil {
		return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_set_character_controller",
			FallbackMessage: "failed to set character controller",
		})
	}
	return nil
}

// ClaimCharacterControl claims character control via gRPC.
func (g characterControlMutationGateway) ClaimCharacterControl(ctx context.Context, campaignID string, characterID string) error {
	if g.mutation.Character == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}

	_, err := g.mutation.Character.ClaimCharacterControl(ctx, &statev1.ClaimCharacterControlRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_claim_character_control",
			FallbackMessage: "failed to claim character control",
		})
	}
	return nil
}

// ReleaseCharacterControl releases character control via gRPC.
func (g characterControlMutationGateway) ReleaseCharacterControl(ctx context.Context, campaignID string, characterID string) error {
	if g.mutation.Character == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}

	_, err := g.mutation.Character.ReleaseCharacterControl(ctx, &statev1.ReleaseCharacterControlRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_release_character_control",
			FallbackMessage: "failed to release character control",
		})
	}
	return nil
}
