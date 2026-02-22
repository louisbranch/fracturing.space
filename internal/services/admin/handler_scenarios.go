package admin

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
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	routepath "github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"github.com/louisbranch/fracturing.space/internal/tools/scenario"
	"golang.org/x/text/message"
)

func (h *Handler) handleScenarios(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.handleScenarioRun(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	loc, lang := h.localizer(w, r)
	view := templates.ScenarioPageView{}
	if shouldPrefillScenarioScript(r) {
		view.Script = defaultScenarioScript()
	}
	pageCtx := h.pageContext(lang, loc, r)
	renderPage(
		w,
		r,
		templates.ScenariosPage(view, loc),
		templates.ScenariosFullPage(view, pageCtx),
		htmxLocalizedPageTitle(loc, "title.scenarios", templates.AppName()),
	)
}

func defaultScenarioScript() string {
	return `local scene = Scenario.new("My Scenario")
scene:campaign({name = "My campaign"})

-- You must gather your party before venturing forth!



return scene`
}

func shouldPrefillScenarioScript(r *http.Request) bool {
	if !isHTMXRequest(r) {
		return true
	}
	return r.URL.Query().Get("prefill") == "1"
}

func (h *Handler) handleScenarioRun(w http.ResponseWriter, r *http.Request) {
	loc, lang := h.localizer(w, r)
	if !requireSameOrigin(w, r, loc) {
		return
	}
	if err := r.ParseForm(); err != nil {
		log.Printf("parse scenario form: %v", err)
		view := templates.ScenarioPageView{
			Logs:        loc.Sprintf("scenarios.error.parse_failed"),
			Status:      loc.Sprintf("scenarios.status.failed"),
			StatusBadge: "error",
		}
		view.HasRun = true
		h.renderScenarioResponse(w, r, view, loc, lang)
		return
	}

	script := strings.TrimSpace(r.FormValue("script"))
	view := templates.ScenarioPageView{Script: script}
	if script == "" {
		view.Logs = loc.Sprintf("scenarios.error.empty_script")
		view.Status = loc.Sprintf("scenarios.status.failed")
		view.StatusBadge = "error"
		view.HasRun = true
		h.renderScenarioResponse(w, r, view, loc, lang)
		return
	}
	if len(script) > maxScenarioScriptSize {
		view.Logs = loc.Sprintf("scenarios.error.script_too_large")
		view.Status = loc.Sprintf("scenarios.status.failed")
		view.StatusBadge = "error"
		view.HasRun = true
		h.renderScenarioResponse(w, r, view, loc, lang)
		return
	}

	logs, campaignID, runErr := h.runScenarioScript(r.Context(), script)
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
		view.CampaignName = getCampaignName(h, r, campaignID, loc)
		view.Events = h.buildScenarioEventsView(r, campaignID, loc)
	}

	h.renderScenarioResponse(w, r, view, loc, lang)
}

func (h *Handler) renderScenarioResponse(w http.ResponseWriter, r *http.Request, view templates.ScenarioPageView, loc *message.Printer, lang string) {
	pageCtx := h.pageContext(lang, loc, r)
	renderPage(
		w,
		r,
		templates.ScenarioScriptPanel(view, loc),
		templates.ScenariosFullPage(view, pageCtx),
		htmxLocalizedPageTitle(loc, "title.scenarios", templates.AppName()),
	)
}

func (h *Handler) handleScenarioEvents(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := h.localizer(w, r)
	view := h.buildScenarioEventsView(r, campaignID, loc)

	pageCtx := h.pageContext(lang, loc, r)
	renderPage(
		w,
		r,
		templates.ScenarioEventsPage(view, loc),
		templates.ScenarioEventsFullPage(view, pageCtx),
		htmxLocalizedPageTitle(loc, "title.scenarios", templates.AppName()),
	)
}

func (h *Handler) handleScenarioEventsTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := h.localizer(w, r)
	message := ""
	var events []templates.EventRow
	var totalCount int32
	var nextToken, prevToken string
	filters := parseEventFilters(r)
	pageToken := r.URL.Query().Get("page_token")

	if eventClient := h.eventClient(); eventClient != nil {
		ctx, cancel := h.gameGRPCCallContext(r.Context())
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

func (h *Handler) handleScenarioTimelineTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := h.localizer(w, r)
	view := h.buildScenarioTimelineView(r, campaignID, loc)

	templ.Handler(templates.ScenarioTimelineTableContent(view, loc)).ServeHTTP(w, r)
}

func (h *Handler) buildScenarioEventsView(r *http.Request, campaignID string, loc *message.Printer) templates.ScenarioEventsView {
	message := ""
	var events []templates.EventRow
	var totalCount int32
	var nextToken, prevToken string
	filters := parseEventFilters(r)
	pageToken := r.URL.Query().Get("page_token")

	if eventClient := h.eventClient(); eventClient != nil {
		ctx, cancel := h.gameGRPCCallContext(r.Context())
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

	campaignName := getCampaignName(h, r, campaignID, loc)
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

func (h *Handler) buildScenarioTimelineView(r *http.Request, campaignID string, loc *message.Printer) templates.ScenarioTimelineView {
	message := ""
	var entries []templates.ScenarioTimelineEntry
	var totalCount int32
	var nextToken, prevToken string
	pageToken := r.URL.Query().Get("page_token")

	if eventClient := h.eventClient(); eventClient != nil {
		ctx, cancel := h.gameGRPCCallContext(r.Context())
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

func (h *Handler) runScenarioScript(ctx context.Context, script string) (string, string, error) {
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
		GRPCAddr:         h.scenarioGRPCAddr(),
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

func (h *Handler) scenarioGRPCAddr() string {
	if h == nil {
		return "localhost:8080"
	}
	if strings.TrimSpace(h.grpcAddr) != "" {
		return h.grpcAddr
	}
	if env := strings.TrimSpace(os.Getenv("FRACTURING_SPACE_GAME_ADDR")); env != "" {
		return env
	}
	return "localhost:8080"
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
