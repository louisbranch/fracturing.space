package invitetransport

import (
	"context"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/campaigntransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/participanttransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListPendingInvites returns a page of pending invites for a campaign.
func (s *Service) ListPendingInvites(ctx context.Context, in *campaignv1.ListPendingInvitesRequest) (*campaignv1.ListPendingInvitesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list pending invites request is required")
	}
	campaignID, err := validate.RequiredID(in.GetCampaignId(), "campaign id")
	if err != nil {
		return nil, err
	}
	campaignRecord, err := s.reads.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if err := authz.RequirePolicy(ctx, s.reads.auth, domainauthz.CapabilityReadInvites, campaignRecord); err != nil {
		return nil, err
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListInvitesPageSize,
		Max:     maxListInvitesPageSize,
	})

	page, err := s.reads.stores.Invite.ListPendingInvites(ctx, campaignID, pageSize, in.GetPageToken())
	if err != nil {
		return nil, grpcerror.Internal("list pending invites", err)
	}

	response := &campaignv1.ListPendingInvitesResponse{NextPageToken: page.NextPageToken}
	if len(page.Invites) == 0 {
		return response, nil
	}

	participants, err := s.reads.stores.Participant.ListParticipantsByCampaign(ctx, campaignID)
	if err != nil {
		return nil, grpcerror.Internal("list participants", err)
	}
	participantsByID := make(map[string]storage.ParticipantRecord, len(participants))
	for _, p := range participants {
		participantsByID[p.ID] = p
	}

	userCache := make(map[string]*authv1.User)
	response.Invites = make([]*campaignv1.PendingInvite, 0, len(page.Invites))
	for _, inv := range page.Invites {
		seat, ok := participantsByID[inv.ParticipantID]
		if !ok {
			return nil, status.Errorf(codes.Internal, "participant seat not found: %s", inv.ParticipantID)
		}
		var createdByUser *authv1.User
		creatorID := strings.TrimSpace(inv.CreatedByParticipantID)
		if creatorID != "" {
			creator, ok := participantsByID[creatorID]
			if !ok {
				return nil, status.Errorf(codes.Internal, "creator participant not found: %s", creatorID)
			}
			creatorUserID := strings.TrimSpace(creator.UserID)
			if creatorUserID != "" {
				if s.reads.authClient == nil {
					return nil, status.Error(codes.Internal, "auth client is not configured")
				}
				cached, ok := userCache[creatorUserID]
				if !ok {
					userResponse, err := s.reads.authClient.GetUser(ctx, &authv1.GetUserRequest{UserId: creatorUserID})
					if err != nil {
						return nil, grpcerror.Internal("get auth user", err)
					}
					if userResponse == nil || userResponse.GetUser() == nil {
						return nil, status.Error(codes.Internal, "auth user response is missing")
					}
					cached = userResponse.GetUser()
					userCache[creatorUserID] = cached
				}
				createdByUser = cached
			}
		}

		response.Invites = append(response.Invites, &campaignv1.PendingInvite{
			Invite:        InviteToProto(inv),
			Participant:   participanttransport.ParticipantToProto(seat),
			CreatedByUser: createdByUser,
		})
	}

	return response, nil
}

// ListPendingInvitesForUser returns a page of pending invites for the current user.
func (s *Service) ListPendingInvitesForUser(ctx context.Context, in *campaignv1.ListPendingInvitesForUserRequest) (*campaignv1.ListPendingInvitesForUserResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list pending invites for user request is required")
	}
	userID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListInvitesPageSize,
		Max:     maxListInvitesPageSize,
	})

	page, err := s.reads.stores.Invite.ListPendingInvitesForRecipient(ctx, userID, pageSize, in.GetPageToken())
	if err != nil {
		return nil, grpcerror.Internal("list pending invites for user", err)
	}

	response := &campaignv1.ListPendingInvitesForUserResponse{NextPageToken: page.NextPageToken}
	if len(page.Invites) == 0 {
		return response, nil
	}

	campaignsByID := make(map[string]storage.CampaignRecord)
	participantsByID := make(map[string]storage.ParticipantRecord)
	response.Invites = make([]*campaignv1.PendingUserInvite, 0, len(page.Invites))
	for _, inv := range page.Invites {
		campaignRecord, ok := campaignsByID[inv.CampaignID]
		if !ok {
			record, err := s.reads.stores.Campaign.Get(ctx, inv.CampaignID)
			if err != nil {
				return nil, err
			}
			campaignRecord = record
			campaignsByID[inv.CampaignID] = campaignRecord
		}

		participantKey := inv.CampaignID + ":" + inv.ParticipantID
		seat, ok := participantsByID[participantKey]
		if !ok {
			record, err := s.reads.stores.Participant.GetParticipant(ctx, inv.CampaignID, inv.ParticipantID)
			if err != nil {
				return nil, err
			}
			seat = record
			participantsByID[participantKey] = seat
		}

		response.Invites = append(response.Invites, &campaignv1.PendingUserInvite{
			Invite:      InviteToProto(inv),
			Campaign:    campaigntransport.CampaignToProto(campaignRecord),
			Participant: participanttransport.ParticipantToProto(seat),
		})
	}

	return response, nil
}
