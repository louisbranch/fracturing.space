package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/i18n"
	webstorage "github.com/louisbranch/fracturing.space/internal/services/web/storage"
	websqlite "github.com/louisbranch/fracturing.space/internal/services/web/storage/sqlite"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"golang.org/x/text/message"
	"google.golang.org/grpc"
)

var subStaticFS = func() (fs.FS, error) {
	return fs.Sub(assetsFS, "static")
}

// Config defines the inputs for the web login server.
type Config struct {
	HTTPAddr             string
	AuthBaseURL          string
	AuthAddr             string
	GameAddr             string
	CacheDBPath          string
	AssetBaseURL         string
	AssetManifestVersion string
	AppName              string
	GRPCDialTimeout      time.Duration
	// OAuthClientID is the first-party OAuth client ID for web login.
	OAuthClientID string
	// CallbackURL is the public URL for the OAuth callback endpoint.
	CallbackURL string
	// AuthTokenURL is the internal auth token endpoint for code exchange.
	AuthTokenURL string
	// Domain is the parent domain used for cross-subdomain cookie scoping.
	Domain string
	// OAuthResourceSecret is used by web service to introspect access tokens.
	OAuthResourceSecret string
}

// Server hosts the web login HTTP server.
type Server struct {
	httpAddr                       string
	httpServer                     *http.Server
	authConn                       *grpc.ClientConn
	gameConn                       *grpc.ClientConn
	cacheStore                     *websqlite.Store
	cacheInvalidationDone          chan struct{}
	cacheInvalidationStop          context.CancelFunc
	campaignUpdateSubscriptionDone chan struct{}
	campaignUpdateSubscriptionStop context.CancelFunc
}

type handler struct {
	config              Config
	authClient          authv1.AuthServiceClient
	accountClient       authv1.AccountServiceClient
	sessions            *sessionStore
	pendingFlows        *pendingFlowStore
	cacheStore          webstorage.Store
	clientInitMu        sync.Mutex
	campaignNameCacheMu sync.RWMutex
	campaignNameCache   map[string]campaignNameCache
	campaignClient      statev1.CampaignServiceClient
	eventClient         statev1.EventServiceClient
	sessionClient       statev1.SessionServiceClient
	participantClient   statev1.ParticipantServiceClient
	characterClient     statev1.CharacterServiceClient
	inviteClient        statev1.InviteServiceClient
	campaignAccess      campaignAccessChecker
}

type handlerDependencies struct {
	campaignAccess    campaignAccessChecker
	cacheStore        webstorage.Store
	accountClient     authv1.AccountServiceClient
	campaignClient    statev1.CampaignServiceClient
	eventClient       statev1.EventServiceClient
	sessionClient     statev1.SessionServiceClient
	participantClient statev1.ParticipantServiceClient
	characterClient   statev1.CharacterServiceClient
	inviteClient      statev1.InviteServiceClient
}

// authGRPCClients holds the auth clients created during web startup.
type authGRPCClients struct {
	conn          *grpc.ClientConn
	authClient    authv1.AuthServiceClient
	accountClient authv1.AccountServiceClient
}

// gameGRPCClients holds the game clients used by the web service.
type gameGRPCClients struct {
	conn              *grpc.ClientConn
	participantClient statev1.ParticipantServiceClient
	campaignClient    statev1.CampaignServiceClient
	eventClient       statev1.EventServiceClient
	sessionClient     statev1.SessionServiceClient
	characterClient   statev1.CharacterServiceClient
	inviteClient      statev1.InviteServiceClient
}

// localizer resolves the request locale, optionally persists a cookie,
// and returns a message printer with the resolved language tag string.
func localizer(w http.ResponseWriter, r *http.Request) (*message.Printer, string) {
	tag, setCookie := webi18n.ResolveTag(r)
	if setCookie {
		webi18n.SetLanguageCookie(w, tag)
	}
	return webi18n.Printer(tag), tag.String()
}

func withStaticMime(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch path := strings.ToLower(r.URL.Path); {
		case strings.HasSuffix(path, ".css"):
			w.Header().Set("Content-Type", "text/css")
		case strings.HasSuffix(path, ".js"):
			w.Header().Set("Content-Type", "application/javascript")
		case strings.HasSuffix(path, ".svg"):
			w.Header().Set("Content-Type", "image/svg+xml")
		}
		next.ServeHTTP(w, r)
	})
}

