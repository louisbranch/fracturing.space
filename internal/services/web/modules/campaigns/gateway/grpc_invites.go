package gateway

import (
	"context"
	"strings"
	"sync"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/grpcpaging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const inviteRecipientLookupMaxConcurrency = 4

// CampaignInvites centralizes this web behavior in one helper seam.
func (g inviteReadGateway) CampaignInvites(ctx context.Context, campaignID string) ([]campaignapp.CampaignInvite, error) {
	if g.read.Invite == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.invite_service_client_is_not_configured", "invite service client is not configured")
	}
	if g.read.Participant == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.participant_service_client_is_not_configured", "participant service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return []campaignapp.CampaignInvite{}, nil
	}
	participants, err := g.read.Participant.ListParticipants(ctx, &statev1.ListParticipantsRequest{
		CampaignId: campaignID,
		PageSize:   100,
	})
	if err != nil {
		return nil, err
	}
	participantNames := make(map[string]string, len(participants.GetParticipants()))
	for _, participant := range participants.GetParticipants() {
		if participant == nil {
			continue
		}
		participantNames[strings.TrimSpace(participant.GetId())] = strings.TrimSpace(participant.GetName())
	}

	invites, err := grpcpaging.CollectPages[*invitev1.Invite, *invitev1.Invite](
		ctx, 10,
		func(ctx context.Context, pageToken string) ([]*invitev1.Invite, string, error) {
			resp, err := g.read.Invite.ListInvites(ctx, &invitev1.ListInvitesRequest{
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
			return resp.GetInvites(), resp.GetNextPageToken(), nil
		},
		func(invite *invitev1.Invite) (*invitev1.Invite, bool) {
			return invite, invite != nil
		},
	)
	if err != nil {
		return nil, err
	}

	recipientUsernames := g.resolveInviteRecipientUsernames(ctx, invites)
	result := make([]campaignapp.CampaignInvite, 0, len(invites))
	for _, invite := range invites {
		recipientUserID := strings.TrimSpace(invite.GetRecipientUserId())
		result = append(result, campaignapp.CampaignInvite{
			ID:                strings.TrimSpace(invite.GetId()),
			ParticipantID:     strings.TrimSpace(invite.GetParticipantId()),
			ParticipantName:   participantNames[strings.TrimSpace(invite.GetParticipantId())],
			RecipientUserID:   recipientUserID,
			RecipientUsername: recipientUsernames[recipientUserID],
			HasRecipient:      recipientUserID != "",
			Status:            inviteStatusLabel(invite.GetStatus()),
		})
	}
	return result, nil
}

// inviteStatusLabel translates invite status enums to display labels.
func inviteStatusLabel(s invitev1.InviteStatus) string {
	switch s {
	case invitev1.InviteStatus_PENDING:
		return "Pending"
	case invitev1.InviteStatus_CLAIMED:
		return "Claimed"
	case invitev1.InviteStatus_REVOKED:
		return "Revoked"
	case invitev1.InviteStatus_DECLINED:
		return "Declined"
	default:
		return "Unspecified"
	}
}

// CreateInvite executes package-scoped creation behavior for this flow.
func (g inviteMutationGateway) CreateInvite(ctx context.Context, campaignID string, input campaignapp.CreateInviteInput) error {
	if g.mutation.Invite == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.invite_service_client_is_not_configured", "invite service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	participantID := strings.TrimSpace(input.ParticipantID)
	if participantID == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.participant_id_is_required", "participant id is required")
	}
	recipientUserID, err := g.resolveInviteRecipientUserID(ctx, input.RecipientUsername)
	if err != nil {
		return err
	}

	_, err = g.mutation.Invite.CreateInvite(ctx, &invitev1.CreateInviteRequest{
		CampaignId:      campaignID,
		ParticipantId:   participantID,
		RecipientUserId: recipientUserID,
	})
	if err != nil {
		return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_create_invite",
			FallbackMessage: "failed to create invite",
		})
	}
	return nil
}

