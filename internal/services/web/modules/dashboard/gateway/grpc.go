package gateway

import (
	"context"
	"strings"
	"time"

	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	dashboardapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/userid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"golang.org/x/text/language"
)

// UserHubClient exposes user-dashboard aggregation operations.
type UserHubClient interface {
	GetDashboard(context.Context, *userhubv1.GetDashboardRequest, ...grpc.CallOption) (*userhubv1.GetDashboardResponse, error)
}

const MaxDashboardCampaignPreviewLimit = 10

// GRPCGateway maps userhub gRPC responses to the app gateway contract.
type GRPCGateway struct {
	Client UserHubClient
}

// NewGRPCGateway builds the production dashboard gateway from the UserHub client.
func NewGRPCGateway(client UserHubClient) dashboardapp.Gateway {
	if client == nil {
		return dashboardapp.NewUnavailableGateway()
	}
	return GRPCGateway{Client: client}
}

// LoadDashboard loads the package state needed for this request path.
func (g GRPCGateway) LoadDashboard(ctx context.Context, userID string, localeTag language.Tag) (dashboardapp.DashboardSnapshot, error) {
	if g.Client == nil {
		return dashboardapp.DashboardSnapshot{}, nil
	}
	userID = userid.Normalize(userID)
	if userID == "" {
		return dashboardapp.DashboardSnapshot{}, nil
	}
	resp, err := g.Client.GetDashboard(
		grpcauthctx.WithUserID(ctx, userID),
		&userhubv1.GetDashboardRequest{
			Locale:               platformi18n.LocaleForTag(localeTag),
			CampaignPreviewLimit: MaxDashboardCampaignPreviewLimit,
		},
	)
	if err != nil {
		return dashboardapp.DashboardSnapshot{}, err
	}
	if resp == nil {
		return dashboardapp.DashboardSnapshot{}, nil
	}
	return dashboardapp.DashboardSnapshot{
		NeedsProfileCompletion:       resp.GetUser().GetNeedsProfileCompletion(),
		HasDraftOrActiveCampaign:     HasDraftOrActiveCampaign(resp.GetCampaigns().GetCampaigns()),
		CampaignsHasMore:             resp.GetCampaigns().GetHasMore(),
		CampaignStartNudgesAvailable: resp.GetCampaignStartNudges().GetAvailable(),
		CampaignStartNudges:          mapCampaignStartNudges(resp.GetCampaignStartNudges().GetNudges()),
		CampaignStartNudgesHasMore:   resp.GetCampaignStartNudges().GetHasMore(),
		ActiveSessionsAvailable:      resp.GetActiveSessions().GetAvailable(),
		ActiveSessions:               mapActiveSessions(resp.GetActiveSessions().GetSessions()),
		DegradedDependencies:         normalizedDependencies(resp.GetMetadata().GetDegradedDependencies()),
		Freshness:                    mapFreshness(resp.GetMetadata().GetFreshness()),
		CacheHit:                     resp.GetMetadata().GetCacheHit(),
		GeneratedAt:                  protoTime(resp.GetMetadata().GetGeneratedAt()),
	}, nil
}

// HasDraftOrActiveCampaign reports whether previews include a draft or active campaign.
func HasDraftOrActiveCampaign(campaigns []*userhubv1.CampaignPreview) bool {
	for _, campaign := range campaigns {
		if campaign == nil {
			continue
		}
		switch campaign.GetStatus() {
		case userhubv1.CampaignStatus_CAMPAIGN_STATUS_DRAFT,
			userhubv1.CampaignStatus_CAMPAIGN_STATUS_ACTIVE:
			return true
		}
	}
	return false
}

