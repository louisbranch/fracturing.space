package discovery

import (
	discoveryapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery/app"
)

// mapEntriesToView converts gateway domain types to template view types.
func mapEntriesToView(entries []discoveryapp.StarterEntry) []StarterEntryView {
	if len(entries) == 0 {
		return nil
	}
	views := make([]StarterEntryView, len(entries))
	for i, entry := range entries {
		views[i] = StarterEntryView{
			CampaignID:  entry.CampaignID,
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
