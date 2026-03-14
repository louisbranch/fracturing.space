package app

import (
	"context"
	"errors"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"github.com/louisbranch/fracturing.space/internal/services/userhub/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const readinessCampaignPageSize = 10

// grpcAuthGateway adapts auth.v1 clients to domain identity lookups.
type grpcAuthGateway struct {
	users authv1.AuthServiceClient
}

// newGRPCAuthGateway constructs an auth gateway from a gRPC client.
func newGRPCAuthGateway(users authv1.AuthServiceClient) *grpcAuthGateway {
	return &grpcAuthGateway{users: users}
}

// GetUserIdentity resolves auth-owned identity data by user ID.
func (g *grpcAuthGateway) GetUserIdentity(ctx context.Context, userID string) (domain.UserIdentity, error) {
	if g == nil || g.users == nil {
		return domain.UserIdentity{}, errors.New("auth client is not configured")
	}
	resp, err := g.users.GetUser(grpcauthctx.WithUserID(ctx, userID), &authv1.GetUserRequest{UserId: userID})
	if err != nil {
		return domain.UserIdentity{}, err
	}
	if resp == nil || resp.GetUser() == nil {
		return domain.UserIdentity{}, errors.New("auth user not found")
	}
	return domain.UserIdentity{Username: strings.TrimSpace(resp.GetUser().GetUsername())}, nil
}

// grpcGameGateway adapts game.v1 clients to domain game gateway behavior.
type grpcGameGateway struct {
	campaigns gamev1.CampaignServiceClient
	invites   gamev1.InviteServiceClient
	sessions  gamev1.SessionServiceClient
}

// newGRPCGameGateway constructs a game gateway from gRPC clients.
func newGRPCGameGateway(
	campaigns gamev1.CampaignServiceClient,
	invites gamev1.InviteServiceClient,
	sessions gamev1.SessionServiceClient,
) *grpcGameGateway {
	return &grpcGameGateway{
		campaigns: campaigns,
		invites:   invites,
		sessions:  sessions,
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

// ListReadinessCampaigns resolves all draft/active campaigns eligible for readiness scanning.
func (g *grpcGameGateway) ListReadinessCampaigns(ctx context.Context, userID string) ([]domain.CampaignPreview, error) {
	if g == nil || g.campaigns == nil {
		return nil, errors.New("game campaign client is not configured")
	}
	callCtx := grpcauthctx.WithUserID(ctx, userID)
	pageToken := ""
	campaigns := make([]domain.CampaignPreview, 0, readinessCampaignPageSize)
	for {
		resp, err := g.campaigns.ListCampaigns(callCtx, &gamev1.ListCampaignsRequest{
			PageSize:  readinessCampaignPageSize,
			PageToken: pageToken,
			Statuses: []gamev1.CampaignStatus{
				gamev1.CampaignStatus_DRAFT,
				gamev1.CampaignStatus_ACTIVE,
			},
		})
		if err != nil {
			return nil, err
		}
		for _, campaign := range resp.GetCampaigns() {
			if campaign == nil {
				continue
			}
			campaigns = append(campaigns, domain.CampaignPreview{
				CampaignID:       campaign.GetId(),
				Name:             campaign.GetName(),
				Status:           campaignStatusFromProto(campaign.GetStatus()),
				ParticipantCount: int(campaign.GetParticipantCount()),
				CharacterCount:   int(campaign.GetCharacterCount()),
				UpdatedAt:        campaign.GetUpdatedAt().AsTime(),
			})
		}
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			return campaigns, nil
		}
	}
}

// GetCampaignReadiness resolves localized campaign session-start blockers for one user-scoped campaign.
func (g *grpcGameGateway) GetCampaignReadiness(ctx context.Context, userID, campaignID string) (domain.CampaignReadiness, error) {
	if g == nil || g.campaigns == nil {
		return domain.CampaignReadiness{}, errors.New("game campaign client is not configured")
	}
	callCtx := grpcauthctx.WithUserID(ctx, userID)
	resp, err := g.campaigns.GetCampaignSessionReadiness(callCtx, &gamev1.GetCampaignSessionReadinessRequest{CampaignId: campaignID})
	if err != nil {
		return domain.CampaignReadiness{}, err
	}
	result := domain.CampaignReadiness{
		Blockers: make([]domain.CampaignReadinessBlocker, 0, len(resp.GetReadiness().GetBlockers())),
	}
	for _, blocker := range resp.GetReadiness().GetBlockers() {
		if blocker == nil {
			continue
		}
		result.Blockers = append(result.Blockers, domain.CampaignReadinessBlocker{
			Code:                strings.TrimSpace(blocker.GetCode()),
			Message:             strings.TrimSpace(blocker.GetMessage()),
			ResponsibleUserIDs:  normalizedIDs(blocker.GetAction().GetResponsibleUserIds()),
			ActionKind:          readinessActionKindFromProto(blocker.GetAction().GetResolutionKind()),
			TargetParticipantID: strings.TrimSpace(blocker.GetAction().GetTargetParticipantId()),
			TargetCharacterID:   strings.TrimSpace(blocker.GetAction().GetTargetCharacterId()),
		})
	}
	return result, nil
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
			InviteID:        invite.GetId(),
			CampaignID:      invite.GetCampaignId(),
			CampaignName:    pending.GetCampaign().GetName(),
			ParticipantID:   invite.GetParticipantId(),
			ParticipantName: pending.GetParticipant().GetName(),
			CreatedAt:       invite.GetCreatedAt().AsTime(),
		})
	}
	return page, nil
}

