package userhub

import (
	"context"
	"errors"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/userhub/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// dashboardGetter is the domain behavior surface consumed by the transport layer.
type dashboardGetter interface {
	GetDashboard(ctx context.Context, input domain.GetDashboardInput) (domain.Dashboard, error)
}

// Service exposes userhub.v1 gRPC endpoints.
type Service struct {
	userhubv1.UnimplementedUserHubServiceServer
	domain dashboardGetter
}

// NewService constructs a userhub gRPC service.
func NewService(domainSvc dashboardGetter) *Service {
	return &Service{domain: domainSvc}
}

// GetDashboard returns one user dashboard summary for the caller.
func (s *Service) GetDashboard(ctx context.Context, in *userhubv1.GetDashboardRequest) (*userhubv1.GetDashboardResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get dashboard request is required")
	}
	if s == nil || s.domain == nil {
		return nil, status.Error(codes.Internal, "userhub domain service is not configured")
	}

	userID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	dashboard, err := s.domain.GetDashboard(ctx, domain.GetDashboardInput{
		UserID:               userID,
		Locale:               normalizeLocale(in.GetLocale()),
		CampaignPreviewLimit: int(in.GetCampaignPreviewLimit()),
		InvitePreviewLimit:   int(in.GetInvitePreviewLimit()),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &userhubv1.GetDashboardResponse{
		Metadata:      dashboardMetadataToProto(dashboard.Metadata),
		User:          userSummaryToProto(dashboard.User),
		Invites:       inviteSummaryToProto(dashboard.Invites),
		Notifications: notificationSummaryToProto(dashboard.Notifications),
		Campaigns:     campaignSummaryToProto(dashboard.Campaigns),
		NextActions:   dashboardActionsToProto(dashboard.NextActions),
	}, nil
}

// normalizeLocale returns a stable locale key for cache partitioning.
func normalizeLocale(locale commonv1.Locale) string {
	if locale == commonv1.Locale_LOCALE_UNSPECIFIED {
		return ""
	}
	return locale.String()
}

// mapDomainError maps domain failures to transport status codes.
func mapDomainError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrUserIDRequired):
		return status.Error(codes.PermissionDenied, domain.ErrUserIDRequired.Error())
	case errors.Is(err, domain.ErrServiceNotConfigured):
		return status.Error(codes.Internal, domain.ErrServiceNotConfigured.Error())
	case errors.Is(err, domain.ErrGameGatewayNotConfigured):
		return status.Error(codes.Internal, domain.ErrGameGatewayNotConfigured.Error())
	case errors.Is(err, domain.ErrSocialGatewayNotConfigured):
		return status.Error(codes.Internal, domain.ErrSocialGatewayNotConfigured.Error())
	case errors.Is(err, domain.ErrNotificationsGatewayNotConfigured):
		return status.Error(codes.Internal, domain.ErrNotificationsGatewayNotConfigured.Error())
	default:
		var dependencyErr *domain.DependencyUnavailableError
		if errors.As(err, &dependencyErr) {
			return status.Errorf(codes.Unavailable, "dashboard dependency unavailable: %s", dependencyErr.Dependency)
		}
		return status.Errorf(codes.Internal, "userhub domain: %v", err)
	}
}

// dashboardMetadataToProto maps domain metadata to proto metadata.
func dashboardMetadataToProto(metadata domain.DashboardMetadata) *userhubv1.DashboardMetadata {
	return &userhubv1.DashboardMetadata{
		Freshness:            freshnessToProto(metadata.Freshness),
		CacheHit:             metadata.CacheHit,
		Degraded:             metadata.Degraded,
		DegradedDependencies: append([]string{}, metadata.DegradedDependencies...),
		GeneratedAt:          timestamppb.New(metadata.GeneratedAt),
	}
}

// userSummaryToProto maps user summary values to proto.
func userSummaryToProto(summary domain.UserSummary) *userhubv1.UserSummary {
	return &userhubv1.UserSummary{
		UserId:                 summary.UserID,
		Username:               summary.Username,
		Name:                   summary.Name,
		ProfileAvailable:       summary.ProfileAvailable,
		Discoverable:           summary.Discoverable,
		NeedsProfileCompletion: summary.NeedsProfileCompletion,
	}
}

// inviteSummaryToProto maps invite summary values to proto.
func inviteSummaryToProto(summary domain.InviteSummary) *userhubv1.InviteSummary {
	result := &userhubv1.InviteSummary{
		Available:   summary.Available,
		ListedCount: int32(summary.ListedCount),
		HasMore:     summary.HasMore,
		Pending:     make([]*userhubv1.PendingInvite, 0, len(summary.Pending)),
	}
	for _, invite := range summary.Pending {
		result.Pending = append(result.Pending, &userhubv1.PendingInvite{
			InviteId:      invite.InviteID,
			CampaignId:    invite.CampaignID,
			CampaignName:  invite.CampaignName,
			ParticipantId: invite.ParticipantID,
			CreatedAt:     timestamppb.New(invite.CreatedAt),
		})
	}
	return result
}