func (h *handler) handleAppRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	appName := h.resolvedAppName()
	page := h.pageContext(w, r)
	page.AppName = appName

	if sess := sessionFromRequest(r, h.sessions); sess != nil {
		if err := h.writePage(w, r, webtemplates.DashboardPage(webtemplates.DashboardPageParams{
			AppName:       appName,
			Lang:          page.Lang,
			UserName:      page.UserName,
			UserAvatarURL: page.UserAvatarURL,
			CurrentPath:   page.CurrentPath,
			Loc:           page.Loc,
		}), composeHTMXTitleForPage(page, "dashboard.title")); err != nil {
			log.Printf("web: failed to render dashboard page: %v", err)
			localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
			return
		}
		return
	}

	params := webtemplates.LandingParams{}
	if strings.TrimSpace(h.config.OAuthClientID) != "" {
		params.SignInURL = "/auth/login"
	}
	if err := h.writePage(w, r, webtemplates.LandingPage(page, appName, params), composeHTMXTitleForPage(page, "title.landing")); err != nil {
		log.Printf("web: failed to render landing page: %v", err)
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
	}
}

// NewHandler creates the HTTP handler for the login UX.
//
// This function is the test-oriented entrypoint that assembles route handlers
// while keeping gRPC dependencies injectable via NewHandlerWithCampaignAccess.
func NewHandler(config Config, authClient authv1.AuthServiceClient) http.Handler {
	handler, err := NewHandlerWithCampaignAccess(config, authClient, handlerDependencies{})
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
		})
	}
	return handler
}

// NewHandlerWithCampaignAccess creates the HTTP handler with campaign access checks.
func NewHandlerWithCampaignAccess(config Config, authClient authv1.AuthServiceClient, deps handlerDependencies) (http.Handler, error) {
	rootMux := http.NewServeMux()
	staticFS, err := subStaticFS()
	if err != nil {
		return nil, fmt.Errorf("resolve static assets: %w", err)
	}
	rootMux.Handle(
		"/static/",
		withStaticMime(
			http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))),
		),
	)

	h := &handler{
		config:            config,
		authClient:        authClient,
		accountClient:     deps.accountClient,
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		cacheStore:        deps.cacheStore,
		campaignNameCache: make(map[string]campaignNameCache),
		campaignClient:    deps.campaignClient,
		eventClient:       deps.eventClient,
		sessionClient:     deps.sessionClient,
		participantClient: deps.participantClient,
		characterClient:   deps.characterClient,
		inviteClient:      deps.inviteClient,
		campaignAccess:    deps.campaignAccess,
	}

	gameMux := http.NewServeMux()
	h.registerGameRoutes(gameMux)

	publicMux := http.NewServeMux()
	h.registerPublicRoutes(publicMux, h.resolvedAppName())

	rootMux.Handle("/dashboard", gameMux)
	rootMux.Handle("/profile", gameMux)
	rootMux.Handle("/campaigns", gameMux)
	rootMux.Handle("/campaigns/", gameMux)
	rootMux.Handle("/invites", gameMux)
	rootMux.Handle("/invites/", gameMux)
	rootMux.Handle("/", publicMux)

	return rootMux, nil
}

func (h *handler) registerGameRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/dashboard", h.handleAppHome)
	mux.HandleFunc("/profile", h.handleAppProfile)
	mux.HandleFunc("/campaigns", h.handleAppCampaigns)
	mux.HandleFunc("/campaigns/create", h.handleAppCampaignCreate)
	mux.HandleFunc("/campaigns/", h.handleAppCampaignDetail)
	mux.HandleFunc("/invites", h.handleAppInvites)
	mux.HandleFunc("/invites/claim", h.handleAppInviteClaim)
}

