package discovery

import webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"

// mapEntriesToView converts gateway domain types to template view types.
func mapEntriesToView(entries []StarterEntry) []webtemplates.StarterEntryView {
	if len(entries) == 0 {
		return nil
	}
	views := make([]webtemplates.StarterEntryView, len(entries))
	for i, entry := range entries {
		views[i] = webtemplates.StarterEntryView{
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
