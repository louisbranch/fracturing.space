package dashboard

import (
	"testing"

	dashboardapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/app"
)

func TestMapDashboardTemplateView(t *testing.T) {
	t.Parallel()

	view := mapDashboardTemplateView(dashboardapp.DashboardView{
		DataStatus:              dashboardapp.DashboardDataStatusDegraded,
		DegradedDependencies:    []string{dashboardapp.DegradedDependencySocialProfile},
		ShowPendingProfileBlock: true,
		ShowAdventureBlock:      false,
		CampaignStartNudges: []dashboardapp.CampaignStartNudgeItem{
			{CampaignID: "camp-2", CampaignName: "Moonwake", BlockerMessage: "Finish Aria", ActionKind: dashboardapp.CampaignStartNudgeActionKindCompleteCharacter, TargetCharacterID: "char-1"},
		},
		CampaignStartNudgesMore: true,
		ActiveSessions: []dashboardapp.ActiveSessionItem{
			{CampaignID: "camp-1", CampaignName: "Sunfall", SessionID: "session-1", SessionName: "The Crossing"},
		},
		ServiceHealth: []dashboardapp.ServiceHealthEntry{
			{Label: "game", Available: true},
			{Label: "status", Available: false},
		},
	})

	if !view.ProfilePending.Visible {
		t.Fatalf("ProfilePending.Visible = false, want true")
	}
	if !view.StatusNotice.Visible || !view.StatusNotice.Degraded {
		t.Fatalf("StatusNotice = %+v, want visible degraded notice", view.StatusNotice)
	}
	if view.Adventure.Visible {
		t.Fatalf("Adventure.Visible = true, want false")
	}
	if !view.CampaignStartNudges.Visible || len(view.CampaignStartNudges.Nudges) != 1 {
		t.Fatalf("CampaignStartNudges = %+v, want one visible nudge", view.CampaignStartNudges)
	}
	if view.CampaignStartNudges.Nudges[0].TargetCharacterID != "char-1" {
		t.Fatalf("CampaignStartNudges[0].TargetCharacterID = %q, want %q", view.CampaignStartNudges.Nudges[0].TargetCharacterID, "char-1")
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
