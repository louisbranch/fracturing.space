package app

import (
	"context"
	"errors"
	"strings"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"github.com/louisbranch/fracturing.space/internal/services/userhub/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// grpcGameGateway adapts game.v1 clients to domain game gateway behavior.
type grpcGameGateway struct {
	campaigns gamev1.CampaignServiceClient
	invites   gamev1.InviteServiceClient
}

// newGRPCGameGateway constructs a game gateway from gRPC clients.
func newGRPCGameGateway(campaigns gamev1.CampaignServiceClient, invites gamev1.InviteServiceClient) *grpcGameGateway {
	return &grpcGameGateway{
		campaigns: campaigns,
		invites:   invites,
	}
}

// ListCampaignPreviews resolves one page of user-scoped campaign previews.
func (g *grpcGameGateway) ListCampaignPreviews(ctx context.Context, userID string, limit int) (domain.CampaignPage, error) {
	if g == nil || g.campaigns == nil {
		return domain.CampaignPage{}, errors.New("game campaign client is not configured")
	}
	callCtx := grpcauthctx.WithUserID(ctx, userID)
	resp, err := g.campaigns.ListCampaigns(callCtx, &gamev1.ListCampaignsRequest{PageSize: int32(limit)})
	if err != nil {
		return domain.CampaignPage{}, err
	}
	page := domain.CampaignPage{
		Campaigns: make([]domain.CampaignPreview, 0, len(resp.GetCampaigns())),
		HasMore:   strings.TrimSpace(resp.GetNextPageToken()) != "",
	}
	for _, campaign := range resp.GetCampaigns() {
		if campaign == nil {
			continue
		}
		page.Campaigns = append(page.Campaigns, domain.CampaignPreview{
			CampaignID:       campaign.GetId(),
			Name:             campaign.GetName(),
			Status:           campaignStatusFromProto(campaign.GetStatus()),
			ParticipantCount: int(campaign.GetParticipantCount()),
			CharacterCount:   int(campaign.GetCharacterCount()),
			UpdatedAt:        campaign.GetUpdatedAt().AsTime(),
		})
	}
	return page, nil
}

// ListPendingInvitePreviews resolves one page of user-scoped pending invites.
func (g *grpcGameGateway) ListPendingInvitePreviews(ctx context.Context, userID string, limit int) (domain.InvitePage, error) {
	if g == nil || g.invites == nil {
		return domain.InvitePage{}, errors.New("game invite client is not configured")
	}
	callCtx := grpcauthctx.WithUserID(ctx, userID)
	resp, err := g.invites.ListPendingInvitesForUser(callCtx, &gamev1.ListPendingInvitesForUserRequest{PageSize: int32(limit)})
	if err != nil {
		return domain.InvitePage{}, err
	}
	page := domain.InvitePage{
		Invites: make([]domain.PendingInvite, 0, len(resp.GetInvites())),
		HasMore: strings.TrimSpace(resp.GetNextPageToken()) != "",
	}
	for _, pending := range resp.GetInvites() {
		if pending == nil || pending.GetInvite() == nil {
			continue
		}
		invite := pending.GetInvite()
		page.Invites = append(page.Invites, domain.PendingInvite{
			InviteID:      invite.GetId(),
			CampaignID:    invite.GetCampaignId(),
			CampaignName:  pending.GetCampaign().GetName(),
			ParticipantID: invite.GetParticipantId(),
			CreatedAt:     invite.GetCreatedAt().AsTime(),
		})
	}
	return page, nil
}

// grpcSocialGateway adapts social.v1 clients to domain profile lookups.
type grpcSocialGateway struct {
	profiles socialv1.SocialServiceClient
}

// newGRPCSocialGateway constructs a social gateway from a gRPC client.
func newGRPCSocialGateway(profiles socialv1.SocialServiceClient) *grpcSocialGateway {
	return &grpcSocialGateway{profiles: profiles}
}

// GetUserProfile resolves one user profile by user ID.
func (g *grpcSocialGateway) GetUserProfile(ctx context.Context, userID string) (domain.UserProfile, error) {
	if g == nil || g.profiles == nil {
		return domain.UserProfile{}, errors.New("social client is not configured")
	}
	resp, err := g.profiles.GetUserProfile(grpcauthctx.WithUserID(ctx, userID), &socialv1.GetUserProfileRequest{UserId: userID})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return domain.UserProfile{}, domain.ErrProfileNotFound
		}
		return domain.UserProfile{}, err
	}
	profile := resp.GetUserProfile()
	if profile == nil {
		return domain.UserProfile{}, domain.ErrProfileNotFound
	}
	return domain.UserProfile{
		Username: profile.GetUsername(),
		Name:     profile.GetName(),
	}, nil
}

// grpcNotificationsGateway adapts notifications.v1 unread status lookups.
type grpcNotificationsGateway struct {
	notifications notificationsv1.NotificationServiceClient
}

// newGRPCNotificationsGateway constructs a notifications gateway from a gRPC client.
func newGRPCNotificationsGateway(notifications notificationsv1.NotificationServiceClient) *grpcNotificationsGateway {
	return &grpcNotificationsGateway{notifications: notifications}
}

// GetUnreadStatus resolves caller unread-notification status.
func (g *grpcNotificationsGateway) GetUnreadStatus(ctx context.Context, userID string) (domain.UnreadStatus, error) {
	if g == nil || g.notifications == nil {
		return domain.UnreadStatus{}, errors.New("notifications client is not configured")
	}
	resp, err := g.notifications.GetUnreadNotificationStatus(
		grpcauthctx.WithUserID(ctx, userID),
		&notificationsv1.GetUnreadNotificationStatusRequest{},
	)
	if err != nil {
		return domain.UnreadStatus{}, err
	}
	return domain.UnreadStatus{
		HasUnread:   resp.GetHasUnread(),
		UnreadCount: int(resp.GetUnreadCount()),
	}, nil
}

// campaignStatusFromProto maps game campaign status to userhub domain status.
func campaignStatusFromProto(value gamev1.CampaignStatus) domain.CampaignStatus {
	switch value {
	case gamev1.CampaignStatus_DRAFT:
		return domain.CampaignStatusDraft
	case gamev1.CampaignStatus_ACTIVE:
		return domain.CampaignStatusActive
	case gamev1.CampaignStatus_COMPLETED:
		return domain.CampaignStatusCompleted
	case gamev1.CampaignStatus_ARCHIVED:
		return domain.CampaignStatusArchived
	default:
		return domain.CampaignStatusUnspecified
	}
}

var (
	_ domain.GameGateway          = (*grpcGameGateway)(nil)
	_ domain.SocialGateway        = (*grpcSocialGateway)(nil)
	_ domain.NotificationsGateway = (*grpcNotificationsGateway)(nil)
)
