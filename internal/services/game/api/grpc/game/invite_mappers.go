package game

import (
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Invite proto conversion helpers.
func inviteToProto(inv storage.InviteRecord) *campaignv1.Invite {
	return &campaignv1.Invite{
		Id:                     inv.ID,
		CampaignId:             inv.CampaignID,
		ParticipantId:          inv.ParticipantID,
		RecipientUserId:        inv.RecipientUserID,
		Status:                 inviteStatusToProto(inv.Status),
		CreatedByParticipantId: inv.CreatedByParticipantID,
		CreatedAt:              timestamppb.New(inv.CreatedAt),
		UpdatedAt:              timestamppb.New(inv.UpdatedAt),
	}
}

func inviteStatusToProto(status invite.Status) campaignv1.InviteStatus {
	switch status {
	case invite.StatusPending:
		return campaignv1.InviteStatus_PENDING
	case invite.StatusClaimed:
		return campaignv1.InviteStatus_CLAIMED
	case invite.StatusRevoked:
		return campaignv1.InviteStatus_REVOKED
	default:
		return campaignv1.InviteStatus_INVITE_STATUS_UNSPECIFIED
	}
}

func inviteStatusFromProto(status campaignv1.InviteStatus) invite.Status {
	switch status {
	case campaignv1.InviteStatus_PENDING:
		return invite.StatusPending
	case campaignv1.InviteStatus_CLAIMED:
		return invite.StatusClaimed
	case campaignv1.InviteStatus_REVOKED:
		return invite.StatusRevoked
	default:
		return invite.StatusUnspecified
	}
}
