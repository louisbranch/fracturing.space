package admin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/a-h/templ"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"github.com/louisbranch/fracturing.space/internal/services/admin/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/tools/scenario"
	"golang.org/x/text/message"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// grpcRequestTimeout caps the gRPC request time for admin requests.
	grpcRequestTimeout = timeouts.GRPCRequest
	// campaignThemePromptLimit caps the number of characters shown in the table.
	campaignThemePromptLimit = 80
	// sessionListPageSize caps the number of sessions shown in the UI.
	sessionListPageSize = 10
	// eventListPageSize caps the number of events shown per page.
	eventListPageSize = 50
	// inviteListPageSize caps the number of invites shown per page.
	inviteListPageSize = 50
	// impersonationCookieName stores the active impersonation session ID.
	impersonationCookieName = "fs-impersonation-session"
	// impersonationSessionTTL controls how long impersonation sessions stay valid.
	impersonationSessionTTL = 24 * time.Hour
	// impersonationCleanupInterval controls how often expired sessions are purged.
	impersonationCleanupInterval = 30 * time.Minute
	// maxScenarioScriptSize caps scenario scripts to limit resource usage.
	maxScenarioScriptSize = 100 * 1024
	// scenarioTempDirEnv configures the temp directory for scenario scripts.
	scenarioTempDirEnv = "FRACTURING_SPACE_SCENARIO_TMPDIR"
)

// GRPCClientProvider supplies gRPC clients for request handling.
type GRPCClientProvider interface {
	AuthClient() authv1.AuthServiceClient
	CampaignClient() statev1.CampaignServiceClient
	SessionClient() statev1.SessionServiceClient
	CharacterClient() statev1.CharacterServiceClient
	ParticipantClient() statev1.ParticipantServiceClient
	InviteClient() statev1.InviteServiceClient
	SnapshotClient() statev1.SnapshotServiceClient
	EventClient() statev1.EventServiceClient
	StatisticsClient() statev1.StatisticsServiceClient
	SystemClient() statev1.SystemServiceClient
}

// Handler routes admin dashboard requests.
type Handler struct {
	clientProvider GRPCClientProvider
	impersonation  *impersonationStore
	grpcAddr       string
}

type impersonationSession struct {
	userID      string
	displayName string
	expiresAt   time.Time
}

type impersonationStore struct {
	mu          sync.RWMutex
	sessions    map[string]impersonationSession
	lastCleanup time.Time
}

func newImpersonationStore() *impersonationStore {
	return &impersonationStore{sessions: make(map[string]impersonationSession)}
}

func (s *impersonationStore) Get(sessionID string) (impersonationSession, bool) {
	if s == nil {
		return impersonationSession{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	s.cleanupLocked(now)
	session, ok := s.sessions[sessionID]
	if !ok {
		return impersonationSession{}, false
	}
	if now.After(session.expiresAt) {
		delete(s.sessions, sessionID)
		return impersonationSession{}, false
	}
	return session, true
}

func (s *impersonationStore) Set(sessionID string, session impersonationSession) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	s.cleanupLocked(now)
	session.expiresAt = now.Add(impersonationSessionTTL)
	s.sessions[sessionID] = session
}

func (s *impersonationStore) Delete(sessionID string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

func (s *impersonationStore) cleanupLocked(now time.Time) {
	if now.Sub(s.lastCleanup) < impersonationCleanupInterval {
		return
	}
	for key, session := range s.sessions {
		if now.After(session.expiresAt) {
			delete(s.sessions, key)
		}
	}
	s.lastCleanup = now
}

// NewHandler builds the HTTP handler for the admin server.
func NewHandler(clientProvider GRPCClientProvider) http.Handler {
	return NewHandlerWithConfig(clientProvider, "")
}

// NewHandlerWithConfig builds the HTTP handler with explicit configuration.
func NewHandlerWithConfig(clientProvider GRPCClientProvider, grpcAddr string) http.Handler {
	handler := &Handler{
		clientProvider: clientProvider,
		impersonation:  newImpersonationStore(),
		grpcAddr:       strings.TrimSpace(grpcAddr),
	}
	return handler.routes()
}

func (h *Handler) localizer(w http.ResponseWriter, r *http.Request) (*message.Printer, string) {
	tag, persist := i18n.ResolveTag(r)
	if persist {
		i18n.SetLanguageCookie(w, tag)
	}
	return i18n.Printer(tag), tag.String()
}

func (h *Handler) pageContext(lang string, loc *message.Printer, r *http.Request) templates.PageContext {
	path := ""
	query := ""
	if r != nil && r.URL != nil {
		path = r.URL.Path
		query = r.URL.RawQuery
	}
	return templates.PageContext{
		Lang:          lang,
		Loc:           loc,
		CurrentPath:   path,
		CurrentQuery:  query,
		Impersonation: h.impersonationView(r),
	}
}

// routes wires the HTTP routes for the admin handler.
func (h *Handler) routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("internal/services/admin/static"))))
	mux.Handle("/", http.HandlerFunc(h.handleDashboard))
	mux.Handle("/dashboard/content", http.HandlerFunc(h.handleDashboardContent))
	mux.Handle("/campaigns", http.HandlerFunc(h.handleCampaignsPage))
	mux.Handle("/campaigns/table", http.HandlerFunc(h.handleCampaignsTable))
	mux.Handle("/campaigns/create", http.HandlerFunc(h.handleCampaignCreate))
	mux.Handle("/campaigns/", http.HandlerFunc(h.handleCampaignRoutes))
	mux.Handle("/systems", http.HandlerFunc(h.handleSystemsPage))
	mux.Handle("/systems/table", http.HandlerFunc(h.handleSystemsTable))
	mux.Handle("/systems/", http.HandlerFunc(h.handleSystemRoutes))
	mux.Handle("/users", http.HandlerFunc(h.handleUsersPage))
	mux.Handle("/users/table", http.HandlerFunc(h.handleUsersTable))
	mux.Handle("/users/lookup", http.HandlerFunc(h.handleUserLookup))
	mux.Handle("/users/create", http.HandlerFunc(h.handleCreateUser))
	mux.Handle("/users/magic-link", http.HandlerFunc(h.handleMagicLink))
	mux.Handle("/users/impersonate", http.HandlerFunc(h.handleImpersonateUser))
	mux.Handle("/users/logout", http.HandlerFunc(h.handleLogout))
	mux.Handle("/users/", http.HandlerFunc(h.handleUserRoutes))
	mux.Handle("/scenarios", http.HandlerFunc(h.handleScenarios))
	mux.Handle("/scenarios/", http.HandlerFunc(h.handleScenarioRoutes))
	return h.withImpersonation(mux)
}