func (h *handler) registerPublicRoutes(mux *http.ServeMux, appName string) {
	mux.HandleFunc("/", h.handleAppRoot)

	// Register OAuth client routes when configured.
	if strings.TrimSpace(h.config.OAuthClientID) != "" {
		mux.HandleFunc("/auth/login", h.handleAuthLogin)
		mux.HandleFunc("/auth/callback", h.handleAuthCallback)
		mux.HandleFunc("/auth/logout", h.handleAuthLogout)
	}

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
			return
		}

		printer, lang := localizer(w, r)

		pendingID := strings.TrimSpace(r.URL.Query().Get("pending_id"))
		if pendingID == "" {
			if strings.TrimSpace(h.config.OAuthClientID) != "" {
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}
			localizeHTTPError(w, r, http.StatusBadRequest, "error.http.pending_id_is_required")
			return
		}
		clientID := strings.TrimSpace(r.URL.Query().Get("client_id"))
		clientName := strings.TrimSpace(r.URL.Query().Get("client_name"))
		errorMessage := strings.TrimSpace(r.URL.Query().Get("error"))
		if clientName == "" {
			if clientID != "" {
				clientName = clientID
			} else {
				clientName = webtemplates.T(printer, "web.login.unknown_client")
			}
		}

		params := webtemplates.LoginParams{
			AppName:    appName,
			PendingID:  pendingID,
			ClientName: clientName,
			Error:      errorMessage,
			Lang:       lang,
			Loc:        printer,
		}
		loginPage := webtemplates.PageContext{
			Lang:    lang,
			Loc:     printer,
			AppName: appName,
		}
		if err := h.writePage(w, r, webtemplates.LoginPage(params), composeHTMXTitleForPage(loginPage, "title.login")); err != nil {
			log.Printf("web: failed to render login page: %v", err)
			localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
			return
		}
	})

	mux.HandleFunc("/magic", h.handleMagicLink)
	mux.HandleFunc("/passkeys/register/start", h.handlePasskeyRegisterStart)
	mux.HandleFunc("/passkeys/register/finish", h.handlePasskeyRegisterFinish)
	mux.HandleFunc("/passkeys/login/start", h.handlePasskeyLoginStart)
	mux.HandleFunc("/passkeys/login/finish", h.handlePasskeyLoginFinish)
	mux.HandleFunc("/up", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
}

// NewServer builds a configured web server.
//
// NewServer is the process entrypoint adapter:
// - wires auth/game gRPC dependencies for handlers
// - falls back to degraded UX mode when services are temporarily unavailable
// - returns a ready-to-run HTTP server wrapper.
func NewServer(config Config) (*Server, error) {
	return NewServerWithContext(context.Background(), config)
}

// NewServerWithContext builds a configured web server.
func NewServerWithContext(ctx context.Context, config Config) (*Server, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	httpAddr := strings.TrimSpace(config.HTTPAddr)
	if httpAddr == "" {
		return nil, errors.New("http address is required")
	}
	if strings.TrimSpace(config.AuthBaseURL) == "" {
		return nil, errors.New("auth base url is required")
	}
	if config.GRPCDialTimeout <= 0 {
		config.GRPCDialTimeout = timeouts.GRPCDial
	}

	cacheStore, err := openWebCacheStore(config.CacheDBPath)
	if err != nil {
		return nil, err
	}

	var authConn *grpc.ClientConn
	var authClient authv1.AuthServiceClient
	var accountClient authv1.AccountServiceClient
	if strings.TrimSpace(config.AuthAddr) != "" {
		clients, err := dialAuthGRPC(ctx, config)
		if err != nil {
			if cacheStore != nil {
				_ = cacheStore.Close()
			}
			return nil, fmt.Errorf("dial auth grpc: %w", err)
		}
		authConn = clients.conn
		authClient = clients.authClient
		accountClient = clients.accountClient
	}

	var gameConn *grpc.ClientConn
	var participantClient statev1.ParticipantServiceClient
	var campaignClient statev1.CampaignServiceClient
	var eventClient statev1.EventServiceClient
	var sessionClient statev1.SessionServiceClient
	var characterClient statev1.CharacterServiceClient
	var inviteClient statev1.InviteServiceClient
	if strings.TrimSpace(config.GameAddr) != "" {
		clients, err := dialGameGRPC(ctx, config)
		if err != nil {
			log.Printf("game gRPC dial failed, campaign access checks disabled: %v", err)
		} else {
			gameConn = clients.conn
			participantClient = clients.participantClient
			campaignClient = clients.campaignClient
			eventClient = clients.eventClient
			sessionClient = clients.sessionClient
			characterClient = clients.characterClient
			inviteClient = clients.inviteClient
		}
	}
	campaignAccess := newCampaignAccessChecker(config, participantClient)
	handler, err := NewHandlerWithCampaignAccess(config, authClient, handlerDependencies{
		campaignAccess:    campaignAccess,
		cacheStore:        cacheStore,
		accountClient:     accountClient,
		campaignClient:    campaignClient,
		eventClient:       eventClient,
		sessionClient:     sessionClient,
		participantClient: participantClient,
		characterClient:   characterClient,
		inviteClient:      inviteClient,
	})
	if err != nil {
		if cacheStore != nil {
			_ = cacheStore.Close()
		}
		return nil, fmt.Errorf("build handler: %w", err)
	}
	httpServer := &http.Server{
		Addr:              httpAddr,
		Handler:           handler,
		ReadHeaderTimeout: timeouts.ReadHeader,
	}

	invalidationStop, invalidationDone := startCacheInvalidationWorker(cacheStore, eventClient)
	campaignUpdateStop, campaignUpdateDone := startCampaignProjectionSubscriptionWorker(cacheStore, eventClient)

	return &Server{
		httpAddr:                       httpAddr,
		httpServer:                     httpServer,
		authConn:                       authConn,
		gameConn:                       gameConn,
		cacheStore:                     cacheStore,
		cacheInvalidationDone:          invalidationDone,
		cacheInvalidationStop:          invalidationStop,
		campaignUpdateSubscriptionDone: campaignUpdateDone,
		campaignUpdateSubscriptionStop: campaignUpdateStop,
	}, nil
}

