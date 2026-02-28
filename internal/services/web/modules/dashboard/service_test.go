package dashboard

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/text/language"
)

func TestLoadDashboardShowsPendingProfileBlock(t *testing.T) {
	t.Parallel()

	svc := newService(&fakeGateway{snapshot: DashboardSnapshot{NeedsProfileCompletion: true}}, nil, nil)

	view, err := svc.loadDashboard(context.Background(), "user-1", language.AmericanEnglish)
	if err != nil {
		t.Fatalf("loadDashboard() error = %v", err)
	}
	if !view.ShowPendingProfileBlock {
		t.Fatalf("ShowPendingProfileBlock = false, want true")
	}
}

func TestLoadDashboardHidesPendingProfileBlockWhenComplete(t *testing.T) {
	t.Parallel()

	svc := newService(&fakeGateway{snapshot: DashboardSnapshot{NeedsProfileCompletion: false}}, nil, nil)

	view, err := svc.loadDashboard(context.Background(), "user-1", language.AmericanEnglish)
	if err != nil {
		t.Fatalf("loadDashboard() error = %v", err)
	}
	if view.ShowPendingProfileBlock {
		t.Fatalf("ShowPendingProfileBlock = true, want false")
	}
}

func TestLoadDashboardHidesPendingProfileBlockWhenSocialDependencyIsDegraded(t *testing.T) {
	t.Parallel()

	svc := newService(&fakeGateway{snapshot: DashboardSnapshot{
		NeedsProfileCompletion: true,
		DegradedDependencies:   []string{"social.profile"},
	}}, nil, nil)

	view, err := svc.loadDashboard(context.Background(), "user-1", language.AmericanEnglish)
	if err != nil {
		t.Fatalf("loadDashboard() error = %v", err)
	}
	if view.ShowPendingProfileBlock {
		t.Fatalf("ShowPendingProfileBlock = true, want false")
	}
}

func TestLoadDashboardReturnsEmptyViewWhenGatewayFails(t *testing.T) {
	t.Parallel()

	svc := newService(&fakeGateway{err: errors.New("dependency unavailable")}, nil, nil)

	view, err := svc.loadDashboard(context.Background(), "user-1", language.AmericanEnglish)
	if err != nil {
		t.Fatalf("loadDashboard() error = %v", err)
	}
	if view.ShowPendingProfileBlock {
		t.Fatalf("ShowPendingProfileBlock = true, want false")
	}
	if view.ShowAdventureBlock {
		t.Fatalf("ShowAdventureBlock = true, want false")
	}
}

func TestLoadDashboardSkipsGatewayWithoutUserID(t *testing.T) {
	t.Parallel()

	gateway := &fakeGateway{}
	svc := newService(gateway, nil, nil)

	view, err := svc.loadDashboard(context.Background(), "  ", language.AmericanEnglish)
	if err != nil {
		t.Fatalf("loadDashboard() error = %v", err)
	}
	if view.ShowPendingProfileBlock {
		t.Fatalf("ShowPendingProfileBlock = true, want false")
	}
	if view.ShowAdventureBlock {
		t.Fatalf("ShowAdventureBlock = true, want false")
	}
	if gateway.calls != 0 {
		t.Fatalf("gateway calls = %d, want 0", gateway.calls)
	}
}

func TestLoadDashboardShowsAdventureBlockWhenNoDraftOrActiveCampaignExists(t *testing.T) {
	t.Parallel()

	svc := newService(&fakeGateway{snapshot: DashboardSnapshot{
		HasDraftOrActiveCampaign: false,
		CampaignsHasMore:         false,
	}}, nil, nil)

	view, err := svc.loadDashboard(context.Background(), "user-1", language.AmericanEnglish)
	if err != nil {
		t.Fatalf("loadDashboard() error = %v", err)
	}
	if !view.ShowAdventureBlock {
		t.Fatalf("ShowAdventureBlock = false, want true")
	}
}

func TestLoadDashboardHidesAdventureBlockWhenDraftOrActiveCampaignExists(t *testing.T) {
	t.Parallel()

	svc := newService(&fakeGateway{snapshot: DashboardSnapshot{HasDraftOrActiveCampaign: true}}, nil, nil)

	view, err := svc.loadDashboard(context.Background(), "user-1", language.AmericanEnglish)
	if err != nil {
		t.Fatalf("loadDashboard() error = %v", err)
	}
	if view.ShowAdventureBlock {
		t.Fatalf("ShowAdventureBlock = true, want false")
	}
}

func TestLoadDashboardHidesAdventureBlockWhenCampaignStateIsTruncated(t *testing.T) {
	t.Parallel()

	svc := newService(&fakeGateway{snapshot: DashboardSnapshot{
		HasDraftOrActiveCampaign: false,
		CampaignsHasMore:         true,
	}}, nil, nil)

	view, err := svc.loadDashboard(context.Background(), "user-1", language.AmericanEnglish)
	if err != nil {
		t.Fatalf("loadDashboard() error = %v", err)
	}
	if view.ShowAdventureBlock {
		t.Fatalf("ShowAdventureBlock = true, want false")
	}
}

func TestLoadDashboardHidesAdventureBlockWhenCampaignDependencyIsDegraded(t *testing.T) {
	t.Parallel()

	svc := newService(&fakeGateway{snapshot: DashboardSnapshot{
		HasDraftOrActiveCampaign: false,
		CampaignsHasMore:         false,
		DegradedDependencies:     []string{"game.campaigns"},
	}}, nil, nil)

	view, err := svc.loadDashboard(context.Background(), "user-1", language.AmericanEnglish)
	if err != nil {
		t.Fatalf("loadDashboard() error = %v", err)
	}
	if view.ShowAdventureBlock {
		t.Fatalf("ShowAdventureBlock = true, want false")
	}
}

func TestLoadDashboardIncludesServiceHealthEntries(t *testing.T) {
	t.Parallel()

	health := []ServiceHealthEntry{
		{Label: "Campaigns", Available: true},
		{Label: "Dashboard", Available: false},
	}
	svc := newService(&fakeGateway{snapshot: DashboardSnapshot{}}, nil, health)

	view, err := svc.loadDashboard(context.Background(), "user-1", language.AmericanEnglish)
	if err != nil {
		t.Fatalf("loadDashboard() error = %v", err)
	}
	if len(view.ServiceHealth) != 2 {
		t.Fatalf("ServiceHealth count = %d, want 2", len(view.ServiceHealth))
	}
	if view.ServiceHealth[0].Label != "Campaigns" || !view.ServiceHealth[0].Available {
		t.Fatalf("ServiceHealth[0] = %+v, want {Campaigns, true}", view.ServiceHealth[0])
	}
	if view.ServiceHealth[1].Label != "Dashboard" || view.ServiceHealth[1].Available {
		t.Fatalf("ServiceHealth[1] = %+v, want {Dashboard, false}", view.ServiceHealth[1])
	}
}

func TestLoadDashboardReturnsNilServiceHealthWhenNotSet(t *testing.T) {
	t.Parallel()

	svc := newService(&fakeGateway{snapshot: DashboardSnapshot{}}, nil, nil)

	view, err := svc.loadDashboard(context.Background(), "user-1", language.AmericanEnglish)
	if err != nil {
		t.Fatalf("loadDashboard() error = %v", err)
	}
	if view.ServiceHealth != nil {
		t.Fatalf("ServiceHealth = %v, want nil", view.ServiceHealth)
	}
}