func (h *Handler) withImpersonation(next http.Handler) http.Handler {
	if h == nil || next == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		impersonation := h.currentImpersonation(r)
		if impersonation != nil && impersonation.userID != "" {
			ctx := metadata.AppendToOutgoingContext(r.Context(), grpcmeta.UserIDHeader, impersonation.userID)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

// handleScenarios renders the scenarios page or runs a scenario script.
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
	if isHTMXRequest(r) {
		templ.Handler(templates.ScenariosPage(view, loc)).ServeHTTP(w, r)
		return
	}
	pageCtx := h.pageContext(lang, loc, r)
	templ.Handler(templates.ScenariosFullPage(view, pageCtx)).ServeHTTP(w, r)
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

// handleScenarioRoutes dispatches scenario subroutes.
func (h *Handler) handleScenarioRoutes(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/") {
		canonical := strings.TrimRight(r.URL.Path, "/")
		if canonical == "" {
			canonical = "/"
		}
		http.Redirect(w, r, canonical, http.StatusMovedPermanently)
		return
	}
	scenarioPath := strings.TrimPrefix(r.URL.Path, "/scenarios/")
	parts := splitPathParts(scenarioPath)

	if len(parts) == 2 && parts[1] == "events" {
		h.handleScenarioEvents(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "events" && parts[2] == "table" {
		h.handleScenarioEventsTable(w, r, parts[0])
		return
	}

	http.NotFound(w, r)
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
	if campaignID != "" {
		view.EventsURL = "/scenarios/" + campaignID + "/events"
	}
	view.HasRun = true
	if campaignID != "" {
		view.CampaignName = getCampaignName(h, r, campaignID, loc)
	}

	h.renderScenarioResponse(w, r, view, loc, lang)
}

func (h *Handler) renderScenarioResponse(w http.ResponseWriter, r *http.Request, view templates.ScenarioPageView, loc *message.Printer, lang string) {
	if isHTMXRequest(r) {
		templ.Handler(templates.ScenarioScriptPanel(view, loc)).ServeHTTP(w, r)
		return
	}
	pageCtx := h.pageContext(lang, loc, r)
	templ.Handler(templates.ScenariosFullPage(view, pageCtx)).ServeHTTP(w, r)
}

func (h *Handler) handleScenarioEvents(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := h.localizer(w, r)
	message := ""
	var events []templates.EventRow
	var totalCount int32
	var nextToken, prevToken string
	filters := parseEventFilters(r)
	pageToken := r.URL.Query().Get("page_token")

	if eventClient := h.eventClient(); eventClient != nil {
		ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
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
	view := templates.ScenarioEventsView{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		Events:       events,
		Filters:      filters,
		TotalCount:   totalCount,
		NextToken:    nextToken,
		PrevToken:    prevToken,
		Message:      message,
	}

	if isHTMXRequest(r) {
		templ.Handler(templates.ScenarioEventsPage(view, loc)).ServeHTTP(w, r)
		return
	}
	pageCtx := h.pageContext(lang, loc, r)
	templ.Handler(templates.ScenarioEventsFullPage(view, pageCtx)).ServeHTTP(w, r)
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
		ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
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

	if pushURL := eventFilterPushURL("/scenarios/"+campaignID+"/events", filters, pageToken); pushURL != "" {
		w.Header().Set("HX-Push-Url", pushURL)
	}

	templ.Handler(templates.ScenarioEventsTableContent(view, loc)).ServeHTTP(w, r)
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
		GRPCAddr:   h.scenarioGRPCAddr(),
		Timeout:    10 * time.Second,
		Assertions: scenario.AssertionStrict,
		Verbose:    true,
		Logger:     logger,
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

func (h *Handler) impersonationView(r *http.Request) *templates.ImpersonationView {
	impersonation := h.currentImpersonation(r)
	if impersonation == nil {
		return nil
	}
	return &templates.ImpersonationView{
		UserID:      impersonation.userID,
		DisplayName: impersonation.displayName,
	}
}

func (h *Handler) currentImpersonation(r *http.Request) *impersonationSession {
	if h == nil || h.impersonation == nil {
		return nil
	}
	sessionID := impersonationSessionID(r)
	if sessionID == "" {
		return nil
	}
	session, ok := h.impersonation.Get(sessionID)
	if !ok {
		return nil
	}
	return &session
}

func impersonationSessionID(r *http.Request) string {
	if r == nil {
		return ""
	}
	cookie, err := r.Cookie(impersonationCookieName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
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

func isHTTPS(r *http.Request) bool {
	return requestScheme(r) == "https"
}

// handleUsersPage renders the users page.
func (h *Handler) handleUsersPage(w http.ResponseWriter, r *http.Request) {
	loc, lang := h.localizer(w, r)
	pageCtx := h.pageContext(lang, loc, r)
	view := templates.UsersPageView{Impersonation: pageCtx.Impersonation}

	if message := strings.TrimSpace(r.URL.Query().Get("message")); message != "" {
		view.Message = message
	}

	if userID := strings.TrimSpace(r.URL.Query().Get("user_id")); userID != "" {
		h.redirectToUserDetail(w, r, userID)
		return
	}

	if isHTMXRequest(r) {
		templ.Handler(templates.UsersPage(view, loc)).ServeHTTP(w, r)
		return
	}

	templ.Handler(templates.UsersFullPage(view, pageCtx)).ServeHTTP(w, r)
}

// handleUserLookup redirects the get-user form to the detail page.
func (h *Handler) handleUserLookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	loc, lang := h.localizer(w, r)
	pageCtx := h.pageContext(lang, loc, r)
	view := templates.UsersPageView{Impersonation: pageCtx.Impersonation}

	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if userID == "" {
		view.Message = loc.Sprintf("error.user_id_required")
		if isHTMXRequest(r) {
			templ.Handler(templates.UsersPage(view, loc)).ServeHTTP(w, r)
			return
		}
		templ.Handler(templates.UsersFullPage(view, pageCtx)).ServeHTTP(w, r)
		return
	}

	h.redirectToUserDetail(w, r, userID)
}

// handleUserRoutes dispatches the user detail route.
func (h *Handler) handleUserRoutes(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/") {
		canonical := strings.TrimRight(r.URL.Path, "/")
		if canonical == "" {
			canonical = "/"
		}
		http.Redirect(w, r, canonical, http.StatusMovedPermanently)
		return
	}
	userPath := strings.TrimPrefix(r.URL.Path, "/users/")
	parts := splitPathParts(userPath)
	if len(parts) == 2 && parts[1] == "invites" {
		h.handleUserInvites(w, r, parts[0])
		return
	}
	if len(parts) == 1 && strings.TrimSpace(parts[0]) != "" {
		h.handleUserDetail(w, r, parts[0])
		return
	}
	http.NotFound(w, r)
}

// handleUserDetail renders the single-user detail page.
func (h *Handler) handleUserDetail(w http.ResponseWriter, r *http.Request, userID string) {
	loc, lang := h.localizer(w, r)
	pageCtx := h.pageContext(lang, loc, r)
	view := templates.UserDetailPageView{Impersonation: pageCtx.Impersonation}

	if message := strings.TrimSpace(r.URL.Query().Get("message")); message != "" {
		view.Message = message
	}

	ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
	defer cancel()

	detail, message := h.loadUserDetail(ctx, userID, loc)
	view.Detail = detail
	if message != "" && view.Message == "" {
		view.Message = message
	}
	h.populateUserInvitesIfImpersonating(ctx, view.Detail, view.Impersonation, loc)

	h.renderUserDetail(w, r, view, pageCtx, loc, "details")
}

// handleUserInvites renders the user pending invites tab.
func (h *Handler) handleUserInvites(w http.ResponseWriter, r *http.Request, userID string) {
	loc, lang := h.localizer(w, r)
	pageCtx := h.pageContext(lang, loc, r)
	view := templates.UserDetailPageView{Impersonation: pageCtx.Impersonation}

	if message := strings.TrimSpace(r.URL.Query().Get("message")); message != "" {
		view.Message = message
	}

	ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
	defer cancel()

	detail, message := h.loadUserDetail(ctx, userID, loc)
	view.Detail = detail
	if message != "" && view.Message == "" {
		view.Message = message
	}

	h.populateUserInvitesIfImpersonating(ctx, view.Detail, view.Impersonation, loc)

	h.renderUserDetail(w, r, view, pageCtx, loc, "invites")
}

// handleCreateUser creates a user from a form submission.
func (h *Handler) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	loc, lang := h.localizer(w, r)
	pageCtx := h.pageContext(lang, loc, r)
	view := templates.UsersPageView{Impersonation: pageCtx.Impersonation}

	if err := r.ParseForm(); err != nil {
		view.Message = loc.Sprintf("error.user_create_invalid")
		templ.Handler(templates.UsersFullPage(view, pageCtx)).ServeHTTP(w, r)
		return
	}

	displayName := strings.TrimSpace(r.FormValue("display_name"))
	if displayName == "" {
		view.Message = loc.Sprintf("error.user_display_name_required")
		templ.Handler(templates.UsersFullPage(view, pageCtx)).ServeHTTP(w, r)
		return
	}

	client := h.authClient()
	if client == nil {
		view.Message = loc.Sprintf("error.user_service_unavailable")
		templ.Handler(templates.UsersFullPage(view, pageCtx)).ServeHTTP(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
	defer cancel()

	locale := localeFromTag(lang)
	response, err := client.CreateUser(ctx, &authv1.CreateUserRequest{
		DisplayName: displayName,
		Locale:      locale,
	})
	if err != nil || response.GetUser() == nil {
		log.Printf("create user: %v", err)
		view.Message = loc.Sprintf("error.user_create_failed")
		templ.Handler(templates.UsersFullPage(view, pageCtx)).ServeHTTP(w, r)
		return
	}

	created := response.GetUser()
	redirectURL := "/users/" + created.GetId()
	message := loc.Sprintf("users.create.success")
	redirectURL = redirectURL + "?message=" + url.QueryEscape(message)
	if isHTMXRequest(r) {
		w.Header().Set("Location", redirectURL)
		w.Header().Set("HX-Redirect", redirectURL)
		w.WriteHeader(http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// handleMagicLink generates a magic link for a user email.
func (h *Handler) handleMagicLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	loc, lang := h.localizer(w, r)
	pageCtx := h.pageContext(lang, loc, r)
	view := templates.UserDetailPageView{Impersonation: pageCtx.Impersonation}
	if !requireSameOrigin(w, r, loc) {
		return
	}

	if err := r.ParseForm(); err != nil {
		view.Message = loc.Sprintf("error.magic_link_invalid")
		h.renderUserDetail(w, r, view, pageCtx, loc, "details")
		return
	}

	userID := strings.TrimSpace(r.FormValue("user_id"))
	email := strings.TrimSpace(r.FormValue("email"))
	if userID == "" || email == "" {
		view.Message = loc.Sprintf("error.magic_link_invalid")
		h.renderUserDetail(w, r, view, pageCtx, loc, "details")
		return
	}

	client := h.authClient()
	if client == nil {
		view.Message = loc.Sprintf("error.user_service_unavailable")
		h.renderUserDetail(w, r, view, pageCtx, loc, "details")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
	defer cancel()
	response, err := client.GenerateMagicLink(ctx, &authv1.GenerateMagicLinkRequest{
		UserId: userID,
		Email:  email,
	})
	if err != nil || response.GetMagicLinkUrl() == "" {
		log.Printf("generate magic link: %v", err)
		view.Message = loc.Sprintf("error.magic_link_failed")
		view.Detail, _ = h.loadUserDetail(ctx, userID, loc)
		h.renderUserDetail(w, r, view, pageCtx, loc, "details")
		return
	}

	detail, message := h.loadUserDetail(ctx, userID, loc)
	view.Detail = detail
	if message != "" && view.Message == "" {
		view.Message = message
	}
	view.MagicLinkURL = response.GetMagicLinkUrl()
	view.MagicLinkEmail = email
	if response.GetExpiresAt() != nil {
		view.MagicLinkExpiresAt = formatTimestamp(response.GetExpiresAt())
	}

	view.Message = loc.Sprintf("users.magic.success")
	h.renderUserDetail(w, r, view, pageCtx, loc, "details")
}

// handleImpersonateUser creates an impersonation session for a user.
func (h *Handler) handleImpersonateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	loc, lang := h.localizer(w, r)
	pageCtx := h.pageContext(lang, loc, r)
	view := templates.UserDetailPageView{Impersonation: pageCtx.Impersonation}
	if !requireSameOrigin(w, r, loc) {
		return
	}

	if err := r.ParseForm(); err != nil {
		view.Message = loc.Sprintf("error.user_impersonate_invalid")
		h.renderUserDetail(w, r, view, pageCtx, loc, "details")
		return
	}

	userID := strings.TrimSpace(r.FormValue("user_id"))
	if userID == "" {
		view.Message = loc.Sprintf("error.user_id_required")
		h.renderUserDetail(w, r, view, pageCtx, loc, "details")
		return
	}

	client := h.authClient()
	if client == nil {
		view.Message = loc.Sprintf("error.user_service_unavailable")
		h.renderUserDetail(w, r, view, pageCtx, loc, "details")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
	defer cancel()

	response, err := client.GetUser(ctx, &authv1.GetUserRequest{UserId: userID})
	if err != nil || response.GetUser() == nil {
		log.Printf("get user for impersonation: %v", err)
		view.Message = loc.Sprintf("error.user_not_found")
		h.renderUserDetail(w, r, view, pageCtx, loc, "details")
		return
	}

	user := response.GetUser()
	if sessionID := impersonationSessionID(r); sessionID != "" {
		if h.impersonation != nil {
			h.impersonation.Delete(sessionID)
		}
	}
	sessionID, err := id.NewID()
	if err != nil {
		log.Printf("impersonation session id: %v", err)
		view.Message = loc.Sprintf("error.user_impersonate_failed")
		h.renderUserDetail(w, r, view, pageCtx, loc, "details")
		return
	}

	if h.impersonation != nil {
		h.impersonation.Set(sessionID, impersonationSession{
			userID:      user.GetId(),
			displayName: user.GetDisplayName(),
		})
	}

	http.SetCookie(w, &http.Cookie{
		Name:     impersonationCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isHTTPS(r),
	})

	view.Detail = buildUserDetail(user)
	view.Impersonation = &templates.ImpersonationView{
		UserID:      user.GetId(),
		DisplayName: user.GetDisplayName(),
	}
	invitesCtx, invitesCancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
	defer invitesCancel()
	h.populateUserInvitesIfImpersonating(invitesCtx, view.Detail, view.Impersonation, loc)
	pageCtx.Impersonation = view.Impersonation
	label := strings.TrimSpace(user.GetDisplayName())
	if label == "" {
		label = user.GetId()
	}
	view.Message = loc.Sprintf("users.impersonate.success", label)

	h.renderUserDetail(w, r, view, pageCtx, loc, "details")
}

// handleLogout clears the current impersonation session.
func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	loc, lang := h.localizer(w, r)
	pageCtx := h.pageContext(lang, loc, r)
	view := templates.UserDetailPageView{}
	if !requireSameOrigin(w, r, loc) {
		return
	}

	if sessionID := impersonationSessionID(r); sessionID != "" {
		if h.impersonation != nil {
			h.impersonation.Delete(sessionID)
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     impersonationCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isHTTPS(r),
	})

	view.Message = loc.Sprintf("users.logout.success")
	pageCtx.Impersonation = nil

	if err := r.ParseForm(); err == nil {
		if userID := strings.TrimSpace(r.FormValue("user_id")); userID != "" {
			ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
			defer cancel()
			detail, message := h.loadUserDetail(ctx, userID, loc)
			view.Detail = detail
			if message != "" && view.Message == "" {
				view.Message = message
			}
			h.populateUserInvitesIfImpersonating(ctx, view.Detail, view.Impersonation, loc)
			h.renderUserDetail(w, r, view, pageCtx, loc, "details")
			return
		}
	}

	templ.Handler(templates.UsersFullPage(templates.UsersPageView{Message: view.Message}, pageCtx)).ServeHTTP(w, r)
}

// handleUsersTable renders the users table via HTMX.
func (h *Handler) handleUsersTable(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.localizer(w, r)
	client := h.authClient()
	if client == nil {
		h.renderUsersTable(w, r, nil, loc.Sprintf("error.user_service_unavailable"), loc)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
	defer cancel()

	response, err := client.ListUsers(ctx, &authv1.ListUsersRequest{PageSize: 50})
	if err != nil {
		log.Printf("list users: %v", err)
		h.renderUsersTable(w, r, nil, loc.Sprintf("error.users_unavailable"), loc)
		return
	}

	users := response.GetUsers()
	if len(users) == 0 {
		h.renderUsersTable(w, r, nil, loc.Sprintf("error.no_users"), loc)
		return
	}

	rows := buildUserRows(users)
	h.renderUsersTable(w, r, rows, "", loc)
}

// handleCampaignsTable returns the first page of campaign rows for HTMX.
func (h *Handler) handleCampaignsTable(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.localizer(w, r)
	campaignClient := h.campaignClient()
	if campaignClient == nil {
		h.renderCampaignTable(w, r, nil, loc.Sprintf("error.campaign_service_unavailable"), loc)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
	defer cancel()

	response, err := campaignClient.ListCampaigns(ctx, &statev1.ListCampaignsRequest{})
	if err != nil {
		log.Printf("list campaigns: %v", err)
		h.renderCampaignTable(w, r, nil, loc.Sprintf("error.campaigns_unavailable"), loc)
		return
	}

	campaigns := response.GetCampaigns()
	if len(campaigns) == 0 {
		h.renderCampaignTable(w, r, nil, loc.Sprintf("error.no_campaigns"), loc)
		return
	}

	rows := buildCampaignRows(campaigns, loc)
	h.renderCampaignTable(w, r, rows, "", loc)
}

// handleCampaignsPage renders the campaigns page fragment or full layout.
func (h *Handler) handleCampaignsPage(w http.ResponseWriter, r *http.Request) {
	loc, lang := h.localizer(w, r)
	pageCtx := h.pageContext(lang, loc, r)
	if isHTMXRequest(r) {
		templ.Handler(templates.CampaignsPage(loc)).ServeHTTP(w, r)
		return
	}

	templ.Handler(templates.CampaignsFullPage(pageCtx)).ServeHTTP(w, r)
}

// handleSystemsPage renders the systems page fragment or full layout.
func (h *Handler) handleSystemsPage(w http.ResponseWriter, r *http.Request) {
	loc, lang := h.localizer(w, r)
	pageCtx := h.pageContext(lang, loc, r)
	if isHTMXRequest(r) {
		templ.Handler(templates.SystemsPage(loc)).ServeHTTP(w, r)
		return
	}

	templ.Handler(templates.SystemsFullPage(pageCtx)).ServeHTTP(w, r)
}

// handleSystemsTable renders the systems table via HTMX.
func (h *Handler) handleSystemsTable(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.localizer(w, r)
	client := h.systemClient()
	if client == nil {
		h.renderSystemsTable(w, r, nil, loc.Sprintf("error.system_service_unavailable"), loc)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
	defer cancel()

	response, err := client.ListGameSystems(ctx, &statev1.ListGameSystemsRequest{})
	if err != nil {
		log.Printf("list game systems: %v", err)
		h.renderSystemsTable(w, r, nil, loc.Sprintf("error.systems_unavailable"), loc)
		return
	}

	systemsList := response.GetSystems()
	if len(systemsList) == 0 {
		h.renderSystemsTable(w, r, nil, loc.Sprintf("error.no_systems"), loc)
		return
	}

	rows := buildSystemRows(systemsList, loc)
	h.renderSystemsTable(w, r, rows, "", loc)
}

// handleSystemRoutes dispatches the system detail route.
func (h *Handler) handleSystemRoutes(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/") {
		canonical := strings.TrimRight(r.URL.Path, "/")
		if canonical == "" {
			canonical = "/"
		}
		http.Redirect(w, r, canonical, http.StatusMovedPermanently)
		return
	}
	systemPath := strings.TrimPrefix(r.URL.Path, "/systems/")
	parts := splitPathParts(systemPath)
	if len(parts) == 1 && strings.TrimSpace(parts[0]) != "" {
		h.handleSystemDetail(w, r, parts[0])
		return
	}
	http.NotFound(w, r)
}

// handleSystemDetail renders the system detail page.
func (h *Handler) handleSystemDetail(w http.ResponseWriter, r *http.Request, systemID string) {
	loc, lang := h.localizer(w, r)
	client := h.systemClient()
	if client == nil {
		h.renderSystemDetail(w, r, templates.SystemDetail{}, loc.Sprintf("error.system_service_unavailable"), lang, loc)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
	defer cancel()

	version := strings.TrimSpace(r.URL.Query().Get("version"))
	parsedID := parseSystemID(systemID)
	if parsedID == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
		h.renderSystemDetail(w, r, templates.SystemDetail{}, loc.Sprintf("error.system_not_found"), lang, loc)
		return
	}
	response, err := client.GetGameSystem(ctx, &statev1.GetGameSystemRequest{
		Id:      parsedID,
		Version: version,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			h.renderSystemDetail(w, r, templates.SystemDetail{}, loc.Sprintf("error.system_not_found"), lang, loc)
			return
		}
		log.Printf("get game system: %v", err)
		h.renderSystemDetail(w, r, templates.SystemDetail{}, loc.Sprintf("error.system_unavailable"), lang, loc)
		return
	}
	if response.GetSystem() == nil {
		h.renderSystemDetail(w, r, templates.SystemDetail{}, loc.Sprintf("error.system_not_found"), lang, loc)
		return
	}

	detail := buildSystemDetail(response.GetSystem(), loc)
	h.renderSystemDetail(w, r, detail, "", lang, loc)
}

func (h *Handler) handleCampaignCreate(w http.ResponseWriter, r *http.Request) {
	loc, lang := h.localizer(w, r)
	pageCtx := h.pageContext(lang, loc, r)
	view := templates.CampaignCreatePageView{
		Impersonation: pageCtx.Impersonation,
		System:        "daggerheart",
		GmMode:        "human",
	}
	if pageCtx.Impersonation != nil {
		view.UserID = pageCtx.Impersonation.UserID
		view.CreatorDisplayName = pageCtx.Impersonation.DisplayName
	}
	renderCreate := func() {
		if isHTMXRequest(r) {
			templ.Handler(templates.CampaignCreatePage(view, loc)).ServeHTTP(w, r)
			return
		}
		templ.Handler(templates.CampaignCreateFullPage(view, pageCtx)).ServeHTTP(w, r)
	}

	switch r.Method {
	case http.MethodGet:
		if message := strings.TrimSpace(r.URL.Query().Get("message")); message != "" {
			view.Message = message
		}
		renderCreate()
		return
	case http.MethodPost:
		if !requireSameOrigin(w, r, loc) {
			return
		}
	default:
		w.Header().Set("Allow", strings.Join([]string{http.MethodGet, http.MethodPost}, ", "))
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		view.Message = loc.Sprintf("error.campaign_create_invalid")
		renderCreate()
		return
	}

	if pageCtx.Impersonation == nil {
		view.UserID = strings.TrimSpace(r.FormValue("user_id"))
		view.CreatorDisplayName = strings.TrimSpace(r.FormValue("creator_display_name"))
	} else {
		view.UserID = pageCtx.Impersonation.UserID
		view.CreatorDisplayName = pageCtx.Impersonation.DisplayName
	}
	view.Name = strings.TrimSpace(r.FormValue("name"))
	view.System = strings.TrimSpace(r.FormValue("system"))
	view.GmMode = strings.TrimSpace(r.FormValue("gm_mode"))
	view.ThemePrompt = strings.TrimSpace(r.FormValue("theme_prompt"))

	if view.UserID == "" {
		view.Message = loc.Sprintf("error.campaign_user_id_required")
		renderCreate()
		return
	}

	if view.Name == "" {
		view.Message = loc.Sprintf("error.campaign_name_required")
		renderCreate()
		return
	}

	system, ok := parseGameSystem(view.System)
	if !ok {
		if view.System == "" {
			view.Message = loc.Sprintf("error.campaign_system_required")
		} else {
			view.Message = loc.Sprintf("error.campaign_system_invalid")
		}
		renderCreate()
		return
	}

	gmMode, ok := parseGmMode(view.GmMode)
	if !ok {
		if view.GmMode == "" {
			view.Message = loc.Sprintf("error.campaign_gm_mode_required")
		} else {
			view.Message = loc.Sprintf("error.campaign_gm_mode_invalid")
		}
		renderCreate()
		return
	}

	client := h.campaignClient()
	if client == nil {
		view.Message = loc.Sprintf("error.campaign_service_unavailable")
		renderCreate()
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
	defer cancel()
	if view.UserID != "" {
		md, _ := metadata.FromOutgoingContext(ctx)
		md = md.Copy()
		md.Set(grpcmeta.UserIDHeader, view.UserID)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	locale := localeFromTag(lang)
	response, err := client.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
		Name:               view.Name,
		Locale:             locale,
		System:             system,
		GmMode:             gmMode,
		ThemePrompt:        view.ThemePrompt,
		CreatorDisplayName: view.CreatorDisplayName,
	})
	if err != nil || response.GetCampaign() == nil {
		log.Printf("create campaign: %v", err)
		view.Message = loc.Sprintf("error.campaign_create_failed")
		renderCreate()
		return
	}

	campaignID := response.GetCampaign().GetId()
	redirectURL := "/campaigns/" + campaignID
	if isHTMXRequest(r) {
		w.Header().Set("Location", redirectURL)
		w.Header().Set("HX-Redirect", redirectURL)
		w.WriteHeader(http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// handleCampaignRoutes dispatches detail and session subroutes.
func (h *Handler) handleCampaignRoutes(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/") {
		canonical := strings.TrimRight(r.URL.Path, "/")
		if canonical == "" {
			canonical = "/"
		}
		http.Redirect(w, r, canonical, http.StatusMovedPermanently)
		return
	}
	campaignPath := strings.TrimPrefix(r.URL.Path, "/campaigns/")
	parts := splitPathParts(campaignPath)

	// /campaigns/{id}/characters
	if len(parts) == 2 && parts[1] == "characters" {
		h.handleCharactersList(w, r, parts[0])
		return
	}
	// /campaigns/{id}/characters/table
	if len(parts) == 3 && parts[1] == "characters" && parts[2] == "table" {
		h.handleCharactersTable(w, r, parts[0])
		return
	}
	// /campaigns/{id}/characters/{characterId}
	if len(parts) == 3 && parts[1] == "characters" {
		h.handleCharacterSheet(w, r, parts[0], parts[2])
		return
	}
	// /campaigns/{id}/characters/{characterId}/activity
	if len(parts) == 4 && parts[1] == "characters" && parts[3] == "activity" {
		h.handleCharacterActivity(w, r, parts[0], parts[2])
		return
	}
	// /campaigns/{id}/participants
	if len(parts) == 2 && parts[1] == "participants" {
		h.handleParticipantsList(w, r, parts[0])
		return
	}
	// /campaigns/{id}/participants/table
	if len(parts) == 3 && parts[1] == "participants" && parts[2] == "table" {
		h.handleParticipantsTable(w, r, parts[0])
		return
	}
	// /campaigns/{id}/invites
	if len(parts) == 2 && parts[1] == "invites" {
		h.handleInvitesList(w, r, parts[0])
		return
	}
	// /campaigns/{id}/invites/table
	if len(parts) == 3 && parts[1] == "invites" && parts[2] == "table" {
		h.handleInvitesTable(w, r, parts[0])
		return
	}
	// /campaigns/{id}/sessions
	if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && parts[1] == "sessions" {
		h.handleSessionsList(w, r, parts[0])
		return
	}
	// /campaigns/{id}/sessions/table
	if len(parts) == 3 && parts[1] == "sessions" && parts[2] == "table" {
		h.handleSessionsTable(w, r, parts[0])
		return
	}
	// /campaigns/{id}/sessions/{sessionId}
	if len(parts) == 3 && parts[1] == "sessions" {
		h.handleSessionDetail(w, r, parts[0], parts[2])
		return
	}
	// /campaigns/{id}/sessions/{sessionId}/events
	if len(parts) == 4 && parts[1] == "sessions" && parts[3] == "events" {
		h.handleSessionEvents(w, r, parts[0], parts[2])
		return
	}
	// /campaigns/{id}/events
	if len(parts) == 2 && parts[1] == "events" {
		h.handleEventLog(w, r, parts[0])
		return
	}
	// /campaigns/{id}/events/table (HTMX fragment)
	if len(parts) == 3 && parts[1] == "events" && parts[2] == "table" {
		h.handleEventLogTable(w, r, parts[0])
		return
	}
	// /campaigns/{id}
	if len(parts) == 1 && strings.TrimSpace(parts[0]) != "" {
		h.handleCampaignDetail(w, r, parts[0])
		return
	}
	http.NotFound(w, r)
}

// handleCampaignDetail renders the single-campaign detail content.
func (h *Handler) handleCampaignDetail(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := h.localizer(w, r)
	campaignClient := h.campaignClient()
	if campaignClient == nil {
		h.renderCampaignDetail(w, r, templates.CampaignDetail{}, loc.Sprintf("error.campaign_service_unavailable"), lang, loc)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
	defer cancel()

	response, err := campaignClient.GetCampaign(ctx, &statev1.GetCampaignRequest{CampaignId: campaignID})
	if err != nil {
		log.Printf("get campaign: %v", err)
		h.renderCampaignDetail(w, r, templates.CampaignDetail{}, loc.Sprintf("error.campaign_unavailable"), lang, loc)
		return
	}

	campaign := response.GetCampaign()
	if campaign == nil {
		h.renderCampaignDetail(w, r, templates.CampaignDetail{}, loc.Sprintf("error.campaign_not_found"), lang, loc)
		return
	}

	detail := buildCampaignDetail(campaign, loc)
	h.renderCampaignDetail(w, r, detail, "", lang, loc)
}

// handleSessionsList renders the sessions list page.
func (h *Handler) handleSessionsList(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := h.localizer(w, r)
	campaignName := getCampaignName(h, r, campaignID, loc)

	if isHTMXRequest(r) {
		templ.Handler(templates.SessionsListPage(campaignID, campaignName, loc)).ServeHTTP(w, r)
		return
	}
	pageCtx := h.pageContext(lang, loc, r)
	templ.Handler(templates.SessionsListFullPage(campaignID, campaignName, pageCtx)).ServeHTTP(w, r)
}

// handleSessionsTable renders the sessions table via HTMX.
func (h *Handler) handleSessionsTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := h.localizer(w, r)
	sessionClient := h.sessionClient()
	if sessionClient == nil {
		h.renderCampaignSessions(w, r, nil, loc.Sprintf("error.session_service_unavailable"), loc)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
	defer cancel()

	response, err := sessionClient.ListSessions(ctx, &statev1.ListSessionsRequest{
		CampaignId: campaignID,
		PageSize:   sessionListPageSize,
	})
	if err != nil {
		log.Printf("list sessions: %v", err)
		h.renderCampaignSessions(w, r, nil, loc.Sprintf("error.sessions_unavailable"), loc)
		return
	}

	sessions := response.GetSessions()
	if len(sessions) == 0 {
		h.renderCampaignSessions(w, r, nil, loc.Sprintf("error.no_sessions"), loc)
		return
	}

	rows := buildCampaignSessionRows(sessions, loc)
	h.renderCampaignSessions(w, r, rows, "", loc)
}

// renderCampaignTable renders a campaign table with optional rows and message.
func (h *Handler) renderCampaignTable(w http.ResponseWriter, r *http.Request, rows []templates.CampaignRow, message string, loc *message.Printer) {
	templ.Handler(templates.CampaignsTable(rows, message, loc)).ServeHTTP(w, r)
}

// renderCampaignDetail renders the campaign detail fragment or full layout.
func (h *Handler) renderCampaignDetail(w http.ResponseWriter, r *http.Request, detail templates.CampaignDetail, message string, lang string, loc *message.Printer) {
	if isHTMXRequest(r) {
		templ.Handler(templates.CampaignDetailPage(detail, message, loc)).ServeHTTP(w, r)
		return
	}

	pageCtx := h.pageContext(lang, loc, r)
	templ.Handler(templates.CampaignDetailFullPage(detail, message, pageCtx)).ServeHTTP(w, r)
}

// renderSystemsTable renders a systems table with optional rows and message.
func (h *Handler) renderSystemsTable(w http.ResponseWriter, r *http.Request, rows []templates.SystemRow, message string, loc *message.Printer) {
	templ.Handler(templates.SystemsTable(rows, message, loc)).ServeHTTP(w, r)
}

// renderSystemDetail renders the system detail fragment or full layout.
func (h *Handler) renderSystemDetail(w http.ResponseWriter, r *http.Request, detail templates.SystemDetail, message string, lang string, loc *message.Printer) {
	if isHTMXRequest(r) {
		templ.Handler(templates.SystemDetailPage(detail, message, loc)).ServeHTTP(w, r)
		return
	}

	pageCtx := h.pageContext(lang, loc, r)
	templ.Handler(templates.SystemDetailFullPage(detail, message, pageCtx)).ServeHTTP(w, r)
}

// renderCampaignSessions renders the session list fragment.
func (h *Handler) renderCampaignSessions(w http.ResponseWriter, r *http.Request, rows []templates.CampaignSessionRow, message string, loc *message.Printer) {
	templ.Handler(templates.CampaignSessionsList(rows, message, loc)).ServeHTTP(w, r)
}

// renderUsersTable renders the users table component.
func (h *Handler) renderUsersTable(w http.ResponseWriter, r *http.Request, rows []templates.UserRow, message string, loc *message.Printer) {
	templ.Handler(templates.UsersTable(rows, message, loc)).ServeHTTP(w, r)
}

func (h *Handler) renderUserDetail(w http.ResponseWriter, r *http.Request, view templates.UserDetailPageView, pageCtx templates.PageContext, loc *message.Printer, activePage string) {
	if isHTMXRequest(r) {
		templ.Handler(templates.UserDetailPage(view, activePage, loc)).ServeHTTP(w, r)
		return
	}

	templ.Handler(templates.UserDetailFullPage(view, activePage, pageCtx)).ServeHTTP(w, r)
}

func (h *Handler) redirectToUserDetail(w http.ResponseWriter, r *http.Request, userID string) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		http.NotFound(w, r)
		return
	}
	redirectURL := "/users/" + userID
	if isHTMXRequest(r) {
		w.Header().Set("Location", redirectURL)
		w.Header().Set("HX-Redirect", redirectURL)
		w.WriteHeader(http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (h *Handler) loadUserDetail(ctx context.Context, userID string, loc *message.Printer) (*templates.UserDetail, string) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, loc.Sprintf("error.user_id_required")
	}
	client := h.authClient()
	if client == nil {
		return nil, loc.Sprintf("error.user_service_unavailable")
	}
	response, err := client.GetUser(ctx, &authv1.GetUserRequest{UserId: userID})
	if err != nil || response.GetUser() == nil {
		log.Printf("get user: %v", err)
		return nil, loc.Sprintf("error.user_not_found")
	}
	detail := buildUserDetail(response.GetUser())
	if detail != nil {
		emails, err := client.ListUserEmails(ctx, &authv1.ListUserEmailsRequest{UserId: userID})
		if err != nil {
			log.Printf("list user emails: %v", err)
		} else {
			detail.Emails = buildUserEmailRows(emails.GetEmails(), loc)
		}
	}
	return detail, ""
}

// authClient returns the currently configured auth client.
func (h *Handler) authClient() authv1.AuthServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.AuthClient()
}

// campaignClient returns the currently configured campaign client.
func (h *Handler) campaignClient() statev1.CampaignServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.CampaignClient()
}

// sessionClient returns the currently configured session client.
func (h *Handler) sessionClient() statev1.SessionServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.SessionClient()
}

// characterClient returns the currently configured character client.
func (h *Handler) characterClient() statev1.CharacterServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.CharacterClient()
}

// participantClient returns the currently configured participant client.
func (h *Handler) participantClient() statev1.ParticipantServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.ParticipantClient()
}

// inviteClient returns the currently configured invite client.
func (h *Handler) inviteClient() statev1.InviteServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.InviteClient()
}

// snapshotClient returns the currently configured snapshot client.
func (h *Handler) snapshotClient() statev1.SnapshotServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.SnapshotClient()
}

// eventClient returns the currently configured event client.
func (h *Handler) eventClient() statev1.EventServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.EventClient()
}

// statisticsClient returns the currently configured statistics client.
func (h *Handler) statisticsClient() statev1.StatisticsServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.StatisticsClient()
}

// systemClient returns the currently configured system client.
func (h *Handler) systemClient() statev1.SystemServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.SystemClient()
}

// isHTMXRequest reports whether the request originated from HTMX.
func isHTMXRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	return strings.EqualFold(r.Header.Get("HX-Request"), "true")
}

// splitPathParts returns non-empty path segments.
func splitPathParts(path string) []string {
	rawParts := strings.Split(path, "/")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		parts = append(parts, trimmed)
	}
	return parts
}