// ListenAndServe runs the HTTP server until the context ends.
//
// On cancellation, it performs a bounded shutdown so in-flight requests
// are drained before hard close.
func (s *Server) ListenAndServe(ctx context.Context) error {
	if s == nil {
		return errors.New("web server is nil")
	}
	if ctx == nil {
		return errors.New("context is required")
	}

	serveErr := make(chan error, 1)
	log.Printf("web login listening on %s", s.httpAddr)
	go func() {
		serveErr <- s.httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), timeouts.Shutdown)
		err := s.httpServer.Shutdown(shutdownCtx)
		cancel()
		if err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}
		return nil
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve http: %w", err)
	}
}

// Close releases any gRPC resources held by the server.
func (s *Server) Close() {
	if s == nil {
		return
	}
	if s.cacheInvalidationStop != nil {
		s.cacheInvalidationStop()
	}
	if s.campaignUpdateSubscriptionStop != nil {
		s.campaignUpdateSubscriptionStop()
	}
	if s.cacheInvalidationDone != nil {
		<-s.cacheInvalidationDone
	}
	if s.campaignUpdateSubscriptionDone != nil {
		<-s.campaignUpdateSubscriptionDone
	}
	if s.authConn != nil {
		if err := s.authConn.Close(); err != nil {
			log.Printf("close auth gRPC connection: %v", err)
		}
	}
	if s.gameConn != nil {
		if err := s.gameConn.Close(); err != nil {
			log.Printf("close game gRPC connection: %v", err)
		}
	}
	if s.cacheStore != nil {
		if err := s.cacheStore.Close(); err != nil {
			log.Printf("close web cache store: %v", err)
		}
	}
}

func openWebCacheStore(path string) (*websqlite.Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create web cache dir: %w", err)
		}
	}
	store, err := websqlite.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open web cache sqlite store: %w", err)
	}
	return store, nil
}

// buildAuthConsentURL resolves the post-magic-link consent callback.
// It keeps OAuth return handling deterministic regardless of deployment prefixing.
func buildAuthConsentURL(base string, pendingID string) string {
	base = strings.TrimSpace(base)
	encoded := url.QueryEscape(pendingID)
	if base == "" {
		return "/authorize/consent?pending_id=" + encoded
	}
	return strings.TrimRight(base, "/") + "/authorize/consent?pending_id=" + encoded
}

// dialAuthGRPC returns a client for auth-backed login/registration operations.
// Auth transport is optional in degraded startup modes so the web package can
// still stand up with limited capability.
func dialAuthGRPC(ctx context.Context, config Config) (authGRPCClients, error) {
	authAddr := strings.TrimSpace(config.AuthAddr)
	if authAddr == "" {
		return authGRPCClients{}, nil
	}
	if ctx == nil {
		return authGRPCClients{}, errors.New("context is required")
	}
	if config.GRPCDialTimeout <= 0 {
		config.GRPCDialTimeout = timeouts.GRPCDial
	}
	logf := func(format string, args ...any) {
		log.Printf("auth %s", fmt.Sprintf(format, args...))
	}
	conn, err := platformgrpc.DialWithHealth(
		ctx,
		nil,
		authAddr,
		config.GRPCDialTimeout,
		logf,
		platformgrpc.DefaultClientDialOptions()...,
	)
	if err != nil {
		var dialErr *platformgrpc.DialError
		if errors.As(err, &dialErr) {
			if dialErr.Stage == platformgrpc.DialStageHealth {
				return authGRPCClients{}, fmt.Errorf("auth gRPC health check failed for %s: %w", authAddr, dialErr.Err)
			}
			return authGRPCClients{}, fmt.Errorf("dial auth gRPC %s: %w", authAddr, dialErr.Err)
		}
		return authGRPCClients{}, fmt.Errorf("dial auth gRPC %s: %w", authAddr, err)
	}
	return authGRPCClients{
		conn:          conn,
		authClient:    authv1.NewAuthServiceClient(conn),
		accountClient: authv1.NewAccountServiceClient(conn),
	}, nil
}

