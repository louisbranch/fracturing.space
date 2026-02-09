package admin

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/a-h/templ"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/admin/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"golang.org/x/text/message"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// campaignsRequestTimeout caps the gRPC request time for campaigns.
	campaignsRequestTimeout = 2 * time.Second
	// campaignThemePromptLimit caps the number of characters shown in the table.
	campaignThemePromptLimit = 80
	// sessionListPageSize caps the number of sessions shown in the UI.
	sessionListPageSize = 10
	// eventListPageSize caps the number of events shown per page.
	eventListPageSize = 50
	// impersonationCookieName stores the active impersonation session ID.
	impersonationCookieName = "fs-impersonation-session"
	// impersonationSessionTTL controls how long impersonation sessions stay valid.
	impersonationSessionTTL = 24 * time.Hour
	// impersonationCleanupInterval controls how often expired sessions are purged.
	impersonationCleanupInterval = 30 * time.Minute
)

// GRPCClientProvider supplies gRPC clients for request handling.
type GRPCClientProvider interface {
	AuthClient() authv1.AuthServiceClient
	CampaignClient() statev1.CampaignServiceClient
	SessionClient() statev1.SessionServiceClient
	CharacterClient() statev1.CharacterServiceClient
	ParticipantClient() statev1.ParticipantServiceClient
	SnapshotClient() statev1.SnapshotServiceClient
	EventClient() statev1.EventServiceClient
	StatisticsClient() statev1.StatisticsServiceClient
}

