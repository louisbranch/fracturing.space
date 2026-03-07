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
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/eventview"
	adminerrors "github.com/louisbranch/fracturing.space/internal/services/admin/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
	routepath "github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"github.com/louisbranch/fracturing.space/internal/tools/scenario"
	"golang.org/x/text/message"
)

const (
	// eventListPageSize caps the number of events shown per page.
	eventListPageSize = 50
	// maxScenarioScriptSize caps scenario scripts to limit resource usage.
	maxScenarioScriptSize = 100 * 1024
	// scenarioTempDirEnv configures the temp directory for scenario scripts.
	scenarioTempDirEnv = "FRACTURING_SPACE_SCENARIO_TMPDIR"
)

// handlers implements the scenarios Handlers contract.
type handlers struct {
	base           modulehandler.Base
	grpcAddr       string
	eventClient    statev1.EventServiceClient
	campaignClient statev1.CampaignServiceClient
}

// NewHandlers returns the scenarios handler implementation.
func NewHandlers(
	base modulehandler.Base,
	grpcAddr string,
	eventClient statev1.EventServiceClient,
	campaignClient statev1.CampaignServiceClient,
) Handlers {
	return &handlers{
		base:           base,
		grpcAddr:       strings.TrimSpace(grpcAddr),
		eventClient:    eventClient,
		campaignClient: campaignClient,
	}
}

// HandleScenarios renders the scenario page and handles scenario run submissions.
func (s *handlers) HandleScenarios(w http.ResponseWriter, r *http.Request) {
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
func (s *handlers) HandleScenarioEvents(w http.ResponseWriter, r *http.Request, campaignID string) {
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
func (s *handlers) HandleScenarioEventsTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := s.base.Localizer(w, r)
	message := ""
	var events []templates.EventRow
	var totalCount int32
	var nextToken, prevToken string
	filters := eventview.ParseEventFilters(r)
	pageToken := r.URL.Query().Get("page_token")

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	filterExpr := eventview.BuildEventFilterExpression(filters)
	eventsResp, err := s.eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   eventListPageSize,
		PageToken:  pageToken,
		OrderBy:    "seq desc",
		Filter:     filterExpr,
	})
	if err != nil {
		adminerrors.LogError(r, "list scenario events: %v", err)
		message = loc.Sprintf("error.events_unavailable")
	} else if eventsResp != nil {
		events = eventview.BuildEventRows(eventsResp.GetEvents(), loc)
		totalCount = eventsResp.GetTotalSize()
		nextToken = eventsResp.GetNextPageToken()
		prevToken = eventsResp.GetPreviousPageToken()
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

	if pushURL := eventview.EventFilterPushURL(routepath.ScenarioEvents(campaignID), filters, pageToken); pushURL != "" {
		w.Header().Set("HX-Push-Url", pushURL)
	}

	templ.Handler(templates.ScenarioEventsTableContent(view, loc)).ServeHTTP(w, r)
}

// HandleScenarioTimelineTable renders the scenario timeline table fragment.
func (s *handlers) HandleScenarioTimelineTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := s.base.Localizer(w, r)
	view := s.buildScenarioTimelineView(r, campaignID, loc)

	templ.Handler(templates.ScenarioTimelineTableContent(view, loc)).ServeHTTP(w, r)
}

func (s *handlers) handleScenarioRun(w http.ResponseWriter, r *http.Request) {
	loc, lang := s.base.Localizer(w, r)
	if !requireSameOrigin(w, r, loc) {
		return
	}
	if err := r.ParseForm(); err != nil {
		adminerrors.LogError(r, "parse scenario form: %v", err)
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

func (s *handlers) renderScenarioResponse(w http.ResponseWriter, r *http.Request, view templates.ScenarioPageView, loc *message.Printer, lang string) {
	pageCtx := s.base.PageContext(lang, loc, r)
	s.base.RenderPage(
		w,
		r,
		templates.ScenarioScriptPanel(view, loc),
		templates.ScenariosFullPage(view, pageCtx),
		s.base.HTMXLocalizedPageTitle(loc, "title.scenarios", templates.AppName()),
	)
}

func (s *handlers) buildScenarioEventsView(r *http.Request, campaignID string, loc *message.Printer) templates.ScenarioEventsView {
	message := ""
	var events []templates.EventRow
	var totalCount int32
	var nextToken, prevToken string
	filters := eventview.ParseEventFilters(r)
	pageToken := r.URL.Query().Get("page_token")

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	filterExpr := eventview.BuildEventFilterExpression(filters)
	eventsResp, err := s.eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   eventListPageSize,
		PageToken:  pageToken,
		OrderBy:    "seq desc",
		Filter:     filterExpr,
	})
	if err != nil {
		adminerrors.LogError(r, "list scenario events: %v", err)
		message = loc.Sprintf("error.events_unavailable")
	} else if eventsResp != nil {
		events = eventview.BuildEventRows(eventsResp.GetEvents(), loc)
		totalCount = eventsResp.GetTotalSize()
		nextToken = eventsResp.GetNextPageToken()
		prevToken = eventsResp.GetPreviousPageToken()
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

func (s *handlers) buildScenarioTimelineView(r *http.Request, campaignID string, loc *message.Printer) templates.ScenarioTimelineView {
	message := ""
	var entries []templates.ScenarioTimelineEntry
	var totalCount int32
	var nextToken, prevToken string
	pageToken := r.URL.Query().Get("page_token")

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	resp, err := s.eventClient.ListTimelineEntries(ctx, &statev1.ListTimelineEntriesRequest{
		CampaignId: campaignID,
		PageSize:   eventListPageSize,
		PageToken:  pageToken,
		OrderBy:    "seq",
	})
	if err != nil {
		adminerrors.LogError(r, "list scenario timeline: %v", err)
		message = loc.Sprintf("error.events_unavailable")
	} else if resp != nil {
		entries = buildScenarioTimelineEntries(resp.GetEntries(), loc)
		totalCount = resp.GetTotalSize()
		nextToken = resp.GetNextPageToken()
		prevToken = resp.GetPreviousPageToken()
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

func (s *handlers) runScenarioScript(ctx context.Context, script string) (string, string, error) {
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

func (s *handlers) scenarioGRPCAddr() string {
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

func (s *handlers) getCampaignName(r *http.Request, campaignID string, loc *message.Printer) string {
	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := s.campaignClient.GetCampaign(ctx, &statev1.GetCampaignRequest{CampaignId: campaignID})
	if err != nil || response == nil || response.GetCampaign() == nil {
		return loc.Sprintf("label.campaign")
	}
	return response.GetCampaign().GetName()
}

func defaultScenarioScript() string {
	return `local scn = Scenario.new("My Scenario")
scn:campaign({name = "My campaign", system = "DAGGERHEART"})

-- You must gather your party before venturing forth!



return scn`
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
