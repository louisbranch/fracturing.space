package scenarios

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/a-h/templ"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
	routepath "github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"github.com/louisbranch/fracturing.space/internal/tools/scenario"
	"golang.org/x/text/message"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// eventListPageSize caps the number of events shown per page.
	eventListPageSize = 50
	// maxScenarioScriptSize caps scenario scripts to limit resource usage.
	maxScenarioScriptSize = 100 * 1024
	// scenarioTempDirEnv configures the temp directory for scenario scripts.
	scenarioTempDirEnv = "FRACTURING_SPACE_SCENARIO_TMPDIR"
)

// service provides module-local scenario handlers backed by shared module dependencies.
type service struct {
	base     modulehandler.Base
	grpcAddr string
}

// NewService returns the scenarios module service implementation.
func NewService(base modulehandler.Base, grpcAddr string) Service {
	return &service{
		base:     base,
		grpcAddr: strings.TrimSpace(grpcAddr),
	}
}

// HandleScenarios renders the scenario page and handles scenario run submissions.
func (s *service) HandleScenarios(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		s.handleScenarioRun(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", strings.Join([]string{http.MethodGet, http.MethodPost}, ", "))
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	loc, lang := s.base.Localizer(w, r)
	view := templates.ScenarioPageView{}
	if shouldPrefillScenarioScript(r, s.base.IsHTMXRequest) {
		view.Script = defaultScenarioScript()
	}
	pageCtx := s.base.PageContext(lang, loc, r)
	s.base.RenderPage(
		w,
		r,
		templates.ScenariosPage(view, loc),
		templates.ScenariosFullPage(view, pageCtx),
		s.base.HTMXLocalizedPageTitle(loc, "title.scenarios", templates.AppName()),
	)
}

// HandleScenarioEvents renders the scenario events page.
func (s *service) HandleScenarioEvents(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := s.base.Localizer(w, r)
	view := s.buildScenarioEventsView(r, campaignID, loc)

	pageCtx := s.base.PageContext(lang, loc, r)
	s.base.RenderPage(
		w,
		r,
		templates.ScenarioEventsPage(view, loc),
		templates.ScenarioEventsFullPage(view, pageCtx),
		s.base.HTMXLocalizedPageTitle(loc, "title.scenarios", templates.AppName()),
	)
}

// HandleScenarioEventsTable renders the scenario events table fragment.
func (s *service) HandleScenarioEventsTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := s.base.Localizer(w, r)
	message := ""
	var events []templates.EventRow
	var totalCount int32
	var nextToken, prevToken string
	filters := parseEventFilters(r)
	pageToken := r.URL.Query().Get("page_token")

	if eventClient := s.base.EventClient(); eventClient != nil {
		ctx, cancel := s.base.GameGRPCCallContext(r.Context())
		defer cancel()

		filterExpr := buildEventFilterExpression(filters)
		eventsResp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   eventListPageSize,
			PageToken:  pageToken,
			OrderBy:    "seq desc",
			Filter:     filterExpr,
		})
		if err != nil {
			log.Printf("list scenario events: %v", err)
			message = loc.Sprintf("error.events_unavailable")
		} else if eventsResp != nil {
			events = buildEventRows(eventsResp.GetEvents(), loc)
			totalCount = eventsResp.GetTotalSize()
			nextToken = eventsResp.GetNextPageToken()
			prevToken = eventsResp.GetPreviousPageToken()
		}
	} else {
		message = loc.Sprintf("error.event_service_unavailable")
	}

	view := templates.ScenarioEventsView{
		CampaignID: campaignID,
		Events:     events,
		Filters:    filters,
		TotalCount: totalCount,
		NextToken:  nextToken,
		PrevToken:  prevToken,
		Message:    message,
	}

	if pushURL := eventFilterPushURL(routepath.ScenarioEvents(campaignID), filters, pageToken); pushURL != "" {
		w.Header().Set("HX-Push-Url", pushURL)
	}

	templ.Handler(templates.ScenarioEventsTableContent(view, loc)).ServeHTTP(w, r)
}

// HandleScenarioTimelineTable renders the scenario timeline table fragment.
func (s *service) HandleScenarioTimelineTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := s.base.Localizer(w, r)
	view := s.buildScenarioTimelineView(r, campaignID, loc)

	templ.Handler(templates.ScenarioTimelineTableContent(view, loc)).ServeHTTP(w, r)
}

