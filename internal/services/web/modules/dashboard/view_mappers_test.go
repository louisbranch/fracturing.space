package dashboard

import "testing"

func TestMapDashboardTemplateView(t *testing.T) {
	t.Parallel()

	view := mapDashboardTemplateView(DashboardView{
		ShowPendingProfileBlock: true,
		ShowAdventureBlock:      false,
		ServiceHealth: []ServiceHealthEntry{
			{Label: "game", Available: true},
			{Label: "status", Available: false},
		},
	})

	if !view.ProfilePending.Visible {
		t.Fatalf("ProfilePending.Visible = false, want true")
	}
	if view.Adventure.Visible {
		t.Fatalf("Adventure.Visible = true, want false")
	}
	if len(view.ServiceHealth) != 2 {
		t.Fatalf("len(ServiceHealth) = %d, want 2", len(view.ServiceHealth))
	}
	if view.ServiceHealth[0].Label != "game" || !view.ServiceHealth[0].Available {
		t.Fatalf("ServiceHealth[0] = %+v, want game/true", view.ServiceHealth[0])
	}
	if view.ServiceHealth[1].Label != "status" || view.ServiceHealth[1].Available {
		t.Fatalf("ServiceHealth[1] = %+v, want status/false", view.ServiceHealth[1])
	}
}
