package templates

// ScenarioPageView provides data for the scenario runner page.
type ScenarioPageView struct {
	// Script is the Lua script content.
	Script string
	// Logs is the latest run log output.
	Logs string
	// CampaignID is the sandbox campaign id created by the run.
	CampaignID string
	// CampaignName is the display name for the sandbox campaign.
	CampaignName string
	// EventsURL is the link to the scenario events tab.
	EventsURL string
	// HasRun reports whether a scenario run has been attempted.
	HasRun bool
	// Status is the outcome label for the latest run.
	Status string
	// StatusBadge is the badge variant for the latest run.
	StatusBadge string
}

// ScenarioEventsView provides data for scenario event listings.
type ScenarioEventsView struct {
	// CampaignID is the sandbox campaign id.
	CampaignID string
	// CampaignName is the display name for the sandbox campaign.
	CampaignName string
	// Events holds formatted event rows.
	Events []EventRow
	// Filters holds the current event filter state.
	Filters EventFilterOptions
	// TotalCount is the total number of events.
	TotalCount int32
	// NextToken is the pagination token for next page.
	NextToken string
	// PrevToken is the pagination token for previous page.
	PrevToken string
	// Message is an optional empty/error message.
	Message string
}