func (s *service) handleScenarioRun(w http.ResponseWriter, r *http.Request) {
	loc, lang := s.base.Localizer(w, r)
	if !requireSameOrigin(w, r, loc) {
		return
	}
	if err := r.ParseForm(); err != nil {
		log.Printf("parse scenario form: %v", err)
		view := templates.ScenarioPageView{
			Logs:        loc.Sprintf("scenarios.error.parse_failed"),
			Status:      loc.Sprintf("scenarios.status.failed"),
			StatusBadge: "error",
			HasRun:      true,
		}
		s.renderScenarioResponse(w, r, view, loc, lang)
		return
	}

	script := strings.TrimSpace(r.FormValue("script"))
	view := templates.ScenarioPageView{Script: script}
	if script == "" {
		view.Logs = loc.Sprintf("scenarios.error.empty_script")
		view.Status = loc.Sprintf("scenarios.status.failed")
		view.StatusBadge = "error"
		view.HasRun = true
		s.renderScenarioResponse(w, r, view, loc, lang)
		return
	}
	if len(script) > maxScenarioScriptSize {
		view.Logs = loc.Sprintf("scenarios.error.script_too_large")
		view.Status = loc.Sprintf("scenarios.status.failed")
		view.StatusBadge = "error"
		view.HasRun = true
		s.renderScenarioResponse(w, r, view, loc, lang)
		return
	}

	logs, campaignID, runErr := s.runScenarioScript(r.Context(), script)
	if runErr != nil {
		logs = strings.TrimSpace(strings.Join([]string{logs, loc.Sprintf("scenarios.log.error_prefix", runErr.Error())}, "\n"))
		view.Status = loc.Sprintf("scenarios.status.failed")
		view.StatusBadge = "error"
	} else {
		view.Status = loc.Sprintf("scenarios.status.success")
		view.StatusBadge = "success"
	}
	view.Logs = logs
	view.CampaignID = campaignID
	view.HasRun = true
	if campaignID != "" {
		view.CampaignName = s.getCampaignName(r, campaignID, loc)
		view.Events = s.buildScenarioEventsView(r, campaignID, loc)
	}

	s.renderScenarioResponse(w, r, view, loc, lang)
}

func (s *service) renderScenarioResponse(w http.ResponseWriter, r *http.Request, view templates.ScenarioPageView, loc *message.Printer, lang string) {
	pageCtx := s.base.PageContext(lang, loc, r)
	s.base.RenderPage(
		w,
		r,
		templates.ScenarioScriptPanel(view, loc),
		templates.ScenariosFullPage(view, pageCtx),
		s.base.HTMXLocalizedPageTitle(loc, "title.scenarios", templates.AppName()),
	)
}

func (s *service) buildScenarioEventsView(r *http.Request, campaignID string, loc *message.Printer) templates.ScenarioEventsView {
	message := ""
	var events []templates.EventRow
	var totalCount int32
	var nextToken, prevToken string
	filters := parseEventFilters(r)
	pageToken := r.URL.Query().Get("page_token")

	if eventClient := s.base.EventClient(); eventClient != nil {
		ctx, cancel := s.base.GameGRPCCallContext(r.Context())
		defer cancel()

		filterExpr := buildEventFilterExpression(filters)
		eventsResp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   eventListPageSize,
			PageToken:  pageToken,
			OrderBy:    "seq desc",
			Filter:     filterExpr,
		})
		if err != nil {
			log.Printf("list scenario events: %v", err)
			message = loc.Sprintf("error.events_unavailable")
		} else if eventsResp != nil {
			events = buildEventRows(eventsResp.GetEvents(), loc)
			totalCount = eventsResp.GetTotalSize()
			nextToken = eventsResp.GetNextPageToken()
			prevToken = eventsResp.GetPreviousPageToken()
		}
	} else {
		message = loc.Sprintf("error.event_service_unavailable")
	}

	campaignName := s.getCampaignName(r, campaignID, loc)
	return templates.ScenarioEventsView{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		Events:       events,
		Filters:      filters,
		TotalCount:   totalCount,
		NextToken:    nextToken,
		PrevToken:    prevToken,
		Message:      message,
	}
}

