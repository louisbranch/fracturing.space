package dashboard

import (
	"context"
	"strings"

	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"google.golang.org/grpc"

	"golang.org/x/text/language"
)

// UserHubClient exposes user-dashboard aggregation operations.
type UserHubClient interface {
	GetDashboard(context.Context, *userhubv1.GetDashboardRequest, ...grpc.CallOption) (*userhubv1.GetDashboardResponse, error)
}

const maxDashboardCampaignPreviewLimit = 10

// NewGRPCGateway builds the production dashboard gateway from the UserHub client.
func NewGRPCGateway(client UserHubClient) DashboardGateway {
	if client == nil {
		return unavailableGateway{}
	}
	return grpcGateway{client: client}
}

type grpcGateway struct {
	client UserHubClient
}

func (g grpcGateway) LoadDashboard(ctx context.Context, userID string, localeTag language.Tag) (DashboardSnapshot, error) {
	if g.client == nil {
		return DashboardSnapshot{}, nil
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return DashboardSnapshot{}, nil
	}
	resp, err := g.client.GetDashboard(
		grpcauthctx.WithUserID(ctx, userID),
		&userhubv1.GetDashboardRequest{
			Locale:               platformi18n.LocaleForTag(localeTag),
			CampaignPreviewLimit: maxDashboardCampaignPreviewLimit,
		},
	)
	if err != nil {
		return DashboardSnapshot{}, err
	}
	if resp == nil {
		return DashboardSnapshot{}, nil
	}
	return DashboardSnapshot{
		NeedsProfileCompletion:   resp.GetUser().GetNeedsProfileCompletion(),
		HasDraftOrActiveCampaign: hasDraftOrActiveCampaign(resp.GetCampaigns().GetCampaigns()),
		CampaignsHasMore:         resp.GetCampaigns().GetHasMore(),
		DegradedDependencies:     normalizedDependencies(resp.GetMetadata().GetDegradedDependencies()),
	}, nil
}

func hasDraftOrActiveCampaign(campaigns []*userhubv1.CampaignPreview) bool {
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