// buildCampaignRows formats campaign rows for the table.
func buildCampaignRows(campaigns []*statev1.Campaign, loc *message.Printer) []templates.CampaignRow {
	rows := make([]templates.CampaignRow, 0, len(campaigns))
	for _, campaign := range campaigns {
		if campaign == nil {
			continue
		}
		rows = append(rows, templates.CampaignRow{
			ID:               campaign.GetId(),
			Name:             campaign.GetName(),
			System:           formatGameSystem(campaign.GetSystem(), loc),
			GMMode:           formatGmMode(campaign.GetGmMode(), loc),
			ParticipantCount: strconv.FormatInt(int64(campaign.GetParticipantCount()), 10),
			CharacterCount:   strconv.FormatInt(int64(campaign.GetCharacterCount()), 10),
			ThemePrompt:      truncateText(campaign.GetThemePrompt(), campaignThemePromptLimit),
			CreatedDate:      formatCreatedDate(campaign.GetCreatedAt()),
		})
	}
	return rows
}

// buildSystemRows formats system rows for the systems table.
func buildSystemRows(systemsList []*statev1.GameSystemInfo, loc *message.Printer) []templates.SystemRow {
	rows := make([]templates.SystemRow, 0, len(systemsList))
	for _, system := range systemsList {
		if system == nil {
			continue
		}
		detailURL := "/systems/" + system.GetId().String()
		version := strings.TrimSpace(system.GetVersion())
		if version != "" {
			detailURL = detailURL + "?version=" + url.QueryEscape(version)
		}
		rows = append(rows, templates.SystemRow{
			Name:                system.GetName(),
			Version:             version,
			ImplementationStage: formatImplementationStage(system.GetImplementationStage(), loc),
			OperationalStatus:   formatOperationalStatus(system.GetOperationalStatus(), loc),
			AccessLevel:         formatAccessLevel(system.GetAccessLevel(), loc),
			IsDefault:           system.GetIsDefault(),
			DetailURL:           detailURL,
		})
	}
	return rows
}

