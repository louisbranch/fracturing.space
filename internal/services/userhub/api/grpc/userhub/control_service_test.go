package userhub

import (
	"context"
	"testing"

	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	"github.com/louisbranch/fracturing.space/internal/services/userhub/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestControlServiceInvalidateDashboardsRequiresTargets(t *testing.T) {
	t.Parallel()

	svc := NewControlService(&controlInvalidatorStub{})
	_, err := svc.InvalidateDashboards(context.Background(), &userhubv1.InvalidateDashboardsRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status.Code(err) = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestControlServiceInvalidateDashboardsForwardsRequest(t *testing.T) {
	t.Parallel()

	invalidator := controlInvalidatorStub{
		result: domain.InvalidateDashboardsResult{InvalidatedEntries: 3},
	}
	svc := NewControlService(&invalidator)

	resp, err := svc.InvalidateDashboards(context.Background(), &userhubv1.InvalidateDashboardsRequest{
		UserIds:     []string{"user-1"},
		CampaignIds: []string{"camp-1"},
		Reason:      "test",
	})
	if err != nil {
		t.Fatalf("InvalidateDashboards() error = %v", err)
	}
	if resp.GetInvalidatedEntries() != 3 {
		t.Fatalf("InvalidatedEntries = %d, want 3", resp.GetInvalidatedEntries())
	}
	if invalidator.input.Reason != "test" {
		t.Fatalf("Reason = %q, want %q", invalidator.input.Reason, "test")
	}
}

type controlInvalidatorStub struct {
	input  domain.InvalidateDashboardsInput
	result domain.InvalidateDashboardsResult
	err    error
}

func (s *controlInvalidatorStub) InvalidateDashboards(_ context.Context, input domain.InvalidateDashboardsInput) (domain.InvalidateDashboardsResult, error) {
	s.input = input
	return s.result, s.err
}