// ListActiveSessionPreviews resolves one page of user-scoped active-session previews.
func (g *grpcGameGateway) ListActiveSessionPreviews(ctx context.Context, userID string, limit int) (domain.ActiveSessionPage, error) {
	if g == nil || g.sessions == nil {
		return domain.ActiveSessionPage{}, errors.New("game session client is not configured")
	}
	callCtx := grpcauthctx.WithUserID(ctx, userID)
	resp, err := g.sessions.ListActiveSessionsForUser(callCtx, &gamev1.ListActiveSessionsForUserRequest{PageSize: int32(limit)})
	if err != nil {
		return domain.ActiveSessionPage{}, err
	}
	page := domain.ActiveSessionPage{
		Sessions: make([]domain.ActiveSessionPreview, 0, len(resp.GetSessions())),
		HasMore:  resp.GetHasMore(),
	}
	for _, session := range resp.GetSessions() {
		if session == nil {
			continue
		}
		page.Sessions = append(page.Sessions, domain.ActiveSessionPreview{
			CampaignID:   session.GetCampaignId(),
			CampaignName: session.GetCampaignName(),
			SessionID:    session.GetSessionId(),
			SessionName:  session.GetSessionName(),
			StartedAt:    session.GetStartedAt().AsTime(),
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
		Name: strings.TrimSpace(profile.GetName()),
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

func readinessActionKindFromProto(value gamev1.CampaignSessionReadinessResolutionKind) domain.CampaignStartNudgeActionKind {
	switch value {
	case gamev1.CampaignSessionReadinessResolutionKind_CAMPAIGN_SESSION_READINESS_RESOLUTION_KIND_CREATE_CHARACTER:
		return domain.CampaignStartNudgeActionCreateCharacter
	case gamev1.CampaignSessionReadinessResolutionKind_CAMPAIGN_SESSION_READINESS_RESOLUTION_KIND_COMPLETE_CHARACTER:
		return domain.CampaignStartNudgeActionCompleteCharacter
	case gamev1.CampaignSessionReadinessResolutionKind_CAMPAIGN_SESSION_READINESS_RESOLUTION_KIND_CONFIGURE_AI_AGENT:
		return domain.CampaignStartNudgeActionConfigureAIAgent
	case gamev1.CampaignSessionReadinessResolutionKind_CAMPAIGN_SESSION_READINESS_RESOLUTION_KIND_INVITE_PLAYER:
		return domain.CampaignStartNudgeActionInvitePlayer
	case gamev1.CampaignSessionReadinessResolutionKind_CAMPAIGN_SESSION_READINESS_RESOLUTION_KIND_MANAGE_PARTICIPANTS:
		return domain.CampaignStartNudgeActionManageParticipants
	default:
		return domain.CampaignStartNudgeActionUnspecified
	}
}

func normalizedIDs(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

var (
	_ domain.AuthGateway          = (*grpcAuthGateway)(nil)
	_ domain.GameGateway          = (*grpcGameGateway)(nil)
	_ domain.SocialGateway        = (*grpcSocialGateway)(nil)
	_ domain.NotificationsGateway = (*grpcNotificationsGateway)(nil)
)
