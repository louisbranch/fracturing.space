package admin

import (
	"log"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	routepath "github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"golang.org/x/text/message"
)

func (h *Handler) handleSessionDetail(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
	loc, lang := h.localizer(w, r)
	sessionClient := h.sessionClient()
	if sessionClient == nil {
		http.Error(w, loc.Sprintf("error.session_service_unavailable"), http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()

	// Get session details
	response, err := sessionClient.GetSession(ctx, &statev1.GetSessionRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	})
	if err != nil {
		log.Printf("get session: %v", err)
		http.Error(w, loc.Sprintf("error.session_unavailable"), http.StatusNotFound)
		return
	}

	session := response.GetSession()
	if session == nil {
		http.Error(w, loc.Sprintf("error.session_not_found"), http.StatusNotFound)
		return
	}

	campaignName := getCampaignName(h, r, campaignID, loc)

	// Get event count for this session
	var eventCount int32
	if eventClient := h.eventClient(); eventClient != nil {
		eventsResp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   1,
			Filter:     "session_id = \"" + sessionID + "\"",
		})
		if err == nil && eventsResp != nil {
			eventCount = eventsResp.GetTotalSize()
		}
	}

	detail := buildSessionDetail(campaignID, campaignName, session, eventCount, loc)
	pageCtx := h.pageContext(lang, loc, r)
	renderPage(
		w,
		r,
		templates.SessionDetailPage(detail, loc),
		templates.SessionDetailFullPage(detail, pageCtx),
		htmxLocalizedPageTitle(loc, "title.session", detail.Name, templates.AppName()),
	)
}

// handleSessionEvents renders the session events via HTMX.
func (h *Handler) handleSessionEvents(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
	loc, _ := h.localizer(w, r)
	eventClient := h.eventClient()
	if eventClient == nil {
		templ.Handler(templates.EmptyState(loc.Sprintf("error.event_service_unavailable"))).ServeHTTP(w, r)
		return
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()

	pageToken := r.URL.Query().Get("page_token")

	// Get events for this session
	eventsResp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   eventListPageSize,
		PageToken:  pageToken,
		OrderBy:    "seq desc",
		Filter:     "session_id = \"" + sessionID + "\"",
	})
	if err != nil {
		log.Printf("list session events: %v", err)
		templ.Handler(templates.EmptyState(loc.Sprintf("error.events_unavailable"))).ServeHTTP(w, r)
		return
	}

	campaignName := getCampaignName(h, r, campaignID, loc)
	sessionName := getSessionName(h, r, campaignID, sessionID, loc)

	events := buildEventRows(eventsResp.GetEvents(), loc)
	detail := templates.SessionDetail{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		ID:           sessionID,
		Name:         sessionName,
		Events:       events,
		EventCount:   eventsResp.GetTotalSize(),
		NextToken:    eventsResp.GetNextPageToken(),
		PrevToken:    eventsResp.GetPreviousPageToken(),
	}

	templ.Handler(templates.SessionEventsContent(detail, loc)).ServeHTTP(w, r)
}

// handleEventLog renders the event log page.
func (h *Handler) handleEventLog(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := h.localizer(w, r)
	campaignName := getCampaignName(h, r, campaignID, loc)
	filters := parseEventFilters(r)

	// Fetch events for initial load
	var events []templates.EventRow
	var totalCount int32
	var nextToken, prevToken string

	if eventClient := h.eventClient(); eventClient != nil {
		ctx, cancel := h.gameGRPCCallContext(r.Context())
		defer cancel()

		filterExpr := buildEventFilterExpression(filters)

		eventsResp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   eventListPageSize,
			OrderBy:    "seq desc",
			Filter:     filterExpr,
		})
		if err == nil && eventsResp != nil {
			events = buildEventRows(eventsResp.GetEvents(), loc)
			totalCount = eventsResp.GetTotalSize()
			nextToken = eventsResp.GetNextPageToken()
			prevToken = eventsResp.GetPreviousPageToken()
		}
	}

	view := templates.EventLogView{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		Events:       events,
		Filters:      filters,
		TotalCount:   totalCount,
		NextToken:    nextToken,
		PrevToken:    prevToken,
	}
	pageCtx := h.pageContext(lang, loc, r)
	renderPage(
		w,
		r,
		templates.EventLogPage(view, loc),
		templates.EventLogFullPage(view, pageCtx),
		htmxLocalizedPageTitle(loc, "title.events", templates.AppName()),
	)
}

// handleEventLogTable renders the event log table via HTMX.
func (h *Handler) handleEventLogTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := h.localizer(w, r)
	eventClient := h.eventClient()
	if eventClient == nil {
		templ.Handler(templates.EmptyState(loc.Sprintf("error.event_service_unavailable"))).ServeHTTP(w, r)
		return
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()

	filters := parseEventFilters(r)
	filterExpr := buildEventFilterExpression(filters)
	pageToken := r.URL.Query().Get("page_token")

	eventsResp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   eventListPageSize,
		PageToken:  pageToken,
		OrderBy:    "seq desc",
		Filter:     filterExpr,
	})
	if err != nil {
		log.Printf("list events: %v", err)
		templ.Handler(templates.EmptyState(loc.Sprintf("error.events_unavailable"))).ServeHTTP(w, r)
		return
	}

	campaignName := getCampaignName(h, r, campaignID, loc)
	events := buildEventRows(eventsResp.GetEvents(), loc)

	view := templates.EventLogView{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		Events:       events,
		Filters:      filters,
		NextToken:    eventsResp.GetNextPageToken(),
		PrevToken:    eventsResp.GetPreviousPageToken(),
		TotalCount:   eventsResp.GetTotalSize(),
	}

	if pushURL := eventFilterPushURL(routepath.CampaignEvents(campaignID), filters, pageToken); pushURL != "" {
		w.Header().Set("HX-Push-Url", pushURL)
	}

	templ.Handler(templates.EventLogTableContent(view, loc)).ServeHTTP(w, r)
}

