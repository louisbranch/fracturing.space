package app

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/text/language"
)

type gatewayStub struct {
	snapshot DashboardSnapshot
	err      error
	calls    int
}

func (g *gatewayStub) LoadDashboard(context.Context, string, language.Tag) (DashboardSnapshot, error) {
	g.calls++
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

	svc := NewService(&gatewayStub{snapshot: DashboardSnapshot{}}, nil, []ServiceHealthEntry{{Label: "Campaigns", Available: true}})
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