// resolveInviteRecipientUserID canonicalizes optional invite usernames into auth user IDs.
func (g inviteMutationGateway) resolveInviteRecipientUserID(ctx context.Context, username string) (string, error) {
	username = strings.TrimSpace(username)
	username = strings.TrimPrefix(username, "@")
	username = strings.TrimSpace(username)
	if username == "" {
		return "", nil
	}
	if g.mutation.Auth == nil {
		return "", apperrors.EK(apperrors.KindUnavailable, "error.web.message.auth_service_is_not_configured", "auth service client is not configured")
	}

	resp, err := g.mutation.Auth.LookupUserByUsername(ctx, &authv1.LookupUserByUsernameRequest{Username: username})
	if err != nil {
		if statusErr, ok := status.FromError(err); ok {
			switch statusErr.Code() {
			case codes.InvalidArgument:
				return "", apperrors.EK(apperrors.KindInvalidInput, "error.web.message.recipient_username_is_invalid", "recipient username is invalid")
			case codes.NotFound:
				return "", apperrors.EK(apperrors.KindInvalidInput, "error.web.message.recipient_username_was_not_found", "recipient username was not found")
			}
		}
		return "", apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnavailable,
			FallbackKey:     "error.web.message.failed_to_create_invite",
			FallbackMessage: "failed to create invite",
		})
	}
	if resp == nil || resp.GetUser() == nil || strings.TrimSpace(resp.GetUser().GetId()) == "" {
		return "", apperrors.EK(apperrors.KindInvalidInput, "error.web.message.recipient_username_was_not_found", "recipient username was not found")
	}
	return strings.TrimSpace(resp.GetUser().GetId()), nil
}

// resolveInviteRecipientUsernames batches best-effort auth lookups so invite pages
// do not serialize one remote call per rendered row.
func (g inviteReadGateway) resolveInviteRecipientUsernames(ctx context.Context, invites []*invitev1.Invite) map[string]string {
	result := map[string]string{}
	if g.read.Auth == nil || len(invites) == 0 {
		return result
	}

	uniqueRecipientUserIDs := make([]string, 0, len(invites))
	seen := make(map[string]struct{}, len(invites))
	for _, invite := range invites {
		if invite == nil {
			continue
		}
		recipientUserID := strings.TrimSpace(invite.GetRecipientUserId())
		if recipientUserID == "" {
			continue
		}
		if _, ok := seen[recipientUserID]; ok {
			continue
		}
		seen[recipientUserID] = struct{}{}
		uniqueRecipientUserIDs = append(uniqueRecipientUserIDs, recipientUserID)
	}
	if len(uniqueRecipientUserIDs) == 0 {
		return result
	}

	sem := make(chan struct{}, inviteRecipientLookupMaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, recipientUserID := range uniqueRecipientUserIDs {
		wg.Add(1)
		go func(recipientUserID string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			userResp, err := g.read.Auth.GetUser(ctx, &authv1.GetUserRequest{UserId: recipientUserID})
			if err != nil || userResp == nil || userResp.GetUser() == nil {
				return
			}
			username := strings.TrimSpace(userResp.GetUser().GetUsername())
			if username == "" {
				return
			}

			mu.Lock()
			result[recipientUserID] = username
			mu.Unlock()
		}(recipientUserID)
	}
	wg.Wait()
	return result
}

// RevokeInvite applies this package workflow transition.
func (g inviteMutationGateway) RevokeInvite(ctx context.Context, campaignID string, input campaignapp.RevokeInviteInput) error {
	if g.mutation.Invite == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.invite_service_client_is_not_configured", "invite service client is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	inviteID := strings.TrimSpace(input.InviteID)
	if inviteID == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.invite_id_is_required", "invite id is required")
	}

	_, err := g.mutation.Invite.RevokeInvite(ctx, &invitev1.RevokeInviteRequest{InviteId: inviteID})
	if err != nil {
		return apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_revoke_invite",
			FallbackMessage: "failed to revoke invite",
		})
	}
	return nil
}