// dialGameGRPC returns clients for campaign/character/session/invite operations.
// This dependency is optional by design so campaign routes can degrade gracefully
// during partial service outages.
func dialGameGRPC(ctx context.Context, config Config) (gameGRPCClients, error) {
	gameAddr := strings.TrimSpace(config.GameAddr)
	if gameAddr == "" {
		return gameGRPCClients{}, nil
	}
	if ctx == nil {
		return gameGRPCClients{}, errors.New("context is required")
	}
	if config.GRPCDialTimeout <= 0 {
		config.GRPCDialTimeout = timeouts.GRPCDial
	}
	logf := func(format string, args ...any) {
		log.Printf("game %s", fmt.Sprintf(format, args...))
	}
	conn, err := platformgrpc.DialWithHealth(
		ctx,
		nil,
		gameAddr,
		config.GRPCDialTimeout,
		logf,
		platformgrpc.DefaultClientDialOptions()...,
	)
	if err != nil {
		var dialErr *platformgrpc.DialError
		if errors.As(err, &dialErr) {
			if dialErr.Stage == platformgrpc.DialStageHealth {
				return gameGRPCClients{}, fmt.Errorf("game gRPC health check failed for %s: %w", gameAddr, dialErr.Err)
			}
			return gameGRPCClients{}, fmt.Errorf("dial game gRPC %s: %w", gameAddr, dialErr.Err)
		}
		return gameGRPCClients{}, fmt.Errorf("dial game gRPC %s: %w", gameAddr, err)
	}
	return gameGRPCClients{
		conn:              conn,
		participantClient: statev1.NewParticipantServiceClient(conn),
		campaignClient:    statev1.NewCampaignServiceClient(conn),
		eventClient:       statev1.NewEventServiceClient(conn),
		sessionClient:     statev1.NewSessionServiceClient(conn),
		characterClient:   statev1.NewCharacterServiceClient(conn),
		inviteClient:      statev1.NewInviteServiceClient(conn),
	}, nil
}

// handlePasskeyLoginStart begins a passkey authentication round trip and returns
// the credential challenge expected by browser/WebAuth clients.
func (h *handler) handlePasskeyLoginStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if h == nil || h.authClient == nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.auth_client_not_configured")
		return
	}

	var payload struct {
		PendingID string `json:"pending_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.invalid_json_body")
		return
	}
	if strings.TrimSpace(payload.PendingID) == "" {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.pending_id_is_required")
		return
	}

	resp, err := h.authClient.BeginPasskeyLogin(r.Context(), &authv1.BeginPasskeyLoginRequest{})
	if err != nil {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.failed_to_start_passkey_login")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"session_id": resp.GetSessionId(),
		"public_key": json.RawMessage(resp.GetCredentialRequestOptionsJson()),
	})
}

// handlePasskeyLoginFinish finalizes passkey authentication and hands control back
// to the consent flow via the shared pending transaction state.
func (h *handler) handlePasskeyLoginFinish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if h == nil || h.authClient == nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.auth_client_not_configured")
		return
	}

	var payload struct {
		PendingID  string          `json:"pending_id"`
		SessionID  string          `json:"session_id"`
		Credential json.RawMessage `json:"credential"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.invalid_json_body")
		return
	}
	if strings.TrimSpace(payload.PendingID) == "" {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.pending_id_is_required")
		return
	}
	if strings.TrimSpace(payload.SessionID) == "" {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.session_id_is_required")
		return
	}
	if len(payload.Credential) == 0 {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.credential_is_required")
		return
	}

	_, err := h.authClient.FinishPasskeyLogin(r.Context(), &authv1.FinishPasskeyLoginRequest{
		SessionId:              payload.SessionID,
		CredentialResponseJson: payload.Credential,
		PendingId:              payload.PendingID,
	})
	if err != nil {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.failed_to_finish_passkey_login")
		return
	}

	redirectURL := buildAuthConsentURL(h.config.AuthBaseURL, payload.PendingID)
	writeJSON(w, http.StatusOK, map[string]any{"redirect_url": redirectURL})
}

