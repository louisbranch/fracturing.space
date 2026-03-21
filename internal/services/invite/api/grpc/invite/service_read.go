package invite

import (
	"context"
	"errors"
	"strings"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	"github.com/louisbranch/fracturing.space/internal/services/invite/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) GetInvite(ctx context.Context, in *invitev1.GetInviteRequest) (*invitev1.GetInviteResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	inviteID := strings.TrimSpace(in.GetInviteId())
	if inviteID == "" {
		return nil, status.Error(codes.InvalidArgument, "invite_id is required")
	}

	inv, err := s.store.GetInvite(ctx, inviteID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "invite not found")
		}
		return nil, status.Errorf(codes.Internal, "load invite: %v", err)
	}

	return &invitev1.GetInviteResponse{Invite: inviteToProto(inv)}, nil
}

func (s *Service) GetPublicInvite(ctx context.Context, in *invitev1.GetPublicInviteRequest) (*invitev1.GetPublicInviteResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	inviteID := strings.TrimSpace(in.GetInviteId())
	if inviteID == "" {
		return nil, status.Error(codes.InvalidArgument, "invite_id is required")
	}

	inv, err := s.store.GetInvite(ctx, inviteID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "invite not found")
		}
		return nil, status.Errorf(codes.Internal, "load invite: %v", err)
	}

	resp := &invitev1.GetPublicInviteResponse{
		Invite: inviteToProto(inv),
	}

	// Game reads use admin override because the invite viewer may not yet be a
	// campaign participant.
	gameCtx := gameReadContext(ctx)

	// Load campaign summary from game service.
	if s.gameCampaign != nil {
		campaignResp, err := s.gameCampaign.GetCampaign(gameCtx, &gamev1.GetCampaignRequest{CampaignId: inv.CampaignID})
		if err == nil && campaignResp.GetCampaign() != nil {
			c := campaignResp.GetCampaign()
			resp.Campaign = &invitev1.InviteCampaignSummary{
				Id:     c.GetId(),
				Name:   c.GetName(),
				Status: c.GetStatus().String(),
			}
		}
	}

	// Load participant summary from game service.
	if s.game != nil {
		participantResp, err := s.game.GetParticipant(gameCtx, &gamev1.GetParticipantRequest{
			CampaignId:    inv.CampaignID,
			ParticipantId: inv.ParticipantID,
		})
		if err == nil && participantResp.GetParticipant() != nil {
			p := participantResp.GetParticipant()
			resp.Participant = &invitev1.InviteParticipantSummary{
				Id:     p.GetId(),
				Name:   p.GetName(),
				Role:   p.GetRole().String(),
				UserId: p.GetUserId(),
			}
		}
	}

	// Load creator user info from auth service.
	if s.auth != nil && inv.CreatedByParticipantID != "" {
		resp.CreatedByUser = loadCreatorUser(gameCtx, s.game, s.auth, inv.CampaignID, inv.CreatedByParticipantID)
	}

	return resp, nil
}

func (s *Service) ListInvites(ctx context.Context, in *invitev1.ListInvitesRequest) (*invitev1.ListInvitesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	statusFilter := statusFromProto(in.GetStatus())
	page, err := s.store.ListInvites(ctx,
		strings.TrimSpace(in.GetCampaignId()),
		strings.TrimSpace(in.GetRecipientUserId()),
		statusFilter,
		int(in.GetPageSize()),
		in.GetPageToken(),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list invites: %v", err)
	}
	return &invitev1.ListInvitesResponse{
		Invites:       invitesToProto(page.Invites),
		NextPageToken: page.NextPageToken,
	}, nil
}

func (s *Service) ListPendingInvites(ctx context.Context, in *invitev1.ListPendingInvitesRequest) (*invitev1.ListPendingInvitesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	page, err := s.store.ListPendingInvites(ctx,
		strings.TrimSpace(in.GetCampaignId()),
		int(in.GetPageSize()),
		in.GetPageToken(),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list pending invites: %v", err)
	}
	return &invitev1.ListPendingInvitesResponse{
		Invites:       invitesToProto(page.Invites),
		NextPageToken: page.NextPageToken,
	}, nil
}

func (s *Service) ListPendingInvitesForUser(ctx context.Context, in *invitev1.ListPendingInvitesForUserRequest) (*invitev1.ListPendingInvitesForUserResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "user identity is required")
	}
	page, err := s.store.ListPendingInvitesForRecipient(ctx, userID, int(in.GetPageSize()), in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list pending invites for user: %v", err)
	}

	gameCtx := gameReadContext(ctx)
	entries := make([]*invitev1.PendingInviteForUserEntry, 0, len(page.Invites))
	for _, inv := range page.Invites {
		entry := &invitev1.PendingInviteForUserEntry{
			Invite: inviteToProto(inv),
		}
		if s.gameCampaign != nil {
			if resp, err := s.gameCampaign.GetCampaign(gameCtx, &gamev1.GetCampaignRequest{CampaignId: inv.CampaignID}); err == nil && resp.GetCampaign() != nil {
				c := resp.GetCampaign()
				entry.Campaign = &invitev1.InviteCampaignSummary{Id: c.GetId(), Name: c.GetName(), Status: c.GetStatus().String()}
			}
		}
		if s.game != nil {
			if resp, err := s.game.GetParticipant(gameCtx, &gamev1.GetParticipantRequest{CampaignId: inv.CampaignID, ParticipantId: inv.ParticipantID}); err == nil && resp.GetParticipant() != nil {
				p := resp.GetParticipant()
				entry.Participant = &invitev1.InviteParticipantSummary{Id: p.GetId(), Name: p.GetName(), Role: p.GetRole().String(), UserId: p.GetUserId()}
			}
		}
		entries = append(entries, entry)
	}

	return &invitev1.ListPendingInvitesForUserResponse{
		Invites:       entries,
		NextPageToken: page.NextPageToken,
	}, nil
}

// gameReadContext builds an outgoing context with admin override so the invite
// service can read campaign/participant data for users who are not yet members.
func gameReadContext(ctx context.Context) context.Context {
	return grpcauthctx.WithAdminOverride(ctx, "invite service enrichment")
}

// userHasSeatInCampaign checks whether a user already has a bound participant
// seat in the campaign by paging through the participant list.
func userHasSeatInCampaign(ctx context.Context, game gamev1.ParticipantServiceClient, campaignID, userID string) bool {
	pageToken := ""
	for {
		resp, err := game.ListParticipants(ctx, &gamev1.ListParticipantsRequest{
			CampaignId: campaignID,
			PageSize:   10,
			PageToken:  pageToken,
		})
		if err != nil {
			return false
		}
		for _, p := range resp.GetParticipants() {
			if strings.TrimSpace(p.GetUserId()) == userID {
				return true
			}
		}
		next := strings.TrimSpace(resp.GetNextPageToken())
		if next == "" {
			return false
		}
		pageToken = next
	}
}