func (s *service) buildScenarioTimelineView(r *http.Request, campaignID string, loc *message.Printer) templates.ScenarioTimelineView {
	message := ""
	var entries []templates.ScenarioTimelineEntry
	var totalCount int32
	var nextToken, prevToken string
	pageToken := r.URL.Query().Get("page_token")

	if eventClient := s.base.EventClient(); eventClient != nil {
		ctx, cancel := s.base.GameGRPCCallContext(r.Context())
		defer cancel()

		resp, err := eventClient.ListTimelineEntries(ctx, &statev1.ListTimelineEntriesRequest{
			CampaignId: campaignID,
			PageSize:   eventListPageSize,
			PageToken:  pageToken,
			OrderBy:    "seq",
		})
		if err != nil {
			log.Printf("list scenario timeline: %v", err)
			message = loc.Sprintf("error.events_unavailable")
		} else if resp != nil {
			entries = buildScenarioTimelineEntries(resp.GetEntries(), loc)
			totalCount = resp.GetTotalSize()
			nextToken = resp.GetNextPageToken()
			prevToken = resp.GetPreviousPageToken()
		}
	} else {
		message = loc.Sprintf("error.event_service_unavailable")
	}

	return templates.ScenarioTimelineView{
		CampaignID: campaignID,
		Entries:    entries,
		TotalCount: totalCount,
		NextToken:  nextToken,
		PrevToken:  prevToken,
		Message:    message,
	}
}

func (s *service) runScenarioScript(ctx context.Context, script string) (string, string, error) {
	tempDir := strings.TrimSpace(os.Getenv(scenarioTempDirEnv))
	if tempDir != "" {
		if err := os.MkdirAll(tempDir, 0o755); err != nil {
			return "", "", err
		}
	}
	file, err := os.CreateTemp(tempDir, "scenario-*.lua")
	if err != nil {
		return "", "", err
	}
	path := file.Name()
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("close scenario temp file: %v", err)
		}
		if err := os.Remove(path); err != nil {
			log.Printf("remove scenario temp file: %v", err)
		}
	}()

	if _, err := io.WriteString(file, script); err != nil {
		return "", "", err
	}

	var output bytes.Buffer
	logger := log.New(&output, "", 0)
	config := scenario.Config{
		GRPCAddr:         s.scenarioGRPCAddr(),
		Timeout:          10 * time.Second,
		Assertions:       scenario.AssertionStrict,
		ValidateComments: false,
		Verbose:          true,
		Logger:           logger,
	}
	if err := scenario.RunFile(ctx, config, path); err != nil {
		return strings.TrimSpace(output.String()), parseScenarioCampaignID(output.String()), err
	}
	return strings.TrimSpace(output.String()), parseScenarioCampaignID(output.String()), nil
}

func (s *service) scenarioGRPCAddr() string {
	if s == nil {
		return "localhost:8080"
	}
	if strings.TrimSpace(s.grpcAddr) != "" {
		return s.grpcAddr
	}
	if env := strings.TrimSpace(os.Getenv("FRACTURING_SPACE_GAME_ADDR")); env != "" {
		return env
	}
	return "localhost:8080"
}

func (s *service) getCampaignName(r *http.Request, campaignID string, loc *message.Printer) string {
	campaignClient := s.base.CampaignClient()
	if campaignClient == nil {
		return loc.Sprintf("label.campaign")
	}

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := campaignClient.GetCampaign(ctx, &statev1.GetCampaignRequest{CampaignId: campaignID})
	if err != nil || response == nil || response.GetCampaign() == nil {
		return loc.Sprintf("label.campaign")
	}
	return response.GetCampaign().GetName()
}

func defaultScenarioScript() string {
	return `local scene = Scenario.new("My Scenario")
scene:campaign({name = "My campaign"})

-- You must gather your party before venturing forth!



return scene`
}

func shouldPrefillScenarioScript(r *http.Request, isHTMXRequest func(*http.Request) bool) bool {
	if !isHTMXRequest(r) {
		return true
	}
	return r.URL.Query().Get("prefill") == "1"
}

func parseScenarioCampaignID(logs string) string {
	const prefix = "campaign created: id="
	for _, line := range strings.Split(logs, "\n") {
		index := strings.Index(line, prefix)
		if index == -1 {
			continue
		}
		remainder := strings.TrimSpace(line[index+len(prefix):])
		parts := strings.Fields(remainder)
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return ""
}

func requireSameOrigin(w http.ResponseWriter, r *http.Request, loc *message.Printer) bool {
	if r == nil {
		http.Error(w, loc.Sprintf("error.csrf_invalid"), http.StatusForbidden)
		return false
	}
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		if !sameOrigin(origin, r) {
			http.Error(w, loc.Sprintf("error.csrf_invalid"), http.StatusForbidden)
			return false
		}
		return true
	}
	if referer := strings.TrimSpace(r.Referer()); referer != "" {
		if !sameOrigin(referer, r) {
			http.Error(w, loc.Sprintf("error.csrf_invalid"), http.StatusForbidden)
			return false
		}
		return true
	}
	http.Error(w, loc.Sprintf("error.csrf_invalid"), http.StatusForbidden)
	return false
}

