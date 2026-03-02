package gateway

import (
	"context"
	"strings"

	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	dashboardapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/app"
	"google.golang.org/grpc"

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
	userID = strings.TrimSpace(userID)
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
		NeedsProfileCompletion:   resp.GetUser().GetNeedsProfileCompletion(),
		HasDraftOrActiveCampaign: HasDraftOrActiveCampaign(resp.GetCampaigns().GetCampaigns()),
		CampaignsHasMore:         resp.GetCampaigns().GetHasMore(),
		DegradedDependencies:     normalizedDependencies(resp.GetMetadata().GetDegradedDependencies()),
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
