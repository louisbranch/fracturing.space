package dashboard

import "testing"

func TestMapDashboardTemplateView(t *testing.T) {
	t.Parallel()

	view := mapDashboardTemplateView(DashboardView{
		ShowPendingProfileBlock: true,
		ShowAdventureBlock:      false,
		ActiveSessions: []ActiveSessionItem{
			{CampaignID: "camp-1", CampaignName: "Sunfall", SessionID: "session-1", SessionName: "The Crossing"},
		},
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
	if !view.ActiveSessions.Visible || len(view.ActiveSessions.Sessions) != 1 {
		t.Fatalf("ActiveSessions = %+v, want one visible session", view.ActiveSessions)
	}
	if view.ActiveSessions.Sessions[0].CampaignID != "camp-1" {
		t.Fatalf("ActiveSessions[0].CampaignID = %q, want %q", view.ActiveSessions.Sessions[0].CampaignID, "camp-1")
	}
	if view.ActiveSessions.Sessions[0].SessionID != "session-1" {
		t.Fatalf("ActiveSessions[0].SessionID = %q, want %q", view.ActiveSessions.Sessions[0].SessionID, "session-1")
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
