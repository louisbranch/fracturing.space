package dashboard

import (
	"context"
	"errors"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
)

func TestLoadDashboardShowsPendingProfileBlock(t *testing.T) {
	t.Parallel()

	svc := newService(gatewayStub{snapshot: DashboardSnapshot{NeedsProfileCompletion: true}})

	view, err := svc.loadDashboard(context.Background(), "user-1", commonv1.Locale_LOCALE_EN_US)
	if err != nil {
		t.Fatalf("loadDashboard() error = %v", err)
	}
	if !view.ShowPendingProfileBlock {
		t.Fatalf("ShowPendingProfileBlock = false, want true")
	}
}

func TestLoadDashboardHidesPendingProfileBlockWhenComplete(t *testing.T) {
	t.Parallel()

	svc := newService(gatewayStub{snapshot: DashboardSnapshot{NeedsProfileCompletion: false}})

	view, err := svc.loadDashboard(context.Background(), "user-1", commonv1.Locale_LOCALE_EN_US)
	if err != nil {
		t.Fatalf("loadDashboard() error = %v", err)
	}
	if view.ShowPendingProfileBlock {
		t.Fatalf("ShowPendingProfileBlock = true, want false")
	}
}

func TestLoadDashboardHidesPendingProfileBlockWhenSocialDependencyIsDegraded(t *testing.T) {
	t.Parallel()

	svc := newService(gatewayStub{snapshot: DashboardSnapshot{
		NeedsProfileCompletion: true,
		DegradedDependencies:   []string{"social.profile"},
	}})

	view, err := svc.loadDashboard(context.Background(), "user-1", commonv1.Locale_LOCALE_EN_US)
	if err != nil {
		t.Fatalf("loadDashboard() error = %v", err)
	}
	if view.ShowPendingProfileBlock {
		t.Fatalf("ShowPendingProfileBlock = true, want false")
	}
}

func TestLoadDashboardReturnsEmptyViewWhenGatewayFails(t *testing.T) {
	t.Parallel()

	svc := newService(gatewayStub{err: errors.New("dependency unavailable")})

	view, err := svc.loadDashboard(context.Background(), "user-1", commonv1.Locale_LOCALE_EN_US)
	if err != nil {
		t.Fatalf("loadDashboard() error = %v", err)
	}
	if view.ShowPendingProfileBlock {
		t.Fatalf("ShowPendingProfileBlock = true, want false")
	}
}

func TestLoadDashboardSkipsGatewayWithoutUserID(t *testing.T) {
	t.Parallel()

	gateway := &gatewayRecorder{}
	svc := newService(gateway)

	view, err := svc.loadDashboard(context.Background(), "  ", commonv1.Locale_LOCALE_EN_US)
	if err != nil {
		t.Fatalf("loadDashboard() error = %v", err)
	}
	if view.ShowPendingProfileBlock {
		t.Fatalf("ShowPendingProfileBlock = true, want false")
	}
	if gateway.calls != 0 {
		t.Fatalf("gateway calls = %d, want 0", gateway.calls)
	}
}

type gatewayStub struct {
	snapshot DashboardSnapshot
	err      error
}

func (g gatewayStub) LoadDashboard(context.Context, string, commonv1.Locale) (DashboardSnapshot, error) {
	if g.err != nil {
		return DashboardSnapshot{}, g.err
	}
	return g.snapshot, nil
}

type gatewayRecorder struct {
	calls int
}

func (g *gatewayRecorder) LoadDashboard(context.Context, string, commonv1.Locale) (DashboardSnapshot, error) {
	g.calls++
	return DashboardSnapshot{}, nil
}