// buildCampaignDetail formats a campaign into detail view data.
func buildCampaignDetail(campaign *statev1.Campaign, loc *message.Printer) templates.CampaignDetail {
	if campaign == nil {
		return templates.CampaignDetail{}
	}
	return templates.CampaignDetail{
		ID:               campaign.GetId(),
		Name:             campaign.GetName(),
		System:           formatGameSystem(campaign.GetSystem(), loc),
		GMMode:           formatGmMode(campaign.GetGmMode(), loc),
		ParticipantCount: strconv.FormatInt(int64(campaign.GetParticipantCount()), 10),
		CharacterCount:   strconv.FormatInt(int64(campaign.GetCharacterCount()), 10),
		ThemePrompt:      campaign.GetThemePrompt(),
		CreatedAt:        formatTimestamp(campaign.GetCreatedAt()),
		UpdatedAt:        formatTimestamp(campaign.GetUpdatedAt()),
	}
}

// buildSystemDetail formats a system into detail view data.
func buildSystemDetail(system *statev1.GameSystemInfo, loc *message.Printer) templates.SystemDetail {
	if system == nil {
		return templates.SystemDetail{}
	}
	return templates.SystemDetail{
		ID:                  system.GetId().String(),
		Name:                system.GetName(),
		Version:             system.GetVersion(),
		ImplementationStage: formatImplementationStage(system.GetImplementationStage(), loc),
		OperationalStatus:   formatOperationalStatus(system.GetOperationalStatus(), loc),
		AccessLevel:         formatAccessLevel(system.GetAccessLevel(), loc),
		IsDefault:           system.GetIsDefault(),
	}
}

