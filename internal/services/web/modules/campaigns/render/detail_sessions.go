package render

import "github.com/a-h/templ"

// SessionsPageView carries session-list page state only.
type SessionsPageView struct {
	CampaignDetailBaseView
	Sessions         []SessionView
	SessionReadiness SessionReadinessView
}

// SessionDetailPageView carries session-detail page state only.
type SessionDetailPageView struct {
	CampaignDetailBaseView
	SessionID string
	Sessions  []SessionView
}

// SessionsFragment renders the session-list page.
func SessionsFragment(view SessionsPageView, loc Localizer) templ.Component {
	return sessionsFragment(view, loc)
}

// SessionDetailFragment renders the selected session page.
func SessionDetailFragment(view SessionDetailPageView, loc Localizer) templ.Component {
	return sessionDetailFragment(view, loc)
}

// SessionView carries session rows for campaign detail pages.
type SessionView struct {
	ID        string
	Name      string
	Status    string
	StartedAt string
	UpdatedAt string
	EndedAt   string
}

// SessionReadinessBlockerView preserves readiness blocker copy for the session
// start affordance.
type SessionReadinessBlockerView struct {
	Code    string
	Message string
}

// SessionReadinessView carries session start state for campaigns detail pages.
type SessionReadinessView struct {
	Ready    bool
	Blockers []SessionReadinessBlockerView
}
