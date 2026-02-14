package game

import (
	"context"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/policy"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultListInvitesPageSize = 10
	maxListInvitesPageSize     = 10
)

// InviteService implements the game.v1.InviteService gRPC API.
type InviteService struct {
	campaignv1.UnimplementedInviteServiceServer
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
	authClient  authv1.AuthServiceClient
}

// NewInviteService creates an InviteService with default dependencies.
func NewInviteService(stores Stores) *InviteService {
	return &InviteService{
		stores:      stores,
		clock:       time.Now,
		idGenerator: id.NewID,
	}
}

// NewInviteServiceWithAuth creates an InviteService with an auth client.
func NewInviteServiceWithAuth(stores Stores, authClient authv1.AuthServiceClient) *InviteService {
	service := NewInviteService(stores)
	service.authClient = authClient
	return service
}

// CreateInvite creates a seat-targeted invite.
func (s *InviteService) CreateInvite(ctx context.Context, in *campaignv1.CreateInviteRequest) (*campaignv1.CreateInviteResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create invite request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	inv, err := newInviteApplication(s).CreateInvite(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.CreateInviteResponse{Invite: inviteToProto(inv)}, nil
}

// ClaimInvite claims a seat-targeted invite.
func (s *InviteService) ClaimInvite(ctx context.Context, in *campaignv1.ClaimInviteRequest) (*campaignv1.ClaimInviteResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "claim invite request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	inv, participantRecord, err := newInviteApplication(s).ClaimInvite(ctx, campaignID, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.ClaimInviteResponse{
		Invite:      inviteToProto(inv),
		Participant: participantToProto(participantRecord),
	}, nil
}

// GetInvite returns an invite by ID.
func (s *InviteService) GetInvite(ctx context.Context, in *campaignv1.GetInviteRequest) (*campaignv1.GetInviteResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get invite request is required")
	}
	inviteID := strings.TrimSpace(in.GetInviteId())
	if inviteID == "" {
		return nil, status.Error(codes.InvalidArgument, "invite id is required")
	}

	inv, err := s.stores.Invite.GetInvite(ctx, inviteID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	campaignRecord, err := s.stores.Campaign.Get(ctx, inv.CampaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, handleDomainError(err)
	}
	if err := requirePolicy(ctx, s.stores, policy.ActionManageInvites, campaignRecord); err != nil {
		return nil, err
	}

	return &campaignv1.GetInviteResponse{Invite: inviteToProto(inv)}, nil
}

// ListInvites returns a page of invites for a campaign.
func (s *InviteService) ListInvites(ctx context.Context, in *campaignv1.ListInvitesRequest) (*campaignv1.ListInvitesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list invites request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpRead); err != nil {
		return nil, handleDomainError(err)
	}
	if err := requirePolicy(ctx, s.stores, policy.ActionManageInvites, campaignRecord); err != nil {
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
		return nil, status.Errorf(codes.Internal, "list invites: %v", err)
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

// ListPendingInvites returns a page of pending invites for a campaign.
func (s *InviteService) ListPendingInvites(ctx context.Context, in *campaignv1.ListPendingInvitesRequest) (*campaignv1.ListPendingInvitesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list pending invites request is required")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := requirePolicy(ctx, s.stores, policy.ActionManageInvites, campaignRecord); err != nil {
		return nil, err
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListInvitesPageSize,
		Max:     maxListInvitesPageSize,
	})

	page, err := s.stores.Invite.ListPendingInvites(ctx, campaignID, pageSize, in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list pending invites: %v", err)
	}

	response := &campaignv1.ListPendingInvitesResponse{NextPageToken: page.NextPageToken}
	if len(page.Invites) == 0 {
		return response, nil
	}

	participants, err := s.stores.Participant.ListParticipantsByCampaign(ctx, campaignID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list participants: %v", err)
	}
	participantsByID := make(map[string]participant.Participant, len(participants))
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
				if s.authClient == nil {
					return nil, status.Error(codes.Internal, "auth client is not configured")
				}
				cached, ok := userCache[creatorUserID]
				if !ok {
					userResponse, err := s.authClient.GetUser(ctx, &authv1.GetUserRequest{UserId: creatorUserID})
					if err != nil {
						return nil, status.Errorf(codes.Internal, "get auth user: %v", err)
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
			Invite:        inviteToProto(inv),
			Participant:   participantToProto(seat),
			CreatedByUser: createdByUser,
		})
	}

	return response, nil
}

// ListPendingInvitesForUser returns a page of pending invites for the current user.
func (s *InviteService) ListPendingInvitesForUser(ctx context.Context, in *campaignv1.ListPendingInvitesForUserRequest) (*campaignv1.ListPendingInvitesForUserResponse, error) {
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

	page, err := s.stores.Invite.ListPendingInvitesForRecipient(ctx, userID, pageSize, in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list pending invites for user: %v", err)
	}

	response := &campaignv1.ListPendingInvitesForUserResponse{NextPageToken: page.NextPageToken}
	if len(page.Invites) == 0 {
		return response, nil
	}

	campaignsByID := make(map[string]campaign.Campaign)
	participantsByID := make(map[string]participant.Participant)
	response.Invites = make([]*campaignv1.PendingUserInvite, 0, len(page.Invites))
	for _, inv := range page.Invites {
		campaignRecord, ok := campaignsByID[inv.CampaignID]
		if !ok {
			record, err := s.stores.Campaign.Get(ctx, inv.CampaignID)
			if err != nil {
				return nil, handleDomainError(err)
			}
			campaignRecord = record
			campaignsByID[inv.CampaignID] = campaignRecord
		}

		participantKey := inv.CampaignID + ":" + inv.ParticipantID
		seat, ok := participantsByID[participantKey]
		if !ok {
			record, err := s.stores.Participant.GetParticipant(ctx, inv.CampaignID, inv.ParticipantID)
			if err != nil {
				return nil, handleDomainError(err)
			}
			seat = record
			participantsByID[participantKey] = seat
		}

		response.Invites = append(response.Invites, &campaignv1.PendingUserInvite{
			Invite:      inviteToProto(inv),
			Campaign:    campaignToProto(campaignRecord),
			Participant: participantToProto(seat),
		})
	}

	return response, nil
}

// RevokeInvite revokes an invite.
func (s *InviteService) RevokeInvite(ctx context.Context, in *campaignv1.RevokeInviteRequest) (*campaignv1.RevokeInviteResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "revoke invite request is required")
	}
	inviteID := strings.TrimSpace(in.GetInviteId())
	if inviteID == "" {
		return nil, status.Error(codes.InvalidArgument, "invite id is required")
	}

	updated, err := newInviteApplication(s).RevokeInvite(ctx, in)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, err
	}

	return &campaignv1.RevokeInviteResponse{Invite: inviteToProto(updated)}, nil
}
