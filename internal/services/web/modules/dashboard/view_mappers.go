package dashboard

import webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"

// mapDashboardTemplateView maps values across transport and domain boundaries.
func mapDashboardTemplateView(view DashboardView) webtemplates.DashboardPageView {
	health := make([]webtemplates.DashboardServiceHealthEntry, len(view.ServiceHealth))
	for i, e := range view.ServiceHealth {
		health[i] = webtemplates.DashboardServiceHealthEntry{Label: e.Label, Available: e.Available}
	}
	activeSessions := make([]webtemplates.DashboardActiveSessionEntry, len(view.ActiveSessions))
	for i, session := range view.ActiveSessions {
		activeSessions[i] = webtemplates.DashboardActiveSessionEntry{
			CampaignID:   session.CampaignID,
			CampaignName: session.CampaignName,
			SessionID:    session.SessionID,
			SessionName:  session.SessionName,
		}
	}
	return webtemplates.DashboardPageView{
		ProfilePending: webtemplates.DashboardProfilePendingBlock{Visible: view.ShowPendingProfileBlock},
		Adventure:      webtemplates.DashboardAdventureBlock{Visible: view.ShowAdventureBlock},
		ActiveSessions: webtemplates.DashboardActiveSessionsBlock{
			Visible:  len(activeSessions) > 0,
			Sessions: activeSessions,
		},
		ServiceHealth: health,
	}
}
