package invite

import (
	"context"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	"github.com/louisbranch/fracturing.space/internal/services/invite/storage"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func inviteToProto(rec storage.InviteRecord) *invitev1.Invite {
	return &invitev1.Invite{
		Id:                     rec.ID,
		CampaignId:             rec.CampaignID,
		ParticipantId:          rec.ParticipantID,
		RecipientUserId:        rec.RecipientUserID,
		Status:                 statusToProto(rec.Status),
		CreatedByParticipantId: rec.CreatedByParticipantID,
		CreatedAt:              timestamppb.New(rec.CreatedAt),
		UpdatedAt:              timestamppb.New(rec.UpdatedAt),
	}
}

func invitesToProto(recs []storage.InviteRecord) []*invitev1.Invite {
	out := make([]*invitev1.Invite, len(recs))
	for i, r := range recs {
		out[i] = inviteToProto(r)
	}
	return out
}

func statusToProto(s storage.Status) invitev1.InviteStatus {
	switch s {
	case storage.StatusPending:
		return invitev1.InviteStatus_PENDING
	case storage.StatusClaimed:
		return invitev1.InviteStatus_CLAIMED
	case storage.StatusRevoked:
		return invitev1.InviteStatus_REVOKED
	case storage.StatusDeclined:
		return invitev1.InviteStatus_DECLINED
	default:
		return invitev1.InviteStatus_INVITE_STATUS_UNSPECIFIED
	}
}

func statusFromProto(s invitev1.InviteStatus) storage.Status {
	switch s {
	case invitev1.InviteStatus_PENDING:
		return storage.StatusPending
	case invitev1.InviteStatus_CLAIMED:
		return storage.StatusClaimed
	case invitev1.InviteStatus_REVOKED:
		return storage.StatusRevoked
	case invitev1.InviteStatus_DECLINED:
		return storage.StatusDeclined
	default:
		return ""
	}
}

// loadCreatorUser resolves the creator participant to a user summary by first
// looking up the participant's user_id, then loading the auth user record.
func loadCreatorUser(ctx context.Context, game gamev1.ParticipantServiceClient, auth authv1.AuthServiceClient, campaignID, participantID string) *invitev1.InviteUserSummary {
	if game == nil || auth == nil {
		return nil
	}
	pResp, err := game.GetParticipant(ctx, &gamev1.GetParticipantRequest{
		CampaignId:    campaignID,
		ParticipantId: participantID,
	})
	if err != nil || pResp.GetParticipant() == nil {
		return nil
	}
	userID := strings.TrimSpace(pResp.GetParticipant().GetUserId())
	if userID == "" {
		return nil
	}
	uResp, err := auth.GetUser(ctx, &authv1.GetUserRequest{UserId: userID})
	if err != nil || uResp.GetUser() == nil {
		return nil
	}
	return &invitev1.InviteUserSummary{
		Id:       uResp.GetUser().GetId(),
		Username: uResp.GetUser().GetUsername(),
	}
}