// normalizedDependencies centralizes this web behavior in one helper seam.
func normalizedDependencies(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// mapActiveSessions converts transport rows into app-layer dashboard items.
func mapActiveSessions(sessions []*userhubv1.ActiveSessionPreview) []dashboardapp.ActiveSessionItem {
	if len(sessions) == 0 {
		return nil
	}
	items := make([]dashboardapp.ActiveSessionItem, 0, len(sessions))
	for _, session := range sessions {
		if session == nil {
			continue
		}
		items = append(items, dashboardapp.ActiveSessionItem{
			CampaignID:   session.GetCampaignId(),
			CampaignName: session.GetCampaignName(),
			SessionID:    session.GetSessionId(),
			SessionName:  session.GetSessionName(),
		})
	}
	if len(items) == 0 {
		return nil
	}
	return items
}

// mapCampaignStartNudges keeps the gateway-to-app mapping for readiness nudges local to the dashboard transport seam.
func mapCampaignStartNudges(nudges []*userhubv1.CampaignStartNudge) []dashboardapp.CampaignStartNudgeItem {
	if len(nudges) == 0 {
		return nil
	}
	items := make([]dashboardapp.CampaignStartNudgeItem, 0, len(nudges))
	for _, nudge := range nudges {
		if nudge == nil {
			continue
		}
		items = append(items, dashboardapp.CampaignStartNudgeItem{
			CampaignID:          nudge.GetCampaignId(),
			CampaignName:        nudge.GetCampaignName(),
			BlockerCode:         nudge.GetBlockerCode(),
			BlockerMessage:      nudge.GetBlockerMessage(),
			ActionKind:          campaignStartNudgeActionKindFromProto(nudge.GetActionKind()),
			TargetParticipantID: nudge.GetTargetParticipantId(),
			TargetCharacterID:   nudge.GetTargetCharacterId(),
		})
	}
	if len(items) == 0 {
		return nil
	}
	return items
}

// mapFreshness preserves userhub cache metadata for dashboard logging seams.
func mapFreshness(value userhubv1.DashboardFreshness) dashboardapp.DashboardFreshness {
	switch value {
	case userhubv1.DashboardFreshness_DASHBOARD_FRESHNESS_FRESH:
		return dashboardapp.DashboardFreshnessFresh
	case userhubv1.DashboardFreshness_DASHBOARD_FRESHNESS_STALE:
		return dashboardapp.DashboardFreshnessStale
	default:
		return dashboardapp.DashboardFreshnessUnspecified
	}
}

// protoTime normalizes optional timestamps into zero-safe Go values.
func protoTime(value *timestamppb.Timestamp) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.AsTime()
}

// campaignStartNudgeActionKindFromProto translates userhub CTA enums into dashboard-local action kinds.
func campaignStartNudgeActionKindFromProto(value userhubv1.CampaignStartNudgeActionKind) dashboardapp.CampaignStartNudgeActionKind {
	switch value {
	case userhubv1.CampaignStartNudgeActionKind_CAMPAIGN_START_NUDGE_ACTION_KIND_CREATE_CHARACTER:
		return dashboardapp.CampaignStartNudgeActionKindCreateCharacter
	case userhubv1.CampaignStartNudgeActionKind_CAMPAIGN_START_NUDGE_ACTION_KIND_COMPLETE_CHARACTER:
		return dashboardapp.CampaignStartNudgeActionKindCompleteCharacter
	case userhubv1.CampaignStartNudgeActionKind_CAMPAIGN_START_NUDGE_ACTION_KIND_CONFIGURE_AI_AGENT:
		return dashboardapp.CampaignStartNudgeActionKindConfigureAIAgent
	case userhubv1.CampaignStartNudgeActionKind_CAMPAIGN_START_NUDGE_ACTION_KIND_INVITE_PLAYER:
		return dashboardapp.CampaignStartNudgeActionKindInvitePlayer
	case userhubv1.CampaignStartNudgeActionKind_CAMPAIGN_START_NUDGE_ACTION_KIND_MANAGE_PARTICIPANTS:
		return dashboardapp.CampaignStartNudgeActionKindManageParticipants
	default:
		return dashboardapp.CampaignStartNudgeActionKindUnspecified
	}
}
