package gateway

import (
	"context"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/grpcpaging"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const campaignAvatarCardDeliveryWidthPX = 384

// CampaignParticipants centralizes this web behavior in one helper seam.
func (g participantReadGateway) CampaignParticipants(ctx context.Context, campaignID string) ([]campaignapp.CampaignParticipant, error) {
	if g.read.Participant == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.participant_service_client_is_not_configured", "participant service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []campaignapp.CampaignParticipant{}, nil
	}

	return grpcpaging.CollectPages[campaignapp.CampaignParticipant, *statev1.Participant](
		ctx, 10,
		func(ctx context.Context, pageToken string) ([]*statev1.Participant, string, error) {
			resp, err := g.read.Participant.ListParticipants(ctx, &statev1.ListParticipantsRequest{
				CampaignId: campaignID,
				PageSize:   10,
				PageToken:  pageToken,
			})
			if err != nil {
				return nil, "", err
			}
			if resp == nil {
				return nil, "", nil
			}
			return resp.GetParticipants(), resp.GetNextPageToken(), nil
		},
		func(participant *statev1.Participant) (campaignapp.CampaignParticipant, bool) {
			if participant == nil {
				return campaignapp.CampaignParticipant{}, false
			}
			participantID := strings.TrimSpace(participant.GetId())
			avatarEntityID := participantID
			if avatarEntityID == "" {
				avatarEntityID = strings.TrimSpace(participant.GetUserId())
			}
			if avatarEntityID == "" {
				avatarEntityID = campaignID
			}
			return campaignapp.CampaignParticipant{
				ID:             participantID,
				UserID:         strings.TrimSpace(participant.GetUserId()),
				Name:           participantDisplayName(participant),
				Role:           participantRoleLabel(participant.GetRole()),
				CampaignAccess: participantCampaignAccessLabel(participant.GetCampaignAccess()),
				Controller:     participantControllerLabel(participant.GetController()),
				Pronouns:       pronouns.FromProto(participant.GetPronouns()),
				AvatarURL: websupport.AvatarImageURL(
					g.assetBaseURL,
					catalog.AvatarRoleParticipant,
					avatarEntityID,
					strings.TrimSpace(participant.GetAvatarSetId()),
					strings.TrimSpace(participant.GetAvatarAssetId()),
					campaignAvatarCardDeliveryWidthPX,
				),
			}, true
		},
	)
}