// handlePasskeyRegisterStart creates a new passkey credential request so users can
// onboard a new WebAuth identity without leaving the current auth flow.
func (h *handler) handlePasskeyRegisterStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if h == nil || h.authClient == nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.auth_client_not_configured")
		return
	}

	var payload struct {
		Email     string `json:"email"`
		PendingID string `json:"pending_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.invalid_json_body")
		return
	}
	if strings.TrimSpace(payload.Email) == "" {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.email_is_required")
		return
	}

	createResp, err := h.authClient.CreateUser(r.Context(), &authv1.CreateUserRequest{Email: payload.Email})
	if err != nil || createResp.GetUser() == nil {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.failed_to_create_user")
		return
	}

	beginResp, err := h.authClient.BeginPasskeyRegistration(r.Context(), &authv1.BeginPasskeyRegistrationRequest{UserId: createResp.GetUser().GetId()})
	if err != nil {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.failed_to_start_passkey_registration")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"session_id": beginResp.GetSessionId(),
		"public_key": json.RawMessage(beginResp.GetCredentialCreationOptionsJson()),
		"user_id":    createResp.GetUser().GetId(),
		"pending_id": strings.TrimSpace(payload.PendingID),
	})
}

// handlePasskeyRegisterFinish completes the registration ceremony and returns the
// newly created participant binding identifiers for client continuation.
func (h *handler) handlePasskeyRegisterFinish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if h == nil || h.authClient == nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.auth_client_not_configured")
		return
	}

	var payload struct {
		PendingID  string          `json:"pending_id"`
		SessionID  string          `json:"session_id"`
		UserID     string          `json:"user_id"`
		Credential json.RawMessage `json:"credential"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.invalid_json_body")
		return
	}
	if strings.TrimSpace(payload.SessionID) == "" {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.session_id_is_required")
		return
	}
	if strings.TrimSpace(payload.UserID) == "" {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.user_id_is_required")
		return
	}
	if len(payload.Credential) == 0 {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.credential_is_required")
		return
	}

	_, err := h.authClient.FinishPasskeyRegistration(r.Context(), &authv1.FinishPasskeyRegistrationRequest{
		SessionId:              payload.SessionID,
		CredentialResponseJson: payload.Credential,
	})
	if err != nil {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.failed_to_finish_passkey_registration")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":    payload.UserID,
		"pending_id": strings.TrimSpace(payload.PendingID),
	})
}

// handleMagicLink validates one-time login tokens and moves valid sessions into the
// normal consent redirect path.
func (h *handler) handleMagicLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	printer, lang := localizer(w, r)
	if h == nil || h.authClient == nil {
		h.renderMagicPage(w, r, http.StatusInternalServerError, webtemplates.MagicParams{
			AppName: h.resolvedAppName(),
			Title:   printer.Sprintf("magic.unavailable.title"),
			Message: printer.Sprintf("magic.unavailable.message"),
			Detail:  printer.Sprintf("magic.unavailable.detail"),
			Loc:     printer,
			Success: false,
			Lang:    lang,
		})
		return
	}

	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		h.renderMagicPage(w, r, http.StatusBadRequest, webtemplates.MagicParams{
			AppName: h.resolvedAppName(),
			Title:   printer.Sprintf("magic.missing.title"),
			Message: printer.Sprintf("magic.missing.message"),
			Detail:  printer.Sprintf("magic.missing.detail"),
			Loc:     printer,
			Success: false,
			Lang:    lang,
		})
		return
	}

	resp, err := h.authClient.ConsumeMagicLink(r.Context(), &authv1.ConsumeMagicLinkRequest{Token: token})
	if err != nil {
		h.renderMagicPage(w, r, http.StatusBadRequest, webtemplates.MagicParams{
			AppName: h.resolvedAppName(),
			Title:   printer.Sprintf("magic.invalid.title"),
			Message: printer.Sprintf("magic.invalid.message"),
			Detail:  printer.Sprintf("magic.invalid.detail"),
			Loc:     printer,
			Success: false,
			Lang:    lang,
		})
		return
	}
	if pendingID := strings.TrimSpace(resp.GetPendingId()); pendingID != "" {
		redirectURL := buildAuthConsentURL(h.config.AuthBaseURL, pendingID)
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}

	h.renderMagicPage(w, r, http.StatusOK, webtemplates.MagicParams{
		AppName:   h.resolvedAppName(),
		Title:     printer.Sprintf("magic.verified.title"),
		Message:   printer.Sprintf("magic.verified.message"),
		Detail:    printer.Sprintf("magic.verified.detail"),
		Loc:       printer,
		Success:   true,
		LinkURL:   "/",
		LinkLabel: printer.Sprintf("magic.verified.link"),
		Lang:      lang,
	})
}

