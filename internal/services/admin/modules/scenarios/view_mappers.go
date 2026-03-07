package scenarios

import (
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/eventview"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"golang.org/x/text/message"
)

func buildScenarioTimelineEntries(entries []*statev1.TimelineEntry, loc *message.Printer) []templates.ScenarioTimelineEntry {
	rows := make([]templates.ScenarioTimelineEntry, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		projection := entry.GetProjection()
		title := strings.TrimSpace(projection.GetTitle())
		eventTypeDisplay := eventview.FormatEventType(entry.GetEventType(), loc)
		if title == "" {
			title = eventTypeDisplay
		}
		subtitle := strings.TrimSpace(projection.GetSubtitle())
		status := strings.TrimSpace(projection.GetStatus())
		iconID := entry.GetIconId()
		if iconID == commonv1.IconId_ICON_ID_UNSPECIFIED {
			iconID = commonv1.IconId_ICON_ID_GENERIC
		}
		rows = append(rows, templates.ScenarioTimelineEntry{
			Seq:              entry.GetSeq(),
			EventType:        entry.GetEventType(),
			EventTypeDisplay: eventTypeDisplay,
			EventTime:        eventview.FormatTimestamp(entry.GetEventTime()),
			IconID:           iconID,
			Title:            title,
			Subtitle:         subtitle,
			Status:           status,
			StatusBadge:      timelineStatusBadgeVariant(status),
			Fields:           buildScenarioTimelineFields(projection.GetFields()),
			PayloadJSON:      strings.TrimSpace(entry.GetEventPayloadJson()),
		})
	}
	return rows
}

func buildScenarioTimelineFields(fields []*statev1.ProjectionField) []templates.ScenarioTimelineField {
	if len(fields) == 0 {
		return nil
	}
	result := make([]templates.ScenarioTimelineField, 0, len(fields))
	for _, field := range fields {
		if field == nil {
			continue
		}
		label := strings.TrimSpace(field.GetLabel())
		value := strings.TrimSpace(field.GetValue())
		if label == "" && value == "" {
			continue
		}
		result = append(result, templates.ScenarioTimelineField{
			Label: label,
			Value: value,
		})
	}
	return result
}

func timelineStatusBadgeVariant(status string) string {
	if status == "" {
		return "secondary"
	}
	normalized := strings.ToUpper(strings.TrimSpace(status))
	switch normalized {
	case "ACTIVE":
		return "success"
	case "DRAFT":
		return "warning"
	case "COMPLETED":
		return "success"
	case "ARCHIVED", "ENDED":
		return "neutral"
	default:
		return "secondary"
	}
}
