package discovery

import (
	discoveryapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery/app"
)

// DiscoveryPageView keeps discovery transport state explicit for the page template.
type DiscoveryPageView struct {
	Status      string
	Unavailable bool
	Entries     []StarterEntryView
}

// mapPageToView converts the app-layer page contract into template state.
func mapPageToView(page discoveryapp.Page) DiscoveryPageView {
	return DiscoveryPageView{
		Status:      string(page.Status),
		Unavailable: page.Status == discoveryapp.PageStatusUnavailable,
		Entries:     mapEntriesToView(page.Entries),
	}
}

// mapEntriesToView converts gateway domain types to template view types.
func mapEntriesToView(entries []discoveryapp.StarterEntry) []StarterEntryView {
	if len(entries) == 0 {
		return nil
	}
	views := make([]StarterEntryView, len(entries))
	for i, entry := range entries {
		views[i] = StarterEntryView{
			EntryID:     entry.EntryID,
			Title:       entry.Title,
			Description: entry.Description,
			Tags:        entry.Tags,
			Difficulty:  entry.Difficulty,
			Duration:    entry.Duration,
			GmMode:      entry.GmMode,
			System:      entry.System,
			Level:       entry.Level,
			Players:     entry.Players,
		}
	}
	return views
}