// buildSessionDetail formats a session into detail view data.
func buildSessionDetail(campaignID, campaignName string, session *statev1.Session, eventCount int32, loc *message.Printer) templates.SessionDetail {
	if session == nil {
		return templates.SessionDetail{}
	}

	status := formatSessionStatus(session.GetStatus(), loc)
	statusBadge := "secondary"
	if session.GetStatus() == statev1.SessionStatus_SESSION_ACTIVE {
		statusBadge = "success"
	}

	detail := templates.SessionDetail{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		ID:           session.GetId(),
		Name:         session.GetName(),
		Status:       status,
		StatusBadge:  statusBadge,
		StartedAt:    formatTimestamp(session.GetStartedAt()),
		EventCount:   eventCount,
	}

	if session.GetEndedAt() != nil {
		detail.EndedAt = formatTimestamp(session.GetEndedAt())
	}

	return detail
}

// buildEventRows formats events for display.
func buildEventRows(events []*statev1.Event, loc *message.Printer) []templates.EventRow {
	rows := make([]templates.EventRow, 0, len(events))
	for _, event := range events {
		if event == nil {
			continue
		}
		rows = append(rows, templates.EventRow{
			CampaignID:       event.GetCampaignId(),
			Seq:              event.GetSeq(),
			Hash:             event.GetHash(),
			Type:             event.GetType(),
			TypeDisplay:      formatEventType(event.GetType(), loc),
			Timestamp:        formatTimestamp(event.GetTs()),
			SessionID:        event.GetSessionId(),
			ActorType:        event.GetActorType(),
			ActorTypeDisplay: formatActorType(event.GetActorType(), loc),
			ActorName:        "",
			EntityType:       event.GetEntityType(),
			EntityID:         event.GetEntityId(),
			EntityName:       event.GetEntityId(),
			Description:      formatEventDescription(event, loc),
			PayloadJSON:      string(event.GetPayloadJson()),
		})
	}
	return rows
}

func buildScenarioTimelineEntries(entries []*statev1.TimelineEntry, loc *message.Printer) []templates.ScenarioTimelineEntry {
	rows := make([]templates.ScenarioTimelineEntry, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		projection := entry.GetProjection()
		title := strings.TrimSpace(projection.GetTitle())
		eventTypeDisplay := formatEventType(entry.GetEventType(), loc)
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
			EventTime:        formatTimestamp(entry.GetEventTime()),
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

// parseEventFilters extracts filter parameters from the request.
func parseEventFilters(r *http.Request) templates.EventFilterOptions {
	return templates.EventFilterOptions{
		SessionID:  r.URL.Query().Get("session_id"),
		EventType:  r.URL.Query().Get("event_type"),
		ActorType:  r.URL.Query().Get("actor_type"),
		EntityType: r.URL.Query().Get("entity_type"),
		StartDate:  r.URL.Query().Get("start_date"),
		EndDate:    r.URL.Query().Get("end_date"),
	}
}

func eventFilterPushURL(basePath string, filters templates.EventFilterOptions, pageToken string) string {
	pushURL := templates.EventFilterBaseURL(basePath, filters)
	if pageToken != "" {
		return templates.AppendQueryParam(pushURL, "page_token", pageToken)
	}
	return pushURL
}

// escapeAIP160StringLiteral escapes special characters for AIP-160 string literals.
// Backslashes and double quotes must be escaped to prevent injection.
func escapeAIP160StringLiteral(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// buildEventFilterExpression creates an AIP-160 filter expression from options.
func buildEventFilterExpression(filters templates.EventFilterOptions) string {
	var parts []string

	if filters.SessionID != "" {
		parts = append(parts, "session_id = \""+escapeAIP160StringLiteral(filters.SessionID)+"\"")
	}
	if filters.EventType != "" {
		parts = append(parts, "type = \""+escapeAIP160StringLiteral(filters.EventType)+"\"")
	}
	if filters.ActorType != "" {
		parts = append(parts, "actor_type = \""+escapeAIP160StringLiteral(filters.ActorType)+"\"")
	}
	if filters.EntityType != "" {
		parts = append(parts, "entity_type = \""+escapeAIP160StringLiteral(filters.EntityType)+"\"")
	}
	if filters.StartDate != "" {
		parts = append(parts, "ts >= timestamp(\""+escapeAIP160StringLiteral(filters.StartDate)+"T00:00:00Z\")")
	}
	if filters.EndDate != "" {
		parts = append(parts, "ts <= timestamp(\""+escapeAIP160StringLiteral(filters.EndDate)+"T23:59:59Z\")")
	}

	return strings.Join(parts, " AND ")
}

// getSessionName fetches the session name by ID.
func getSessionName(h *Handler, r *http.Request, campaignID, sessionID string, loc *message.Printer) string {
	sessionClient := h.sessionClient()
	if sessionClient == nil {
		return loc.Sprintf("label.session")
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()

	response, err := sessionClient.GetSession(ctx, &statev1.GetSessionRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	})
	if err != nil || response == nil || response.GetSession() == nil {
		return loc.Sprintf("label.session")
	}

	return response.GetSession().GetName()
}