// buildCampaignSessionRows formats session rows for the detail view.
func buildCampaignSessionRows(sessions []*statev1.Session, loc *message.Printer) []templates.CampaignSessionRow {
	rows := make([]templates.CampaignSessionRow, 0, len(sessions))
	for _, session := range sessions {
		if session == nil {
			continue
		}
		statusBadge := "secondary"
		if session.GetStatus() == statev1.SessionStatus_SESSION_ACTIVE {
			statusBadge = "success"
		}
		row := templates.CampaignSessionRow{
			ID:          session.GetId(),
			CampaignID:  session.GetCampaignId(),
			Name:        session.GetName(),
			Status:      formatSessionStatus(session.GetStatus(), loc),
			StatusBadge: statusBadge,
			StartedAt:   formatTimestamp(session.GetStartedAt()),
		}
		if session.GetEndedAt() != nil {
			row.EndedAt = formatTimestamp(session.GetEndedAt())
		}
		rows = append(rows, row)
	}
	return rows
}

// buildUserRows formats user rows for the table.
func buildUserRows(users []*authv1.User) []templates.UserRow {
	rows := make([]templates.UserRow, 0, len(users))
	for _, u := range users {
		if u == nil {
			continue
		}
		rows = append(rows, templates.UserRow{
			ID:          u.GetId(),
			DisplayName: u.GetDisplayName(),
			CreatedAt:   formatTimestamp(u.GetCreatedAt()),
			UpdatedAt:   formatTimestamp(u.GetUpdatedAt()),
		})
	}
	return rows
}

// buildUserDetail formats a user detail view.
func buildUserDetail(u *authv1.User) *templates.UserDetail {
	if u == nil {
		return nil
	}
	return &templates.UserDetail{
		ID:          u.GetId(),
		DisplayName: u.GetDisplayName(),
		CreatedAt:   formatTimestamp(u.GetCreatedAt()),
		UpdatedAt:   formatTimestamp(u.GetUpdatedAt()),
	}
}

