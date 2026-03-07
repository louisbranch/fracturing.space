package app

import (
	"context"
	"errors"
	"net/http"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
)

type gatewayStub struct {
	snapshot DashboardSnapshot
	err      error
	calls    int
	lastUser string
}

func (g *gatewayStub) LoadDashboard(_ context.Context, userID string, _ language.Tag) (DashboardSnapshot, error) {
	g.calls++
	g.lastUser = userID
	if g.err != nil {
		return DashboardSnapshot{}, g.err
	}
	return g.snapshot, nil
}

func TestLoadDashboardSkipsBlankUserID(t *testing.T) {
	t.Parallel()

	gw := &gatewayStub{}
	svc := NewService(gw, nil, nil)
	view, err := svc.LoadDashboard(context.Background(), "   ", language.AmericanEnglish)
	if err != nil {
		t.Fatalf("LoadDashboard() error = %v", err)
	}
	if view.ShowPendingProfileBlock || view.ShowAdventureBlock {
		t.Fatalf("unexpected visible blocks: %+v", view)
	}
	if gw.calls != 0 {
		t.Fatalf("gateway calls = %d, want 0", gw.calls)
	}
}

func TestLoadDashboardNormalizesUserID(t *testing.T) {
	t.Parallel()

	gw := &gatewayStub{}
	svc := NewService(gw, nil, nil)
	if _, err := svc.LoadDashboard(context.Background(), " user-7 ", language.AmericanEnglish); err != nil {
		t.Fatalf("LoadDashboard() error = %v", err)
	}
	if gw.calls != 1 {
		t.Fatalf("gateway calls = %d, want 1", gw.calls)
	}
	if gw.lastUser != "user-7" {
		t.Fatalf("gateway user id = %q, want %q", gw.lastUser, "user-7")
	}
}

func TestUnavailableGatewayFailsClosed(t *testing.T) {
	t.Parallel()

	gateway := NewUnavailableGateway()
	if IsGatewayHealthy(nil) {
		t.Fatalf("IsGatewayHealthy(nil) = true, want false")
	}
	if IsGatewayHealthy(gateway) {
		t.Fatalf("IsGatewayHealthy(unavailable) = true, want false")
	}
	if !IsGatewayHealthy(&gatewayStub{}) {
		t.Fatalf("IsGatewayHealthy(stub) = false, want true")
	}

	view, err := gateway.LoadDashboard(context.Background(), "user-1", language.AmericanEnglish)
	if err == nil {
		t.Fatalf("LoadDashboard() error = nil, want unavailable error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("LoadDashboard() status = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if view.NeedsProfileCompletion || view.HasDraftOrActiveCampaign || view.CampaignsHasMore || len(view.DegradedDependencies) != 0 {
		t.Fatalf("LoadDashboard() view = %+v, want zero value", view)
	}
}

func TestLoadDashboardHandlesErrorsAndDegradedDependencies(t *testing.T) {
	t.Parallel()

	svc := NewService(&gatewayStub{err: errors.New("boom")}, nil, nil)
	view, err := svc.LoadDashboard(context.Background(), "user-1", language.AmericanEnglish)
	if err != nil {
		t.Fatalf("LoadDashboard() error = %v", err)
	}
	if view.ShowPendingProfileBlock || view.ShowAdventureBlock {
		t.Fatalf("unexpected visible blocks: %+v", view)
	}

	svc = NewService(&gatewayStub{snapshot: DashboardSnapshot{NeedsProfileCompletion: true, DegradedDependencies: []string{DegradedDependencySocialProfile}}}, nil, nil)
	view, err = svc.LoadDashboard(context.Background(), "user-1", language.AmericanEnglish)
	if err != nil {
		t.Fatalf("LoadDashboard() error = %v", err)
	}
	if view.ShowPendingProfileBlock {
		t.Fatalf("ShowPendingProfileBlock = true, want false")
	}
}

func TestLoadDashboardAdventureVisibility(t *testing.T) {
	t.Parallel()

	healthProvider := func(ctx context.Context) []ServiceHealthEntry {
		return []ServiceHealthEntry{{Label: "Campaigns", Available: true}}
	}
	svc := NewService(&gatewayStub{snapshot: DashboardSnapshot{}}, nil, healthProvider)
	view, err := svc.LoadDashboard(context.Background(), "user-1", language.AmericanEnglish)
	if err != nil {
		t.Fatalf("LoadDashboard() error = %v", err)
	}
	if !view.ShowAdventureBlock {
		t.Fatalf("ShowAdventureBlock = false, want true")
	}
	if len(view.ServiceHealth) != 1 || view.ServiceHealth[0].Label != "Campaigns" {
		t.Fatalf("ServiceHealth = %+v", view.ServiceHealth)
	}
}

func TestHasDegradedDependency(t *testing.T) {
	t.Parallel()

	if !HasDegradedDependency([]string{" social.profile "}, DegradedDependencySocialProfile) {
		t.Fatalf("expected degraded dependency to be found")
	}
	if HasDegradedDependency([]string{"game.campaigns"}, "") {
		t.Fatalf("expected empty lookup key to return false")
	}
}