// renderMagicPage writes the status code and renders the magic-link templ page.
func (h *handler) renderMagicPage(w http.ResponseWriter, r *http.Request, status int, params webtemplates.MagicParams) {
	writeGameContentType(w)
	w.WriteHeader(status)
	if err := h.writePage(
		w,
		r,
		webtemplates.MagicPage(params),
		composeHTMXTitleForPage(webtemplates.PageContext{
			Lang:    params.Lang,
			Loc:     params.Loc,
			AppName: h.resolvedAppName(),
		}, params.Title),
	); err != nil {
		log.Printf("web: failed to render magic page: %v", err)
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
	}
}

// handleAuthLogin initiates the OAuth PKCE flow by redirecting to the auth server
// with state and challenge that ties browser and token exchange together.
func (h *handler) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	verifier, err := generateCodeVerifier()
	if err != nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.failed_to_generate_pkce_verifier")
		return
	}
	challenge := computeS256Challenge(verifier)
	state := h.pendingFlows.create(verifier)

	authorizeURL := strings.TrimRight(strings.TrimSpace(h.config.AuthBaseURL), "/") + "/authorize"
	redirectURL, err := url.Parse(authorizeURL)
	if err != nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.invalid_auth_base_url")
		return
	}
	q := redirectURL.Query()
	q.Set("response_type", "code")
	q.Set("client_id", h.config.OAuthClientID)
	q.Set("redirect_uri", h.config.CallbackURL)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)
	redirectURL.RawQuery = q.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

// handleAuthCallback exchanges the authorization code for a token and creates a
// web session that subsequent web handlers can reuse for campaign membership checks.
func (h *handler) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	state := strings.TrimSpace(r.URL.Query().Get("state"))

	if code == "" || state == "" {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.missing_code_or_state")
		return
	}

	flow := h.pendingFlows.consume(state)
	if flow == nil {
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.invalid_or_expired_state")
		return
	}

	tokenURL := strings.TrimSpace(h.config.AuthTokenURL)
	if tokenURL == "" {
		tokenURL = strings.TrimRight(strings.TrimSpace(h.config.AuthBaseURL), "/") + "/token"
	}

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {h.config.CallbackURL},
		"code_verifier": {flow.codeVerifier},
		"client_id":     {h.config.OAuthClientID},
	}

	resp, err := http.PostForm(tokenURL, form)
	if err != nil {
		localizeHTTPError(w, r, http.StatusBadGateway, "error.http.token_exchange_failed")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		localizeHTTPError(w, r, http.StatusBadGateway, "error.http.token_exchange_returned", resp.Status)
		return
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		localizeHTTPError(w, r, http.StatusBadGateway, "error.http.failed_to_decode_token_response")
		return
	}

	if tokenResp.AccessToken == "" {
		localizeHTTPError(w, r, http.StatusBadGateway, "error.http.empty_access_token")
		return
	}

	expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	sessionID := h.sessions.create(tokenResp.AccessToken, "", expiry)
	setSessionCookie(w, sessionID)
	setTokenCookie(w, tokenResp.AccessToken, h.config.Domain, int(tokenResp.ExpiresIn))
	http.Redirect(w, r, "/", http.StatusFound)
}

// handleAuthLogout clears both local and cross-subdomain session/token artifacts to
// avoid mixed-session conditions across web and auth-aware siblings.
func (h *handler) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		h.sessions.delete(cookie.Value)
	}
	clearSessionCookie(w)
	clearTokenCookie(w, h.config.Domain)
	http.Redirect(w, r, "/", http.StatusFound)
}

// writeJSON writes JSON responses with a consistent content type for auth flows.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	_ = encoder.Encode(payload)
}