// CampaignParticipant centralizes this web behavior in one helper seam.
func (g participantReadGateway) CampaignParticipant(ctx context.Context, campaignID string, participantID string) (campaignapp.CampaignParticipant, error) {
	if g.read.Participant == nil {
		return campaignapp.CampaignParticipant{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.participant_service_client_is_not_configured", "participant service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return campaignapp.CampaignParticipant{}, apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	participantID = strings.TrimSpace(participantID)
	if participantID == "" {
		return campaignapp.CampaignParticipant{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.participant_id_is_required", "participant id is required")
	}

	resp, err := g.read.Participant.GetParticipant(ctx, &statev1.GetParticipantRequest{
		CampaignId:    campaignID,
		ParticipantId: participantID,
	})
	if err != nil {
		return campaignapp.CampaignParticipant{}, apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_load_participant",
			FallbackMessage: "failed to load participant",
		})
	}
	participant := resp.GetParticipant()
	if participant == nil {
		return campaignapp.CampaignParticipant{}, apperrors.E(apperrors.KindNotFound, "participant not found")
	}

	avatarEntityID := strings.TrimSpace(participant.GetId())
	if avatarEntityID == "" {
		avatarEntityID = strings.TrimSpace(participant.GetUserId())
	}
	if avatarEntityID == "" {
		avatarEntityID = campaignID
	}
	return campaignapp.CampaignParticipant{
		ID:             strings.TrimSpace(participant.GetId()),
		UserID:         strings.TrimSpace(participant.GetUserId()),
		Name:           participantDisplayName(participant),
		Role:           participantRoleLabel(participant.GetRole()),
		CampaignAccess: participantCampaignAccessLabel(participant.GetCampaignAccess()),
		Controller:     participantControllerLabel(participant.GetController()),
		Pronouns:       pronouns.FromProto(participant.GetPronouns()),
		AvatarURL: websupport.AvatarImageURL(
			g.assetBaseURL,
			catalog.AvatarRoleParticipant,
			avatarEntityID,
			strings.TrimSpace(participant.GetAvatarSetId()),
			strings.TrimSpace(participant.GetAvatarAssetId()),
			campaignAvatarCardDeliveryWidthPX,
		),
	}, nil
}

// CreateParticipant executes package-scoped creation behavior for this flow.
func (g participantMutationGateway) CreateParticipant(ctx context.Context, campaignID string, input campaignapp.CreateParticipantInput) (campaignapp.CreateParticipantResult, error) {
	if g.mutation.Participant == nil {
		return campaignapp.CreateParticipantResult{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.participant_service_client_is_not_configured", "participant service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return campaignapp.CreateParticipantResult{}, apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return campaignapp.CreateParticipantResult{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.participant_name_is_required", "participant name is required")
	}
	role := mapParticipantRoleToProto(input.Role)
	if role == statev1.ParticipantRole_ROLE_UNSPECIFIED {
		return campaignapp.CreateParticipantResult{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.participant_role_value_is_invalid", "participant role value is invalid")
	}
	access := mapParticipantAccessToProto(input.CampaignAccess)
	if access == statev1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED {
		return campaignapp.CreateParticipantResult{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.campaign_access_value_is_invalid", "campaign access value is invalid")
	}

	resp, err := g.mutation.Participant.CreateParticipant(ctx, &statev1.CreateParticipantRequest{
		CampaignId:     campaignID,
		Name:           name,
		Role:           role,
		Controller:     statev1.Controller_CONTROLLER_HUMAN,
		CampaignAccess: access,
	})
	if err != nil {
		return campaignapp.CreateParticipantResult{}, apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_create_participant",
			FallbackMessage: "failed to create participant",
		})
	}
	if resp.GetParticipant() == nil {
		return campaignapp.CreateParticipantResult{}, apperrors.EK(apperrors.KindUnknown, "error.web.message.created_participant_id_was_empty", "created participant id was empty")
	}
	return campaignapp.CreateParticipantResult{ParticipantID: strings.TrimSpace(resp.GetParticipant().GetId())}, nil
}

// UpdateParticipant applies this package workflow transition.
func (g participantMutationGateway) UpdateParticipant(ctx context.Context, campaignID string, input campaignapp.UpdateParticipantInput) error {
	if g.mutation.Participant == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.participant_service_client_is_not_configured", "participant service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	participantID := strings.TrimSpace(input.ParticipantID)
	if participantID == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.participant_id_is_required", "participant id is required")
	}
	role := mapParticipantRoleToProto(input.Role)
	if role == statev1.ParticipantRole_ROLE_UNSPECIFIED {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.participant_role_value_is_invalid", "participant role value is invalid")
	}
	access := mapParticipantAccessToProto(input.CampaignAccess)
	if strings.TrimSpace(input.CampaignAccess) != "" && access == statev1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.campaign_access_value_is_invalid", "campaign access value is invalid")
	}

	request := &statev1.UpdateParticipantRequest{
		CampaignId:    campaignID,
		ParticipantId: participantID,
		Name:          wrapperspb.String(strings.TrimSpace(input.Name)),
		Role:          role,
		Pronouns:      pronouns.ToProto(strings.TrimSpace(input.Pronouns)),
	}
	if access != statev1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED {
		request.CampaignAccess = access
	}

	_, err := g.mutation.Participant.UpdateParticipant(ctx, request)
	if err != nil {
		return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_update_participant",
			FallbackMessage: "failed to update participant",
		})
	}
	return nil
}

// DeleteParticipant applies this package workflow transition.
func (g participantMutationGateway) DeleteParticipant(ctx context.Context, campaignID string, participantID string) error {
	if g.mutation.Participant == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.participant_service_client_is_not_configured", "participant service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	participantID = strings.TrimSpace(participantID)
	if participantID == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.participant_id_is_required", "participant id is required")
	}

	_, err := g.mutation.Participant.DeleteParticipant(ctx, &statev1.DeleteParticipantRequest{
		CampaignId:    campaignID,
		ParticipantId: participantID,
	})
	if err != nil {
		return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_delete_participant",
			FallbackMessage: "failed to delete participant",
		})
	}
	return nil
}
