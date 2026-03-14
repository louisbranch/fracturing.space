package game

import (
	"context"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/campaigntransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/participanttransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetInvite returns an invite by ID.
func (s *InviteService) GetInvite(ctx context.Context, in *campaignv1.GetInviteRequest) (*campaignv1.GetInviteResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get invite request is required")
	}
	inviteID, err := validate.RequiredID(in.GetInviteId(), "invite id")
	if err != nil {
		return nil, err
	}

	inv, err := s.stores.Invite.GetInvite(ctx, inviteID)
	if err != nil {
		return nil, err
	}
	campaignRecord, err := s.stores.Campaign.Get(ctx, inv.CampaignID)
	if err != nil {
		return nil, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, err
	}
	if err := requirePolicy(ctx, s.stores, domainauthz.CapabilityReadInvites, campaignRecord); err != nil {
		return nil, err
	}

	return &campaignv1.GetInviteResponse{Invite: inviteToProto(inv)}, nil
}

// GetPublicInvite returns the public invite landing data by invite ID.
func (s *InviteService) GetPublicInvite(ctx context.Context, in *campaignv1.GetPublicInviteRequest) (*campaignv1.GetPublicInviteResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get public invite request is required")
	}
	inviteID, err := validate.RequiredID(in.GetInviteId(), "invite id")
	if err != nil {
		return nil, err
	}

	inv, err := s.stores.Invite.GetInvite(ctx, inviteID)
	if err != nil {
		return nil, err
	}
	campaignRecord, err := s.stores.Campaign.Get(ctx, inv.CampaignID)
	if err != nil {
		return nil, err
	}
	seat, err := s.stores.Participant.GetParticipant(ctx, inv.CampaignID, inv.ParticipantID)
	if err != nil {
		return nil, err
	}

	var createdByUser *authv1.User
	if usernameUser, ok := s.publicInviteCreatorUser(ctx, inv); ok {
		createdByUser = usernameUser
	}

	return &campaignv1.GetPublicInviteResponse{
		Invite: inviteToProto(inv),
		Campaign: &campaignv1.PublicInviteCampaign{
			Id:     campaignRecord.ID,
			Name:   campaignRecord.Name,
			Status: campaigntransport.CampaignStatusToProto(campaignRecord.Status),
		},
		Participant:   participanttransport.ParticipantToProto(seat),
		CreatedByUser: createdByUser,
	}, nil
}

// ListInvites returns a page of invites for a campaign.
func (s *InviteService) ListInvites(ctx context.Context, in *campaignv1.ListInvitesRequest) (*campaignv1.ListInvitesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list invites request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpRead); err != nil {
		return nil, err
	}
	if err := requirePolicy(ctx, s.stores, domainauthz.CapabilityReadInvites, campaignRecord); err != nil {
		return nil, err
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListInvitesPageSize,
		Max:     maxListInvitesPageSize,
	})

	statusFilter := invite.StatusUnspecified
	if in.GetStatus() != campaignv1.InviteStatus_INVITE_STATUS_UNSPECIFIED {
		statusFilter = inviteStatusFromProto(in.GetStatus())
	}

	page, err := s.stores.Invite.ListInvites(ctx, campaignID, strings.TrimSpace(in.GetRecipientUserId()), statusFilter, pageSize, in.GetPageToken())
	if err != nil {
		return nil, grpcerror.Internal("list invites", err)
	}

	response := &campaignv1.ListInvitesResponse{NextPageToken: page.NextPageToken}
	if len(page.Invites) == 0 {
		return response, nil
	}

	response.Invites = make([]*campaignv1.Invite, 0, len(page.Invites))
	for _, inv := range page.Invites {
		response.Invites = append(response.Invites, inviteToProto(inv))
	}

	return response, nil
}

func (s *InviteService) publicInviteCreatorUser(ctx context.Context, inv storage.InviteRecord) (*authv1.User, bool) {
	if s.authClient == nil {
		return nil, false
	}
	creatorParticipantID := strings.TrimSpace(inv.CreatedByParticipantID)
	if creatorParticipantID == "" {
		return nil, false
	}
	creatorParticipant, err := s.stores.Participant.GetParticipant(ctx, inv.CampaignID, creatorParticipantID)
	if err != nil {
		return nil, false
	}
	creatorUserID := strings.TrimSpace(creatorParticipant.UserID)
	if creatorUserID == "" {
		return nil, false
	}
	userResponse, err := s.authClient.GetUser(ctx, &authv1.GetUserRequest{UserId: creatorUserID})
	if err != nil || userResponse == nil || userResponse.GetUser() == nil {
		return nil, false
	}
	return userResponse.GetUser(), true
}
