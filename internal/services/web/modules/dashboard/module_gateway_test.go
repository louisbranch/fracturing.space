package dashboard

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"google.golang.org/grpc"
	grpcmetadata "google.golang.org/grpc/metadata"
)

func TestNewGRPCGatewayWithoutClientFallsBackToUnavailableGateway(t *testing.T) {
	t.Parallel()

	gateway := NewGRPCGateway(module.Dependencies{})
	snapshot, err := gateway.LoadDashboard(context.Background(), "user-1", commonv1.Locale_LOCALE_EN_US)
	if err != nil {
		t.Fatalf("LoadDashboard() error = %v", err)
	}
	if snapshot.NeedsProfileCompletion {
		t.Fatalf("NeedsProfileCompletion = true, want false")
	}
	if snapshot.HasDraftOrActiveCampaign {
		t.Fatalf("HasDraftOrActiveCampaign = true, want false")
	}
	if snapshot.CampaignsHasMore {
		t.Fatalf("CampaignsHasMore = true, want false")
	}
}

func TestGRPCGatewayMapsDashboardResponseAuthMetadataAndCampaignState(t *testing.T) {
	t.Parallel()

	client := &dashboardUserHubClientRecorder{resp: &userhubv1.GetDashboardResponse{
		User:     &userhubv1.UserSummary{NeedsProfileCompletion: true},
		Metadata: &userhubv1.DashboardMetadata{DegradedDependencies: []string{" social.profile ", ""}},
		Campaigns: &userhubv1.CampaignSummary{
			HasMore: true,
			Campaigns: []*userhubv1.CampaignPreview{
				{CampaignId: "camp-completed", Status: userhubv1.CampaignStatus_CAMPAIGN_STATUS_COMPLETED},
				{CampaignId: "camp-active", Status: userhubv1.CampaignStatus_CAMPAIGN_STATUS_ACTIVE},
			},
		},
	}}
	gateway := NewGRPCGateway(module.Dependencies{UserHubClient: client})

	snapshot, err := gateway.LoadDashboard(context.Background(), " user-1 ", commonv1.Locale_LOCALE_UNSPECIFIED)
	if err != nil {
		t.Fatalf("LoadDashboard() error = %v", err)
	}
	if !snapshot.NeedsProfileCompletion {
		t.Fatalf("NeedsProfileCompletion = false, want true")
	}
	if len(snapshot.DegradedDependencies) != 1 || snapshot.DegradedDependencies[0] != "social.profile" {
		t.Fatalf("DegradedDependencies = %v, want [social.profile]", snapshot.DegradedDependencies)
	}
	if !snapshot.HasDraftOrActiveCampaign {
		t.Fatalf("HasDraftOrActiveCampaign = false, want true")
	}
	if !snapshot.CampaignsHasMore {
		t.Fatalf("CampaignsHasMore = false, want true")
	}
	if client.lastReq == nil {
		t.Fatalf("expected dashboard request")
	}
	if client.lastReq.GetLocale() != commonv1.Locale_LOCALE_EN_US {
		t.Fatalf("request locale = %v, want %v", client.lastReq.GetLocale(), commonv1.Locale_LOCALE_EN_US)
	}
	if client.lastReq.GetCampaignPreviewLimit() != 10 {
		t.Fatalf("campaign preview limit = %d, want %d", client.lastReq.GetCampaignPreviewLimit(), 10)
	}
	if client.lastUserID != "user-1" {
		t.Fatalf("metadata user-id = %q, want %q", client.lastUserID, "user-1")
	}
}

func TestHasDraftOrActiveCampaign(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		campaigns []*userhubv1.CampaignPreview
		want      bool
	}{
		{
			name: "draft present",
			campaigns: []*userhubv1.CampaignPreview{
				{Status: userhubv1.CampaignStatus_CAMPAIGN_STATUS_DRAFT},
			},
			want: true,
		},
		{
			name: "active present",
			campaigns: []*userhubv1.CampaignPreview{
				{Status: userhubv1.CampaignStatus_CAMPAIGN_STATUS_ACTIVE},
			},
			want: true,
		},
		{
			name: "completed only",
			campaigns: []*userhubv1.CampaignPreview{
				{Status: userhubv1.CampaignStatus_CAMPAIGN_STATUS_COMPLETED},
			},
			want: false,
		},
		{
			name:      "empty",
			campaigns: nil,
			want:      false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := hasDraftOrActiveCampaign(tc.campaigns); got != tc.want {
				t.Fatalf("hasDraftOrActiveCampaign() = %t, want %t", got, tc.want)
			}
		})
	}
}

type dashboardUserHubClientRecorder struct {
	resp       *userhubv1.GetDashboardResponse
	err        error
	lastReq    *userhubv1.GetDashboardRequest
	lastUserID string
}

func (r *dashboardUserHubClientRecorder) GetDashboard(ctx context.Context, req *userhubv1.GetDashboardRequest, _ ...grpc.CallOption) (*userhubv1.GetDashboardResponse, error) {
	r.lastReq = req
	if md, ok := grpcmetadata.FromOutgoingContext(ctx); ok {
		values := md.Get(grpcmeta.UserIDHeader)
		if len(values) > 0 {
			r.lastUserID = values[0]
		}
	}
	if r.err != nil {
		return nil, r.err
	}
	return r.resp, nil
}