func buildUserEmailRows(emails []*authv1.UserEmail, loc *message.Printer) []templates.UserEmailRow {
	rows := make([]templates.UserEmailRow, 0, len(emails))
	for _, email := range emails {
		if email == nil {
			continue
		}
		verified := "-"
		if email.GetVerifiedAt() != nil {
			verified = formatTimestamp(email.GetVerifiedAt())
		}
		rows = append(rows, templates.UserEmailRow{
			Email:      email.GetEmail(),
			VerifiedAt: verified,
			CreatedAt:  formatTimestamp(email.GetCreatedAt()),
			UpdatedAt:  formatTimestamp(email.GetUpdatedAt()),
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return rows
}

func (h *Handler) populateUserInvites(ctx context.Context, detail *templates.UserDetail, loc *message.Printer) {
	if detail == nil {
		return
	}
	rows, message := h.listPendingInvitesForUser(ctx, detail.ID, loc)
	detail.PendingInvites = rows
	detail.PendingInvitesMessage = message
}

func (h *Handler) populateUserInvitesIfImpersonating(ctx context.Context, detail *templates.UserDetail, impersonation *templates.ImpersonationView, loc *message.Printer) {
	if detail == nil {
		return
	}
	if impersonation == nil || strings.TrimSpace(impersonation.UserID) == "" || impersonation.UserID != detail.ID {
		detail.PendingInvites = nil
		detail.PendingInvitesMessage = loc.Sprintf("users.invites.require_impersonation")
		return
	}
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	md.Set(grpcmeta.UserIDHeader, impersonation.UserID)
	ctx = metadata.NewOutgoingContext(ctx, md)
	h.populateUserInvites(ctx, detail, loc)
}

func (h *Handler) listPendingInvitesForUser(ctx context.Context, userID string, loc *message.Printer) ([]templates.InviteRow, string) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, loc.Sprintf("users.invites.empty")
	}
	inviteClient := h.inviteClient()
	if inviteClient == nil {
		return nil, loc.Sprintf("error.pending_invites_unavailable")
	}

	rows := make([]templates.InviteRow, 0)
	pageToken := ""
	for {
		resp, err := inviteClient.ListPendingInvitesForUser(ctx, &statev1.ListPendingInvitesForUserRequest{
			PageSize:  inviteListPageSize,
			PageToken: pageToken,
		})
		if err != nil {
			log.Printf("list pending invites for user: %v", err)
			return nil, loc.Sprintf("error.pending_invites_unavailable")
		}
		for _, pending := range resp.GetInvites() {
			if pending == nil {
				continue
			}
			inv := pending.GetInvite()
			campaign := pending.GetCampaign()
			participant := pending.GetParticipant()

			campaignID := strings.TrimSpace(campaign.GetId())
			if campaignID == "" && inv != nil {
				campaignID = strings.TrimSpace(inv.GetCampaignId())
			}
			campaignName := strings.TrimSpace(campaign.GetName())
			if campaignName == "" {
				if campaignID != "" {
					campaignName = campaignID
				} else {
					campaignName = loc.Sprintf("label.unknown")
				}
			}

			participantLabel := strings.TrimSpace(participant.GetDisplayName())
			if participantLabel == "" {
				participantLabel = loc.Sprintf("label.unknown")
			}

			inviteID := ""
			status := statev1.InviteStatus_INVITE_STATUS_UNSPECIFIED
			createdAt := ""
			if inv != nil {
				inviteID = inv.GetId()
				status = inv.GetStatus()
				createdAt = formatTimestamp(inv.GetCreatedAt())
			}
			statusLabel, statusVariant := formatInviteStatus(status, loc)

			rows = append(rows, templates.InviteRow{
				ID:            inviteID,
				CampaignID:    campaignID,
				CampaignName:  campaignName,
				Participant:   participantLabel,
				Status:        statusLabel,
				StatusVariant: statusVariant,
				CreatedAt:     createdAt,
			})
		}
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}

	if len(rows) == 0 {
		return nil, loc.Sprintf("users.invites.empty")
	}
	return rows, ""
}

// formatGmMode returns a display label for a GM mode enum.
func formatGmMode(mode statev1.GmMode, loc *message.Printer) string {
	switch mode {
	case statev1.GmMode_HUMAN:
		return loc.Sprintf("label.human")
	case statev1.GmMode_AI:
		return loc.Sprintf("label.ai")
	case statev1.GmMode_HYBRID:
		return loc.Sprintf("label.hybrid")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func formatGameSystem(system commonv1.GameSystem, loc *message.Printer) string {
	switch system {
	case commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART:
		return loc.Sprintf("label.daggerheart")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func formatImplementationStage(stage commonv1.GameSystemImplementationStage, loc *message.Printer) string {
	switch stage {
	case commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_PLANNED:
		return loc.Sprintf("label.system_stage_planned")
	case commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_PARTIAL:
		return loc.Sprintf("label.system_stage_partial")
	case commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_COMPLETE:
		return loc.Sprintf("label.system_stage_complete")
	case commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_DEPRECATED:
		return loc.Sprintf("label.system_stage_deprecated")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func formatOperationalStatus(status commonv1.GameSystemOperationalStatus, loc *message.Printer) string {
	switch status {
	case commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OFFLINE:
		return loc.Sprintf("label.system_status_offline")
	case commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_DEGRADED:
		return loc.Sprintf("label.system_status_degraded")
	case commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OPERATIONAL:
		return loc.Sprintf("label.system_status_operational")
	case commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_MAINTENANCE:
		return loc.Sprintf("label.system_status_maintenance")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func formatAccessLevel(level commonv1.GameSystemAccessLevel, loc *message.Printer) string {
	switch level {
	case commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_INTERNAL:
		return loc.Sprintf("label.system_access_internal")
	case commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_BETA:
		return loc.Sprintf("label.system_access_beta")
	case commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_PUBLIC:
		return loc.Sprintf("label.system_access_public")
	case commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_RETIRED:
		return loc.Sprintf("label.system_access_retired")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func parseSystemID(value string) commonv1.GameSystem {
	trimmed := strings.ToUpper(strings.TrimSpace(value))
	if trimmed == "" {
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED
	}
	if trimmed == "DAGGERHEART" {
		return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART
	}
	if enumValue, ok := commonv1.GameSystem_value[trimmed]; ok {
		return commonv1.GameSystem(enumValue)
	}
	return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED
}

func parseGameSystem(value string) (commonv1.GameSystem, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "daggerheart":
		return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, true
	default:
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, false
	}
}

func parseGmMode(value string) (statev1.GmMode, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "human":
		return statev1.GmMode_HUMAN, true
	case "ai":
		return statev1.GmMode_AI, true
	case "hybrid":
		return statev1.GmMode_HYBRID, true
	default:
		return statev1.GmMode_GM_MODE_UNSPECIFIED, false
	}
}

// formatSessionStatus returns a display label for a session status.
func formatSessionStatus(status statev1.SessionStatus, loc *message.Printer) string {
	switch status {
	case statev1.SessionStatus_SESSION_ACTIVE:
		return loc.Sprintf("label.active")
	case statev1.SessionStatus_SESSION_ENDED:
		return loc.Sprintf("label.ended")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func formatInviteStatus(status statev1.InviteStatus, loc *message.Printer) (string, string) {
	switch status {
	case statev1.InviteStatus_PENDING:
		return loc.Sprintf("label.invite_pending"), "warning"
	case statev1.InviteStatus_CLAIMED:
		return loc.Sprintf("label.invite_claimed"), "success"
	case statev1.InviteStatus_REVOKED:
		return loc.Sprintf("label.invite_revoked"), "error"
	default:
		return loc.Sprintf("label.unspecified"), "secondary"
	}
}

// formatCreatedDate returns a YYYY-MM-DD string for a timestamp.
func formatCreatedDate(createdAt *timestamppb.Timestamp) string {
	if createdAt == nil {
		return ""
	}
	return createdAt.AsTime().Format("2006-01-02")
}

// formatTimestamp returns a YYYY-MM-DD HH:MM:SS string for a timestamp.
func formatTimestamp(value *timestamppb.Timestamp) string {
	if value == nil {
		return ""
	}
	return value.AsTime().Format("2006-01-02 15:04:05")
}

// truncateText shortens text to a maximum length with an ellipsis.
func truncateText(text string, limit int) string {
	if limit <= 0 || text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "..."
}

// handleDashboard renders the dashboard page.
func (h *Handler) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	loc, lang := h.localizer(w, r)
	pageCtx := h.pageContext(lang, loc, r)
	if isHTMXRequest(r) {
		templ.Handler(templates.DashboardPage(loc)).ServeHTTP(w, r)
		return
	}
	templ.Handler(templates.DashboardFullPage(pageCtx)).ServeHTTP(w, r)
}

// handleDashboardContent loads and renders the dashboard statistics and recent activity.
func (h *Handler) handleDashboardContent(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
	defer cancel()
	loc, _ := h.localizer(w, r)

	stats := templates.DashboardStats{
		TotalSystems:      "0",
		TotalCampaigns:    "0",
		TotalSessions:     "0",
		TotalCharacters:   "0",
		TotalParticipants: "0",
		TotalUsers:        "0",
	}

	var activities []templates.ActivityEvent

	if statisticsClient := h.statisticsClient(); statisticsClient != nil {
		resp, err := statisticsClient.GetGameStatistics(ctx, &statev1.GetGameStatisticsRequest{})
		if err == nil && resp != nil && resp.GetStats() != nil {
			stats.TotalCampaigns = strconv.FormatInt(resp.GetStats().GetCampaignCount(), 10)
			stats.TotalSessions = strconv.FormatInt(resp.GetStats().GetSessionCount(), 10)
			stats.TotalCharacters = strconv.FormatInt(resp.GetStats().GetCharacterCount(), 10)
			stats.TotalParticipants = strconv.FormatInt(resp.GetStats().GetParticipantCount(), 10)
		}
	}

	if systemClient := h.systemClient(); systemClient != nil {
		systemsResp, err := systemClient.ListGameSystems(ctx, &statev1.ListGameSystemsRequest{})
		if err == nil && systemsResp != nil {
			stats.TotalSystems = strconv.FormatInt(int64(len(systemsResp.GetSystems())), 10)
		}
	}

	if authClient := h.authClient(); authClient != nil {
		var totalUsers int64
		pageToken := ""
		ok := true
		for {
			resp, err := authClient.ListUsers(ctx, &authv1.ListUsersRequest{
				PageSize:  50,
				PageToken: pageToken,
			})
			if err != nil || resp == nil {
				log.Printf("list users for dashboard: %v", err)
				ok = false
				break
			}
			totalUsers += int64(len(resp.GetUsers()))
			pageToken = strings.TrimSpace(resp.GetNextPageToken())
			if pageToken == "" {
				break
			}
		}
		if ok {
			stats.TotalUsers = strconv.FormatInt(totalUsers, 10)
		}
	}

	// Fetch recent activity (last 15 events across all campaigns)
	if eventClient := h.eventClient(); eventClient != nil {
		if campaignClient := h.campaignClient(); campaignClient != nil {
			campaignsResp, err := campaignClient.ListCampaigns(ctx, &statev1.ListCampaignsRequest{})
			if err == nil && campaignsResp != nil {
				// Get events from each campaign and merge
				allEvents := make([]struct {
					event        *statev1.Event
					campaignName string
				}, 0)

				for _, campaign := range campaignsResp.GetCampaigns() {
					if campaign == nil {
						continue
					}
					eventsResp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
						CampaignId: campaign.GetId(),
						PageSize:   5, // Get top 5 from each campaign
						OrderBy:    "seq desc",
					})
					if err == nil && eventsResp != nil {
						for _, event := range eventsResp.GetEvents() {
							if event != nil {
								allEvents = append(allEvents, struct {
									event        *statev1.Event
									campaignName string
								}{event, campaign.GetName()})
							}
						}
					}
				}

				// Sort by timestamp descending and take top 15
				// Simple bubble sort for small datasets
				for i := 0; i < len(allEvents); i++ {
					for j := i + 1; j < len(allEvents); j++ {
						iTs := allEvents[i].event.GetTs()
						jTs := allEvents[j].event.GetTs()
						if iTs != nil && jTs != nil && iTs.AsTime().Before(jTs.AsTime()) {
							allEvents[i], allEvents[j] = allEvents[j], allEvents[i]
						}
					}
				}

				maxEvents := 15
				if len(allEvents) < maxEvents {
					maxEvents = len(allEvents)
				}

				for i := 0; i < maxEvents; i++ {
					evt := allEvents[i].event
					activities = append(activities, templates.ActivityEvent{
						CampaignID:   evt.GetCampaignId(),
						CampaignName: allEvents[i].campaignName,
						EventType:    formatEventType(evt.GetType(), loc),
						Timestamp:    formatTimestamp(evt.GetTs()),
						Description:  formatEventDescription(evt, loc),
					})
				}
			}
		}
	}

	templ.Handler(templates.DashboardContent(stats, activities, loc)).ServeHTTP(w, r)
}

// formatEventType returns a display label for an event type string.
func formatEventType(eventType string, loc *message.Printer) string {
	switch eventType {
	// Campaign events
	case "campaign.created":
		return loc.Sprintf("event.campaign_created")
	case "campaign.forked":
		return loc.Sprintf("event.campaign_forked")
	case "campaign.updated":
		return loc.Sprintf("event.campaign_updated")
	// Participant events
	case "participant.joined":
		return loc.Sprintf("event.participant_joined")
	case "participant.left":
		return loc.Sprintf("event.participant_left")
	case "participant.updated":
		return loc.Sprintf("event.participant_updated")
	// Character events
	case "character.created":
		return loc.Sprintf("event.character_created")
	case "character.deleted":
		return loc.Sprintf("event.character_deleted")
	case "character.updated":
		return loc.Sprintf("event.character_updated")
	case "character.profile_updated":
		return loc.Sprintf("event.character_profile_updated")
	// Session events
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
	// Invite events
	case "invite.created":
		return loc.Sprintf("event.invite_created")
	case "invite.updated":
		return loc.Sprintf("event.invite_updated")
	// Action events
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
		// Fallback: capitalize and format unknown types
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

// formatActorType returns a display label for an actor type string.
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

// formatEventDescription generates a human-readable event description.
func formatEventDescription(event *statev1.Event, loc *message.Printer) string {
	if event == nil {
		return ""
	}
	return formatEventType(event.GetType(), loc)
}

func localeFromTag(tag string) commonv1.Locale {
	if locale, ok := platformi18n.ParseLocale(tag); ok {
		return locale
	}
	return platformi18n.DefaultLocale()
}

// handleCharactersList renders the characters list page.
func (h *Handler) handleCharactersList(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := h.localizer(w, r)
	campaignName := getCampaignName(h, r, campaignID, loc)

	if isHTMXRequest(r) {
		templ.Handler(templates.CharactersListPage(campaignID, campaignName, loc)).ServeHTTP(w, r)
		return
	}
	pageCtx := h.pageContext(lang, loc, r)
	templ.Handler(templates.CharactersListFullPage(campaignID, campaignName, pageCtx)).ServeHTTP(w, r)
}

// handleCharactersTable renders the characters table.
func (h *Handler) handleCharactersTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := h.localizer(w, r)
	characterClient := h.characterClient()
	if characterClient == nil {
		h.renderCharactersTable(w, r, nil, loc.Sprintf("error.character_service_unavailable"), loc)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
	defer cancel()

	// Get characters
	response, err := characterClient.ListCharacters(ctx, &statev1.ListCharactersRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		log.Printf("list characters: %v", err)
		h.renderCharactersTable(w, r, nil, loc.Sprintf("error.characters_unavailable"), loc)
		return
	}

	characters := response.GetCharacters()
	if len(characters) == 0 {
		h.renderCharactersTable(w, r, nil, loc.Sprintf("error.no_characters"), loc)
		return
	}

	participantNames := map[string]string{}
	if participantClient := h.participantClient(); participantClient != nil {
		participantsResp, err := participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
			CampaignId: campaignID,
		})
		if err != nil {
			log.Printf("list participants for character table: %v", err)
		} else {
			for _, participant := range participantsResp.GetParticipants() {
				if participant != nil {
					participantNames[participant.GetId()] = participant.GetDisplayName()
				}
			}
		}
	}

	rows := buildCharacterRows(characters, participantNames, loc)
	h.renderCharactersTable(w, r, rows, "", loc)
}

// handleCharacterSheet renders the character sheet page.
func (h *Handler) handleCharacterSheet(w http.ResponseWriter, r *http.Request, campaignID string, characterID string) {
	h.renderCharacterSheet(w, r, campaignID, characterID, "info")
}

// handleCharacterActivity renders the character activity tab.
func (h *Handler) handleCharacterActivity(w http.ResponseWriter, r *http.Request, campaignID string, characterID string) {
	h.renderCharacterSheet(w, r, campaignID, characterID, "activity")
}

func (h *Handler) renderCharacterSheet(w http.ResponseWriter, r *http.Request, campaignID string, characterID string, activePage string) {
	loc, lang := h.localizer(w, r)
	characterClient := h.characterClient()
	if characterClient == nil {
		http.Error(w, loc.Sprintf("error.character_service_unavailable"), http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
	defer cancel()

	response, err := characterClient.GetCharacterSheet(ctx, &statev1.GetCharacterSheetRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		log.Printf("get character sheet: %v", err)
		http.Error(w, loc.Sprintf("error.character_unavailable"), http.StatusNotFound)
		return
	}

	character := response.GetCharacter()
	if character == nil {
		http.Error(w, loc.Sprintf("error.character_not_found"), http.StatusNotFound)
		return
	}

	campaignName := getCampaignName(h, r, campaignID, loc)

	var recentEvents []templates.EventRow
	if eventClient := h.eventClient(); eventClient != nil {
		eventsResp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   20,
			OrderBy:    "seq desc",
			Filter:     "entity_id = \"" + characterID + "\"",
		})
		if err == nil && eventsResp != nil {
			for _, event := range eventsResp.GetEvents() {
				if event != nil {
					recentEvents = append(recentEvents, templates.EventRow{
						Seq:         event.GetSeq(),
						Type:        formatEventType(event.GetType(), loc),
						Timestamp:   formatTimestamp(event.GetTs()),
						Description: formatEventDescription(event, loc),
						PayloadJSON: string(event.GetPayloadJson()),
					})
				}
			}
		}
	}

	controller := loc.Sprintf("label.unassigned")
	participantID := ""
	if character.GetParticipantId() != nil {
		participantID = strings.TrimSpace(character.GetParticipantId().GetValue())
	}
	if participantID != "" {
		if participantClient := h.participantClient(); participantClient != nil {
			participantResp, err := participantClient.GetParticipant(ctx, &statev1.GetParticipantRequest{
				CampaignId:    campaignID,
				ParticipantId: participantID,
			})
			if err != nil {
				log.Printf("get participant for character sheet: %v", err)
				controller = loc.Sprintf("label.unknown")
			} else if participant := participantResp.GetParticipant(); participant != nil {
				controller = participant.GetDisplayName()
			} else {
				controller = loc.Sprintf("label.unknown")
			}
		} else {
			controller = loc.Sprintf("label.unknown")
		}
	}

	sheet := buildCharacterSheet(campaignID, campaignName, character, recentEvents, controller, loc)

	if isHTMXRequest(r) {
		templ.Handler(templates.CharacterSheetPage(sheet, activePage, loc)).ServeHTTP(w, r)
		return
	}
	pageCtx := h.pageContext(lang, loc, r)
	templ.Handler(templates.CharacterSheetFullPage(sheet, activePage, pageCtx)).ServeHTTP(w, r)
}

// renderCharactersTable renders the characters table component.
func (h *Handler) renderCharactersTable(w http.ResponseWriter, r *http.Request, rows []templates.CharacterRow, message string, loc *message.Printer) {
	templ.Handler(templates.CharactersTable(rows, message, loc)).ServeHTTP(w, r)
}

// buildCharacterRows formats character rows for the table.
func buildCharacterRows(characters []*statev1.Character, participantNames map[string]string, loc *message.Printer) []templates.CharacterRow {
	rows := make([]templates.CharacterRow, 0, len(characters))
	for _, character := range characters {
		if character == nil {
			continue
		}

		controller := formatCharacterController(character, participantNames, loc)

		rows = append(rows, templates.CharacterRow{
			ID:         character.GetId(),
			CampaignID: character.GetCampaignId(),
			Name:       character.GetName(),
			Kind:       formatCharacterKind(character.GetKind(), loc),
			Controller: controller,
		})
	}
	return rows
}

// buildCharacterSheet formats character sheet data.
func buildCharacterSheet(campaignID, campaignName string, character *statev1.Character, recentEvents []templates.EventRow, controller string, loc *message.Printer) templates.CharacterSheetView {
	return templates.CharacterSheetView{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		Character:    character,
		Controller:   controller,
		CreatedAt:    formatTimestamp(character.GetCreatedAt()),
		UpdatedAt:    formatTimestamp(character.GetUpdatedAt()),
		RecentEvents: recentEvents,
	}
}

func formatCharacterController(character *statev1.Character, participantNames map[string]string, loc *message.Printer) string {
	if character == nil {
		return loc.Sprintf("label.unassigned")
	}
	participantID := ""
	if character.GetParticipantId() != nil {
		participantID = strings.TrimSpace(character.GetParticipantId().GetValue())
	}
	if participantID == "" {
		return loc.Sprintf("label.unassigned")
	}
	if name, ok := participantNames[participantID]; ok {
		return name
	}
	return loc.Sprintf("label.unknown")
}

// formatCharacterKind returns a display label for a character kind.
func formatCharacterKind(kind statev1.CharacterKind, loc *message.Printer) string {
	switch kind {
	case statev1.CharacterKind_PC:
		return loc.Sprintf("label.pc")
	case statev1.CharacterKind_NPC:
		return loc.Sprintf("label.npc")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

// getCampaignName fetches the campaign name by ID.
func getCampaignName(h *Handler, r *http.Request, campaignID string, loc *message.Printer) string {
	campaignClient := h.campaignClient()
	if campaignClient == nil {
		return loc.Sprintf("label.campaign")
	}

	ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
	defer cancel()

	response, err := campaignClient.GetCampaign(ctx, &statev1.GetCampaignRequest{CampaignId: campaignID})
	if err != nil || response == nil || response.GetCampaign() == nil {
		return loc.Sprintf("label.campaign")
	}

	return response.GetCampaign().GetName()
}

// handleParticipantsList renders the participants list page.
func (h *Handler) handleParticipantsList(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := h.localizer(w, r)
	campaignName := getCampaignName(h, r, campaignID, loc)

	if isHTMXRequest(r) {
		templ.Handler(templates.ParticipantsListPage(campaignID, campaignName, loc)).ServeHTTP(w, r)
		return
	}
	pageCtx := h.pageContext(lang, loc, r)
	templ.Handler(templates.ParticipantsListFullPage(campaignID, campaignName, pageCtx)).ServeHTTP(w, r)
}

// handleParticipantsTable renders the participants table.
func (h *Handler) handleParticipantsTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := h.localizer(w, r)
	participantClient := h.participantClient()
	if participantClient == nil {
		h.renderParticipantsTable(w, r, nil, loc.Sprintf("error.participant_service_unavailable"), loc)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
	defer cancel()

	response, err := participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		log.Printf("list participants: %v", err)
		h.renderParticipantsTable(w, r, nil, loc.Sprintf("error.participants_unavailable"), loc)
		return
	}

	participants := response.GetParticipants()
	if len(participants) == 0 {
		h.renderParticipantsTable(w, r, nil, loc.Sprintf("error.no_participants"), loc)
		return
	}

	rows := buildParticipantRows(participants, loc)
	h.renderParticipantsTable(w, r, rows, "", loc)
}

// handleInvitesList renders the invites list page.
func (h *Handler) handleInvitesList(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := h.localizer(w, r)
	campaignName := getCampaignName(h, r, campaignID, loc)

	if isHTMXRequest(r) {
		templ.Handler(templates.InvitesListPage(campaignID, campaignName, loc)).ServeHTTP(w, r)
		return
	}
	pageCtx := h.pageContext(lang, loc, r)
	templ.Handler(templates.InvitesListFullPage(campaignID, campaignName, pageCtx)).ServeHTTP(w, r)
}

// handleInvitesTable renders the invites table.
func (h *Handler) handleInvitesTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := h.localizer(w, r)
	inviteClient := h.inviteClient()
	if inviteClient == nil {
		h.renderInvitesTable(w, r, nil, loc.Sprintf("error.invite_service_unavailable"), loc)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
	defer cancel()

	if impersonation := h.currentImpersonation(r); impersonation != nil {
		participantID, err := h.resolveParticipantIDForUser(ctx, campaignID, impersonation.userID)
		if err != nil {
			log.Printf("resolve participant for invites: %v", err)
		}
		if participantID != "" {
			md, _ := metadata.FromOutgoingContext(ctx)
			md = md.Copy()
			md.Set(grpcmeta.ParticipantIDHeader, participantID)
			ctx = metadata.NewOutgoingContext(ctx, md)
		}
	}

	response, err := inviteClient.ListInvites(ctx, &statev1.ListInvitesRequest{
		CampaignId: campaignID,
		PageSize:   inviteListPageSize,
	})
	if err != nil {
		log.Printf("list invites: %v", err)
		h.renderInvitesTable(w, r, nil, loc.Sprintf("error.invites_unavailable"), loc)
		return
	}

	invites := response.GetInvites()
	if len(invites) == 0 {
		h.renderInvitesTable(w, r, nil, loc.Sprintf("error.no_invites"), loc)
		return
	}

	participantNames := map[string]string{}
	if participantClient := h.participantClient(); participantClient != nil {
		participantsResp, err := participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
			CampaignId: campaignID,
		})
		if err != nil {
			log.Printf("list participants for invites: %v", err)
		} else {
			for _, participant := range participantsResp.GetParticipants() {
				if participant != nil {
					participantNames[participant.GetId()] = participant.GetDisplayName()
				}
			}
		}
	}

	recipientNames := map[string]string{}
	if authClient := h.authClient(); authClient != nil {
		for _, inv := range invites {
			if inv == nil {
				continue
			}
			recipientID := strings.TrimSpace(inv.GetRecipientUserId())
			if recipientID == "" {
				continue
			}
			if _, ok := recipientNames[recipientID]; ok {
				continue
			}
			userResp, err := authClient.GetUser(ctx, &authv1.GetUserRequest{UserId: recipientID})
			if err != nil {
				log.Printf("get invite recipient: %v", err)
				recipientNames[recipientID] = ""
				continue
			}
			if user := userResp.GetUser(); user != nil {
				recipientNames[recipientID] = user.GetDisplayName()
			}
		}
	}

	rows := buildInviteRows(invites, participantNames, recipientNames, loc)
	h.renderInvitesTable(w, r, rows, "", loc)
}

func (h *Handler) resolveParticipantIDForUser(ctx context.Context, campaignID, userID string) (string, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", nil
	}
	participantClient := h.participantClient()
	if participantClient == nil {
		return "", fmt.Errorf("participant client unavailable")
	}
	pageToken := ""
	for {
		resp, err := participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
			CampaignId: campaignID,
			PageSize:   100,
			PageToken:  pageToken,
		})
		if err != nil {
			return "", err
		}
		for _, participant := range resp.GetParticipants() {
			if participant == nil {
				continue
			}
			if strings.TrimSpace(participant.GetUserId()) == userID {
				return participant.GetId(), nil
			}
		}
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}
	return "", nil
}

// renderInvitesTable renders the invites table component.
func (h *Handler) renderInvitesTable(w http.ResponseWriter, r *http.Request, rows []templates.InviteRow, message string, loc *message.Printer) {
	templ.Handler(templates.InvitesTable(rows, message, loc)).ServeHTTP(w, r)
}

// buildInviteRows formats invite rows for the table.
func buildInviteRows(invites []*statev1.Invite, participantNames map[string]string, recipientNames map[string]string, loc *message.Printer) []templates.InviteRow {
	rows := make([]templates.InviteRow, 0, len(invites))
	for _, inv := range invites {
		if inv == nil {
			continue
		}

		participantLabel := participantNames[inv.GetParticipantId()]
		if participantLabel == "" {
			participantLabel = loc.Sprintf("label.unknown")
		}

		recipientLabel := loc.Sprintf("label.unassigned")
		recipientID := strings.TrimSpace(inv.GetRecipientUserId())
		if recipientID != "" {
			recipientLabel = recipientNames[recipientID]
			if recipientLabel == "" {
				recipientLabel = recipientID
			}
		}

		statusLabel, statusVariant := formatInviteStatus(inv.GetStatus(), loc)

		rows = append(rows, templates.InviteRow{
			ID:            inv.GetId(),
			CampaignID:    inv.GetCampaignId(),
			Participant:   participantLabel,
			Recipient:     recipientLabel,
			Status:        statusLabel,
			StatusVariant: statusVariant,
			CreatedAt:     formatTimestamp(inv.GetCreatedAt()),
			UpdatedAt:     formatTimestamp(inv.GetUpdatedAt()),
		})
	}
	return rows
}

// renderParticipantsTable renders the participants table component.
func (h *Handler) renderParticipantsTable(w http.ResponseWriter, r *http.Request, rows []templates.ParticipantRow, message string, loc *message.Printer) {
	templ.Handler(templates.ParticipantsTable(rows, message, loc)).ServeHTTP(w, r)
}

// buildParticipantRows formats participant rows for the table.
func buildParticipantRows(participants []*statev1.Participant, loc *message.Printer) []templates.ParticipantRow {
	rows := make([]templates.ParticipantRow, 0, len(participants))
	for _, participant := range participants {
		if participant == nil {
			continue
		}

		role, roleVariant := formatParticipantRole(participant.GetRole(), loc)
		access, accessVariant := formatParticipantAccess(participant.GetCampaignAccess(), loc)
		controller, controllerVariant := formatParticipantController(participant.GetController(), loc)

		rows = append(rows, templates.ParticipantRow{
			ID:                participant.GetId(),
			DisplayName:       participant.GetDisplayName(),
			Role:              role,
			RoleVariant:       roleVariant,
			Access:            access,
			AccessVariant:     accessVariant,
			Controller:        controller,
			ControllerVariant: controllerVariant,
			CreatedDate:       formatCreatedDate(participant.GetCreatedAt()),
		})
	}
	return rows
}

// formatParticipantRole returns a display label and variant for a participant role.
func formatParticipantRole(role statev1.ParticipantRole, loc *message.Printer) (string, string) {
	switch role {
	case statev1.ParticipantRole_GM:
		return loc.Sprintf("label.gm"), "info"
	case statev1.ParticipantRole_PLAYER:
		return loc.Sprintf("label.player"), "success"
	default:
		return loc.Sprintf("label.unspecified"), "secondary"
	}
}

// formatParticipantController returns a display label and variant for a controller type.
func formatParticipantController(controller statev1.Controller, loc *message.Printer) (string, string) {
	switch controller {
	case statev1.Controller_CONTROLLER_HUMAN:
		return loc.Sprintf("label.human"), "success"
	case statev1.Controller_CONTROLLER_AI:
		return loc.Sprintf("label.ai"), "info"
	default:
		return loc.Sprintf("label.unspecified"), "secondary"
	}
}

// formatParticipantAccess returns a display label and variant for campaign access.
func formatParticipantAccess(access statev1.CampaignAccess, loc *message.Printer) (string, string) {
	switch access {
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER:
		return loc.Sprintf("label.member"), "secondary"
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER:
		return loc.Sprintf("label.manager"), "info"
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER:
		return loc.Sprintf("label.owner"), "warning"
	default:
		return loc.Sprintf("label.unspecified"), "secondary"
	}
}

// handleSessionDetail renders the session detail page.
func (h *Handler) handleSessionDetail(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
	loc, lang := h.localizer(w, r)
	sessionClient := h.sessionClient()
	if sessionClient == nil {
		http.Error(w, loc.Sprintf("error.session_service_unavailable"), http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
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

	if isHTMXRequest(r) {
		templ.Handler(templates.SessionDetailPage(detail, loc)).ServeHTTP(w, r)
		return
	}
	pageCtx := h.pageContext(lang, loc, r)
	templ.Handler(templates.SessionDetailFullPage(detail, pageCtx)).ServeHTTP(w, r)
}

// handleSessionEvents renders the session events via HTMX.
func (h *Handler) handleSessionEvents(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
	loc, _ := h.localizer(w, r)
	eventClient := h.eventClient()
	if eventClient == nil {
		templ.Handler(templates.EmptyState(loc.Sprintf("error.event_service_unavailable"))).ServeHTTP(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
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
		ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
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

	if isHTMXRequest(r) {
		templ.Handler(templates.EventLogPage(view, loc)).ServeHTTP(w, r)
		return
	}

	pageCtx := h.pageContext(lang, loc, r)
	templ.Handler(templates.EventLogFullPage(view, pageCtx)).ServeHTTP(w, r)
}

// handleEventLogTable renders the event log table via HTMX.
func (h *Handler) handleEventLogTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := h.localizer(w, r)
	eventClient := h.eventClient()
	if eventClient == nil {
		templ.Handler(templates.EmptyState(loc.Sprintf("error.event_service_unavailable"))).ServeHTTP(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
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

	if pushURL := eventFilterPushURL("/campaigns/"+campaignID+"/events", filters, pageToken); pushURL != "" {
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

	ctx, cancel := context.WithTimeout(r.Context(), grpcRequestTimeout)
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
