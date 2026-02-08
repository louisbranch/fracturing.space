package campaign

import (
	"context"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/campaign/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/campaign"
	"github.com/louisbranch/fracturing.space/internal/campaign/invite"
	"github.com/louisbranch/fracturing.space/internal/campaign/policy"
	"github.com/louisbranch/fracturing.space/internal/id"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultListInvitesPageSize = 10
	maxListInvitesPageSize     = 10
)

// InviteService implements the campaign.v1.InviteService gRPC API.
type InviteService struct {
	campaignv1.UnimplementedInviteServiceServer
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
}

// NewInviteService creates an InviteService with default dependencies.
func NewInviteService(stores Stores) *InviteService {
	return &InviteService{
		stores:      stores,
		clock:       time.Now,
		idGenerator: id.NewID,
	}
}

// CreateInvite creates a seat-targeted invite.
func (s *InviteService) CreateInvite(ctx context.Context, in *campaignv1.CreateInviteRequest) (*campaignv1.CreateInviteResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create invite request is required")
	}
	if s.stores.Invite == nil {
		return nil, status.Error(codes.Internal, "invite store is not configured")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Participant == nil {
		return nil, status.Error(codes.Internal, "participant store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	participantID := strings.TrimSpace(in.GetParticipantId())
	if participantID == "" {
		return nil, status.Error(codes.InvalidArgument, "participant id is required")
	}

	campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, handleDomainError(err)
	}
	if err := requirePolicy(ctx, s.stores, policy.ActionManageInvites, campaignRecord); err != nil {
		return nil, err
	}
	if _, err := s.stores.Participant.GetParticipant(ctx, campaignID, participantID); err != nil {
		return nil, handleDomainError(err)
	}

	created, err := invite.CreateInvite(invite.CreateInviteInput{
		CampaignID:             campaignID,
		ParticipantID:          participantID,
		CreatedByParticipantID: strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx)),
	}, s.clock, s.idGenerator)
	if err != nil {
		return nil, handleDomainError(err)
	}

	if err := s.stores.Invite.PutInvite(ctx, created); err != nil {
		return nil, status.Errorf(codes.Internal, "put invite: %v", err)
	}

	return &campaignv1.CreateInviteResponse{Invite: inviteToProto(created)}, nil
}

// GetInvite returns an invite by ID.
func (s *InviteService) GetInvite(ctx context.Context, in *campaignv1.GetInviteRequest) (*campaignv1.GetInviteResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get invite request is required")
	}
	if s.stores.Invite == nil {
		return nil, status.Error(codes.Internal, "invite store is not configured")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
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
	if s.stores.Invite == nil {
		return nil, status.Error(codes.Internal, "invite store is not configured")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
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

	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = defaultListInvitesPageSize
	}
	if pageSize > maxListInvitesPageSize {
		pageSize = maxListInvitesPageSize
	}

	page, err := s.stores.Invite.ListInvites(ctx, campaignID, pageSize, in.GetPageToken())
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

// RevokeInvite revokes an invite.
func (s *InviteService) RevokeInvite(ctx context.Context, in *campaignv1.RevokeInviteRequest) (*campaignv1.RevokeInviteResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "revoke invite request is required")
	}
	if s.stores.Invite == nil {
		return nil, status.Error(codes.Internal, "invite store is not configured")
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
	if err := requirePolicy(ctx, s.stores, policy.ActionManageInvites, campaignRecord); err != nil {
		return nil, err
	}
	if inv.Status == invite.StatusRevoked {
		return nil, status.Error(codes.FailedPrecondition, "invite already revoked")
	}
	if inv.Status == invite.StatusClaimed {
		return nil, status.Error(codes.FailedPrecondition, "invite already claimed")
	}

	updatedAt := s.clock().UTC()
	if err := s.stores.Invite.UpdateInviteStatus(ctx, inv.ID, invite.StatusRevoked, updatedAt); err != nil {
		return nil, status.Errorf(codes.Internal, "revoke invite: %v", err)
	}
	inv.Status = invite.StatusRevoked
	inv.UpdatedAt = updatedAt

	return &campaignv1.RevokeInviteResponse{Invite: inviteToProto(inv)}, nil
}