// notificationSummaryToProto maps notification summary values to proto.
func notificationSummaryToProto(summary domain.NotificationSummary) *userhubv1.NotificationSummary {
	return &userhubv1.NotificationSummary{
		Available:   summary.Available,
		HasUnread:   summary.HasUnread,
		UnreadCount: int32(summary.UnreadCount),
	}
}

// campaignSummaryToProto maps campaign summary values to proto.
func campaignSummaryToProto(summary domain.CampaignSummary) *userhubv1.CampaignSummary {
	result := &userhubv1.CampaignSummary{
		Available:   summary.Available,
		ListedCount: int32(summary.ListedCount),
		ActiveCount: int32(summary.ActiveCount),
		HasMore:     summary.HasMore,
		Campaigns:   make([]*userhubv1.CampaignPreview, 0, len(summary.Campaigns)),
	}
	for _, campaign := range summary.Campaigns {
		result.Campaigns = append(result.Campaigns, &userhubv1.CampaignPreview{
			CampaignId:       campaign.CampaignID,
			Name:             campaign.Name,
			Status:           campaignStatusToProto(campaign.Status),
			ParticipantCount: int32(campaign.ParticipantCount),
			CharacterCount:   int32(campaign.CharacterCount),
			UpdatedAt:        timestamppb.New(campaign.UpdatedAt),
		})
	}
	return result
}

// dashboardActionsToProto maps action lists to proto values.
func dashboardActionsToProto(actions []domain.DashboardAction) []*userhubv1.DashboardAction {
	if len(actions) == 0 {
		return nil
	}
	result := make([]*userhubv1.DashboardAction, 0, len(actions))
	for _, action := range actions {
		result = append(result, &userhubv1.DashboardAction{
			Id:       dashboardActionToProto(action.ID),
			Priority: int32(action.Priority),
		})
	}
	return result
}

// freshnessToProto maps domain freshness enums to proto enums.
func freshnessToProto(value domain.Freshness) userhubv1.DashboardFreshness {
	switch value {
	case domain.FreshnessFresh:
		return userhubv1.DashboardFreshness_DASHBOARD_FRESHNESS_FRESH
	case domain.FreshnessStale:
		return userhubv1.DashboardFreshness_DASHBOARD_FRESHNESS_STALE
	default:
		return userhubv1.DashboardFreshness_DASHBOARD_FRESHNESS_UNSPECIFIED
	}
}

// campaignStatusToProto maps domain campaign status enums to proto enums.
func campaignStatusToProto(value domain.CampaignStatus) userhubv1.CampaignStatus {
	switch value {
	case domain.CampaignStatusDraft:
		return userhubv1.CampaignStatus_CAMPAIGN_STATUS_DRAFT
	case domain.CampaignStatusActive:
		return userhubv1.CampaignStatus_CAMPAIGN_STATUS_ACTIVE
	case domain.CampaignStatusCompleted:
		return userhubv1.CampaignStatus_CAMPAIGN_STATUS_COMPLETED
	case domain.CampaignStatusArchived:
		return userhubv1.CampaignStatus_CAMPAIGN_STATUS_ARCHIVED
	default:
		return userhubv1.CampaignStatus_CAMPAIGN_STATUS_UNSPECIFIED
	}
}

// dashboardActionToProto maps domain action IDs to proto action IDs.
func dashboardActionToProto(value domain.DashboardActionID) userhubv1.DashboardActionID {
	switch value {
	case domain.DashboardActionReviewPendingInvites:
		return userhubv1.DashboardActionID_DASHBOARD_ACTION_ID_REVIEW_PENDING_INVITES
	case domain.DashboardActionCompleteProfile:
		return userhubv1.DashboardActionID_DASHBOARD_ACTION_ID_COMPLETE_PROFILE
	case domain.DashboardActionCreateOrJoinCampaign:
		return userhubv1.DashboardActionID_DASHBOARD_ACTION_ID_CREATE_OR_JOIN_CAMPAIGN
	case domain.DashboardActionContinueActiveCampaign:
		return userhubv1.DashboardActionID_DASHBOARD_ACTION_ID_CONTINUE_ACTIVE_CAMPAIGN
	case domain.DashboardActionReviewNotifications:
		return userhubv1.DashboardActionID_DASHBOARD_ACTION_ID_REVIEW_NOTIFICATIONS
	default:
		return userhubv1.DashboardActionID_DASHBOARD_ACTION_ID_UNSPECIFIED
	}
}
