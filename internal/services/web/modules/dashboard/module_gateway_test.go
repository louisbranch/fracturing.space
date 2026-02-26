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
}

func TestGRPCGatewayMapsDashboardResponseAndAuthMetadata(t *testing.T) {
	t.Parallel()

	client := &dashboardUserHubClientRecorder{resp: &userhubv1.GetDashboardResponse{
		User:     &userhubv1.UserSummary{NeedsProfileCompletion: true},
		Metadata: &userhubv1.DashboardMetadata{DegradedDependencies: []string{" social.profile ", ""}},
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
	if client.lastReq == nil {
		t.Fatalf("expected dashboard request")
	}
	if client.lastReq.GetLocale() != commonv1.Locale_LOCALE_EN_US {
		t.Fatalf("request locale = %v, want %v", client.lastReq.GetLocale(), commonv1.Locale_LOCALE_EN_US)
	}
	if client.lastUserID != "user-1" {
		t.Fatalf("metadata user-id = %q, want %q", client.lastUserID, "user-1")
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