// Handler routes admin dashboard requests.
type Handler struct {
	clientProvider GRPCClientProvider
	impersonation  *impersonationStore
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
	handler := &Handler{
		clientProvider: clientProvider,
		impersonation:  newImpersonationStore(),
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
	return templates.PageContext{
		Lang:          lang,
		Loc:           loc,
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
	mux.Handle("/users", http.HandlerFunc(h.handleUsersPage))
	mux.Handle("/users/table", http.HandlerFunc(h.handleUsersTable))
	mux.Handle("/users/create", http.HandlerFunc(h.handleCreateUser))
	mux.Handle("/users/impersonate", http.HandlerFunc(h.handleImpersonateUser))
	mux.Handle("/users/logout", http.HandlerFunc(h.handleLogout))
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

	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if userID != "" {
		client := h.authClient()
		if client == nil {
			view.Message = loc.Sprintf("error.user_service_unavailable")
		} else {
			ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
			defer cancel()

			response, err := client.GetUser(ctx, &authv1.GetUserRequest{UserId: userID})
			if err != nil || response.GetUser() == nil {
				log.Printf("get user: %v", err)
				view.Message = loc.Sprintf("error.user_not_found")
			} else {
				view.Detail = buildUserDetail(response.GetUser())
			}
		}
	}

	if isHTMXRequest(r) {
		templ.Handler(templates.UsersPage(view, loc)).ServeHTTP(w, r)
		return
	}

	templ.Handler(templates.UsersFullPage(view, pageCtx)).ServeHTTP(w, r)
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

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
	defer cancel()

	response, err := client.CreateUser(ctx, &authv1.CreateUserRequest{DisplayName: displayName})
	if err != nil || response.GetUser() == nil {
		log.Printf("create user: %v", err)
		view.Message = loc.Sprintf("error.user_create_failed")
		templ.Handler(templates.UsersFullPage(view, pageCtx)).ServeHTTP(w, r)
		return
	}

	created := response.GetUser()
	view.Detail = buildUserDetail(created)
	view.Message = loc.Sprintf("users.create.success")

	templ.Handler(templates.UsersFullPage(view, pageCtx)).ServeHTTP(w, r)
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
	view := templates.UsersPageView{Impersonation: pageCtx.Impersonation}
	if !requireSameOrigin(w, r, loc) {
		return
	}

	if err := r.ParseForm(); err != nil {
		view.Message = loc.Sprintf("error.user_impersonate_invalid")
		templ.Handler(templates.UsersFullPage(view, pageCtx)).ServeHTTP(w, r)
		return
	}

	userID := strings.TrimSpace(r.FormValue("user_id"))
	if userID == "" {
		view.Message = loc.Sprintf("error.user_id_required")
		templ.Handler(templates.UsersFullPage(view, pageCtx)).ServeHTTP(w, r)
		return
	}

	client := h.authClient()
	if client == nil {
		view.Message = loc.Sprintf("error.user_service_unavailable")
		templ.Handler(templates.UsersFullPage(view, pageCtx)).ServeHTTP(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
	defer cancel()

	response, err := client.GetUser(ctx, &authv1.GetUserRequest{UserId: userID})
	if err != nil || response.GetUser() == nil {
		log.Printf("get user for impersonation: %v", err)
		view.Message = loc.Sprintf("error.user_not_found")
		templ.Handler(templates.UsersFullPage(view, pageCtx)).ServeHTTP(w, r)
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
		templ.Handler(templates.UsersFullPage(view, pageCtx)).ServeHTTP(w, r)
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
	pageCtx.Impersonation = view.Impersonation
	label := strings.TrimSpace(user.GetDisplayName())
	if label == "" {
		label = user.GetId()
	}
	view.Message = loc.Sprintf("users.impersonate.success", label)

	templ.Handler(templates.UsersFullPage(view, pageCtx)).ServeHTTP(w, r)
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
	view := templates.UsersPageView{}
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
	templ.Handler(templates.UsersFullPage(view, pageCtx)).ServeHTTP(w, r)
}

// handleUsersTable renders the users table via HTMX.
func (h *Handler) handleUsersTable(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.localizer(w, r)
	client := h.authClient()
	if client == nil {
		h.renderUsersTable(w, r, nil, loc.Sprintf("error.user_service_unavailable"), loc)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
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

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
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

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
	defer cancel()
	if view.UserID != "" {
		md, _ := metadata.FromOutgoingContext(ctx)
		md = md.Copy()
		md.Set(grpcmeta.UserIDHeader, view.UserID)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	response, err := client.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
		Name:               view.Name,
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

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
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

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
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

// renderCampaignSessions renders the session list fragment.
func (h *Handler) renderCampaignSessions(w http.ResponseWriter, r *http.Request, rows []templates.CampaignSessionRow, message string, loc *message.Printer) {
	templ.Handler(templates.CampaignSessionsList(rows, message, loc)).ServeHTTP(w, r)
}

// renderUsersTable renders the users table component.
func (h *Handler) renderUsersTable(w http.ResponseWriter, r *http.Request, rows []templates.UserRow, message string, loc *message.Printer) {
	templ.Handler(templates.UsersTable(rows, message, loc)).ServeHTTP(w, r)
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
	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
	defer cancel()
	loc, _ := h.localizer(w, r)

	stats := templates.DashboardStats{
		TotalCampaigns:    "0",
		TotalSessions:     "0",
		TotalCharacters:   "0",
		TotalParticipants: "0",
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
	case "campaign.status_changed":
		return loc.Sprintf("event.campaign_status_changed")
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
	case "character.controller_assigned":
		return loc.Sprintf("event.character_controller_assigned")
	// Snapshot-related events
	case "snapshot.character_state_changed":
		return loc.Sprintf("event.snapshot_character_state_changed")
	case "snapshot.gm_fear_changed":
		return loc.Sprintf("event.snapshot_gm_fear_changed")
	// Session events
	case "session.started":
		return loc.Sprintf("event.session_started")
	case "session.ended":
		return loc.Sprintf("event.session_ended")
	// Action events
	case "action.roll_resolved":
		return loc.Sprintf("event.action_roll_resolved")
	case "action.outcome_applied":
		return loc.Sprintf("event.action_outcome_applied")
	case "action.outcome_rejected":
		return loc.Sprintf("event.action_outcome_rejected")
	case "action.note_added":
		return loc.Sprintf("event.action_note_added")
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

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
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

	rows := buildCharacterRows(characters, loc)
	h.renderCharactersTable(w, r, rows, "", loc)
}

// handleCharacterSheet renders the character sheet page.
func (h *Handler) handleCharacterSheet(w http.ResponseWriter, r *http.Request, campaignID string, characterID string) {
	loc, lang := h.localizer(w, r)
	characterClient := h.characterClient()
	if characterClient == nil {
		http.Error(w, loc.Sprintf("error.character_service_unavailable"), http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
	defer cancel()

	// Get character sheet
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

	// Get campaign name
	campaignName := getCampaignName(h, r, campaignID, loc)

	// Get recent events for this character
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

	sheet := buildCharacterSheet(campaignID, campaignName, character, recentEvents, loc)

	if isHTMXRequest(r) {
		templ.Handler(templates.CharacterSheetPage(sheet, loc)).ServeHTTP(w, r)
		return
	}
	pageCtx := h.pageContext(lang, loc, r)
	templ.Handler(templates.CharacterSheetFullPage(sheet, pageCtx)).ServeHTTP(w, r)
}

// renderCharactersTable renders the characters table component.
func (h *Handler) renderCharactersTable(w http.ResponseWriter, r *http.Request, rows []templates.CharacterRow, message string, loc *message.Printer) {
	templ.Handler(templates.CharactersTable(rows, message, loc)).ServeHTTP(w, r)
}

// buildCharacterRows formats character rows for the table.
func buildCharacterRows(characters []*statev1.Character, loc *message.Printer) []templates.CharacterRow {
	rows := make([]templates.CharacterRow, 0, len(characters))
	for _, character := range characters {
		if character == nil {
			continue
		}

		// Format controller
		controller := loc.Sprintf("label.unknown")
		// TODO: Get controller information (requires join with participant data)

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
func buildCharacterSheet(campaignID, campaignName string, character *statev1.Character, recentEvents []templates.EventRow, loc *message.Printer) templates.CharacterSheetView {
	return templates.CharacterSheetView{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		Character:    character,
		Controller:   loc.Sprintf("label.unknown"),
		CreatedAt:    formatTimestamp(character.GetCreatedAt()),
		UpdatedAt:    formatTimestamp(character.GetUpdatedAt()),
		RecentEvents: recentEvents,
	}
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

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
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

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
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

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
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

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
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
		ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
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

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
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

	ctx, cancel := context.WithTimeout(r.Context(), campaignsRequestTimeout)
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