func sameOrigin(rawURL string, r *http.Request) bool {
	if rawURL == "" || rawURL == "null" || r == nil {
		return false
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return false
	}
	if !strings.EqualFold(parsed.Host, r.Host) {
		return false
	}
	if parsed.Scheme != "" {
		return strings.EqualFold(parsed.Scheme, requestScheme(r))
	}
	return true
}

func requestScheme(r *http.Request) string {
	if r == nil {
		return "http"
	}
	if proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); proto != "" {
		parts := strings.Split(proto, ",")
		return strings.ToLower(strings.TrimSpace(parts[0]))
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

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

func escapeAIP160StringLiteral(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}

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

func formatEventType(eventType string, loc *message.Printer) string {
	switch eventType {
	case "campaign.created":
		return loc.Sprintf("event.campaign_created")
	case "campaign.forked":
		return loc.Sprintf("event.campaign_forked")
	case "campaign.updated":
		return loc.Sprintf("event.campaign_updated")
	case "participant.joined":
		return loc.Sprintf("event.participant_joined")
	case "participant.left":
		return loc.Sprintf("event.participant_left")
	case "participant.updated":
		return loc.Sprintf("event.participant_updated")
	case "character.created":
		return loc.Sprintf("event.character_created")
	case "character.deleted":
		return loc.Sprintf("event.character_deleted")
	case "character.updated":
		return loc.Sprintf("event.character_updated")
	case "character.profile_updated":
		return loc.Sprintf("event.character_profile_updated")
	case "session.started":
		return loc.Sprintf("event.session_started")
	case "session.ended":
		return loc.Sprintf("event.session_ended")
	case "session.gate_opened":
		return loc.Sprintf("event.session_gate_opened")
	case "session.gate_resolved":
		return loc.Sprintf("event.session_gate_resolved")
	case "session.gate_abandoned":
		return loc.Sprintf("event.session_gate_abandoned")
	case "session.spotlight_set":
		return loc.Sprintf("event.session_spotlight_set")
	case "session.spotlight_cleared":
		return loc.Sprintf("event.session_spotlight_cleared")
	case "invite.created":
		return loc.Sprintf("event.invite_created")
	case "invite.updated":
		return loc.Sprintf("event.invite_updated")
	case "action.roll_resolved":
		return loc.Sprintf("event.action_roll_resolved")
	case "action.outcome_applied":
		return loc.Sprintf("event.action_outcome_applied")
	case "action.outcome_rejected":
		return loc.Sprintf("event.action_outcome_rejected")
	case "action.note_added":
		return loc.Sprintf("event.action_note_added")
	case "action.character_state_patched":
		return loc.Sprintf("event.action_character_state_patched")
	case "action.gm_fear_changed":
		return loc.Sprintf("event.action_gm_fear_changed")
	case "action.death_move_resolved":
		return loc.Sprintf("event.action_death_move_resolved")
	case "action.blaze_of_glory_resolved":
		return loc.Sprintf("event.action_blaze_of_glory_resolved")
	case "action.attack_resolved":
		return loc.Sprintf("event.action_attack_resolved")
	case "action.reaction_resolved":
		return loc.Sprintf("event.action_reaction_resolved")
	case "action.damage_roll_resolved":
		return loc.Sprintf("event.action_damage_roll_resolved")
	case "action.adversary_action_resolved":
		return loc.Sprintf("event.action_adversary_action_resolved")
	default:
		parts := strings.Split(eventType, ".")
		if len(parts) > 0 {
			last := parts[len(parts)-1]
			if len(last) > 0 {
				formatted := strings.ReplaceAll(last, "_", " ")
				return strings.ToUpper(formatted[:1]) + formatted[1:]
			}
		}
		return eventType
	}
}

func formatActorType(actorType string, loc *message.Printer) string {
	if actorType == "" {
		return ""
	}
	switch actorType {
	case "system":
		return loc.Sprintf("filter.actor.system")
	case "participant":
		return loc.Sprintf("filter.actor.participant")
	case "gm":
		return loc.Sprintf("filter.actor.gm")
	default:
		return actorType
	}
}

func formatEventDescription(event *statev1.Event, loc *message.Printer) string {
	if event == nil {
		return ""
	}
	return formatEventType(event.GetType(), loc)
}

func formatTimestamp(value *timestamppb.Timestamp) string {
	if value == nil {
		return ""
	}
	return value.AsTime().Format("2006-01-02 15:04:05")
}
