package dashboard

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"google.golang.org/grpc"
)

type fakeStatusClient struct {
	resp         *statusv1.GetSystemStatusResponse
	err          error
	deadlineSeen bool
}

func (f *fakeStatusClient) ReportStatus(context.Context, *statusv1.ReportStatusRequest, ...grpc.CallOption) (*statusv1.ReportStatusResponse, error) {
	return &statusv1.ReportStatusResponse{}, nil
}

func (f *fakeStatusClient) GetSystemStatus(ctx context.Context, _ *statusv1.GetSystemStatusRequest, _ ...grpc.CallOption) (*statusv1.GetSystemStatusResponse, error) {
	_, f.deadlineSeen = ctx.Deadline()
	return f.resp, f.err
}

func (f *fakeStatusClient) SetOverride(context.Context, *statusv1.SetOverrideRequest, ...grpc.CallOption) (*statusv1.SetOverrideResponse, error) {
	return &statusv1.SetOverrideResponse{}, nil
}

func (f *fakeStatusClient) ClearOverride(context.Context, *statusv1.ClearOverrideRequest, ...grpc.CallOption) (*statusv1.ClearOverrideResponse, error) {
	return &statusv1.ClearOverrideResponse{}, nil
}

type fakeUserHubClient struct{}

func (fakeUserHubClient) GetDashboard(context.Context, *userhubv1.GetDashboardRequest, ...grpc.CallOption) (*userhubv1.GetDashboardResponse, error) {
	return &userhubv1.GetDashboardResponse{}, nil
}

func TestStatusHealthProviderNilClientReturnsNil(t *testing.T) {
	t.Parallel()

	if provider := StatusHealthProvider(nil, nil); provider != nil {
		t.Fatalf("provider = %v, want nil", provider)
	}
}

func TestStatusHealthProviderMapsAndSortsEntries(t *testing.T) {
	t.Parallel()

	client := &fakeStatusClient{
		resp: &statusv1.GetSystemStatusResponse{
			Services: []*statusv1.ServiceStatus{
				{Service: "worker", AggregateStatus: statusv1.CapabilityStatus_CAPABILITY_STATUS_DEGRADED},
				nil,
				{Service: "auth", AggregateStatus: statusv1.CapabilityStatus_CAPABILITY_STATUS_OPERATIONAL},
			},
		},
	}

	entries := StatusHealthProvider(client, slog.Default())(context.Background())
	if !client.deadlineSeen {
		t.Fatal("status client did not observe timeout context")
	}
	if len(entries) != 2 {
		t.Fatalf("entries len = %d, want 2", len(entries))
	}
	if entries[0].Label != "Auth" || !entries[0].Available {
		t.Fatalf("entries[0] = %+v", entries[0])
	}
	if entries[1].Label != "Worker" || entries[1].Available {
		t.Fatalf("entries[1] = %+v", entries[1])
	}
}

func TestStatusHealthProviderReturnsNilOnErrorOrEmptyResponse(t *testing.T) {
	t.Parallel()

	if entries := StatusHealthProvider(&fakeStatusClient{err: errors.New("boom")}, nil)(context.Background()); entries != nil {
		t.Fatalf("entries on error = %+v, want nil", entries)
	}
	if entries := StatusHealthProvider(&fakeStatusClient{resp: &statusv1.GetSystemStatusResponse{}}, nil)(context.Background()); entries != nil {
		t.Fatalf("entries on empty response = %+v, want nil", entries)
	}
}

func TestComposeBuildsDashboardModule(t *testing.T) {
	t.Parallel()

	module := Compose(fakeUserHubClient{}, &fakeStatusClient{}, modulehandler.NewBase(nil, nil, nil), slog.Default())
	mount, err := module.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	if mount.Prefix == "" || mount.Handler == nil {
		t.Fatalf("mount = %+v", mount)
	}
}

func TestCapitalizeService(t *testing.T) {
	t.Parallel()

	if got := capitalizeService(""); got != "" {
		t.Fatalf("capitalizeService(empty) = %q, want empty", got)
	}
	if got := capitalizeService("status"); got != "Status" {
		t.Fatalf("capitalizeService(status) = %q, want Status", got)
	}
}

func TestStatusHealthTimeoutIsReasonable(t *testing.T) {
	t.Parallel()

	if statusHealthTimeout < time.Second {
		t.Fatalf("statusHealthTimeout = %s, want >= 1s", statusHealthTimeout)
	}
}
