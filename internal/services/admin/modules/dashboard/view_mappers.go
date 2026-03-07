package dashboard

import (
	"strconv"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/eventview"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"golang.org/x/text/message"
)

// buildDashboardStats maps gRPC statistics and system/user counts into a
// template-ready stats view.
func buildDashboardStats(stats *statev1.GameStatistics, systemCount, userCount int64) templates.DashboardStats {
	ds := templates.DashboardStats{
		TotalSystems:      "0",
		TotalCampaigns:    "0",
		TotalSessions:     "0",
		TotalCharacters:   "0",
		TotalParticipants: "0",
		TotalUsers:        "0",
	}
	if stats != nil {
		ds.TotalCampaigns = strconv.FormatInt(stats.GetCampaignCount(), 10)
		ds.TotalSessions = strconv.FormatInt(stats.GetSessionCount(), 10)
		ds.TotalCharacters = strconv.FormatInt(stats.GetCharacterCount(), 10)
		ds.TotalParticipants = strconv.FormatInt(stats.GetParticipantCount(), 10)
	}
	if systemCount >= 0 {
		ds.TotalSystems = strconv.FormatInt(systemCount, 10)
	}
	if userCount >= 0 {
		ds.TotalUsers = strconv.FormatInt(userCount, 10)
	}
	return ds
}

// buildActivityEvents maps activity records into template-ready event views.
func buildActivityEvents(records []activityRecord, loc *message.Printer) []templates.ActivityEvent {
	events := make([]templates.ActivityEvent, 0, len(records))
	for _, record := range records {
		evt := record.event
		events = append(events, templates.ActivityEvent{
			CampaignID:   evt.GetCampaignId(),
			CampaignName: record.campaignName,
			EventType:    eventview.FormatEventType(evt.GetType(), loc),
			Timestamp:    eventview.FormatTimestamp(evt.GetTs()),
			Description:  eventview.FormatEventDescription(evt, loc),
		})
	}
	return events
}
