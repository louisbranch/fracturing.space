package web

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
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
	ChatHTTPAddr         string
	AuthBaseURL          string
	AuthAddr             string
	ConnectionsAddr      string
	GameAddr             string
	NotificationsAddr    string
	AIAddr               string
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
	connectionsConn                *grpc.ClientConn
	gameConn                       *grpc.ClientConn
	notificationsConn              *grpc.ClientConn
	aiConn                         *grpc.ClientConn
	cacheStore                     *websqlite.Store
	cacheInvalidationDone          chan struct{}
	cacheInvalidationStop          context.CancelFunc
	campaignUpdateSubscriptionDone chan struct{}
	campaignUpdateSubscriptionStop context.CancelFunc
}

type handler struct {
	config              Config
	authClient          authv1.AuthServiceClient
	connectionsClient   connectionsv1.ConnectionsServiceClient
	accountClient       authv1.AccountServiceClient
	credentialClient    aiv1.CredentialServiceClient
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
	notificationClient  notificationsv1.NotificationServiceClient
	campaignAccess      campaignAccessChecker
}

type handlerDependencies struct {
	campaignAccess     campaignAccessChecker
	cacheStore         webstorage.Store
	accountClient      authv1.AccountServiceClient
	connectionsClient  connectionsv1.ConnectionsServiceClient
	credentialClient   aiv1.CredentialServiceClient
	campaignClient     statev1.CampaignServiceClient
	eventClient        statev1.EventServiceClient
	sessionClient      statev1.SessionServiceClient
	participantClient  statev1.ParticipantServiceClient
	characterClient    statev1.CharacterServiceClient
	inviteClient       statev1.InviteServiceClient
	notificationClient notificationsv1.NotificationServiceClient
}

// authGRPCClients holds the auth clients created during web startup.
type authGRPCClients struct {
	conn          *grpc.ClientConn
	authClient    authv1.AuthServiceClient
	accountClient authv1.AccountServiceClient
}

// connectionsGRPCClients holds the connections clients used by the web service.
type connectionsGRPCClients struct {
	conn              *grpc.ClientConn
	connectionsClient connectionsv1.ConnectionsServiceClient
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

// aiGRPCClients holds AI clients used by the web service.
type aiGRPCClients struct {
	conn             *grpc.ClientConn
	credentialClient aiv1.CredentialServiceClient
}

// notificationsGRPCClients holds notifications clients used by the web service.
type notificationsGRPCClients struct {
	conn               *grpc.ClientConn
	notificationClient notificationsv1.NotificationServiceClient
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
			AppName:                appName,
			Lang:                   page.Lang,
			UserName:               page.UserName,
			UserAvatarURL:          page.UserAvatarURL,
			HasUnreadNotifications: page.HasUnreadNotifications,
			CurrentPath:            page.CurrentPath,
			Loc:                    page.Loc,
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
	var sessionPersistence sessionPersistence
	if webSessionStore, ok := deps.cacheStore.(*websqlite.Store); ok && webSessionStore != nil {
		sessionPersistence = webSessionStore
	}
	rootMux.Handle(
		"/static/",
		withStaticMime(
			http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))),
		),
	)

	h := &handler{
		config:             config,
		authClient:         authClient,
		connectionsClient:  deps.connectionsClient,
		accountClient:      deps.accountClient,
		credentialClient:   deps.credentialClient,
		sessions:           newSessionStore(sessionPersistence),
		pendingFlows:       newPendingFlowStore(),
		cacheStore:         deps.cacheStore,
		campaignNameCache:  make(map[string]campaignNameCache),
		campaignClient:     deps.campaignClient,
		eventClient:        deps.eventClient,
		sessionClient:      deps.sessionClient,
		participantClient:  deps.participantClient,
		characterClient:    deps.characterClient,
		inviteClient:       deps.inviteClient,
		notificationClient: deps.notificationClient,
		campaignAccess:     deps.campaignAccess,
	}

	gameMux := http.NewServeMux()
	h.registerGameRoutes(gameMux)

	publicMux := http.NewServeMux()
	h.registerPublicRoutes(publicMux, h.resolvedAppName())

	rootMux.Handle("/dashboard", gameMux)
	rootMux.Handle("/profile", gameMux)
	rootMux.Handle("/settings", gameMux)
	rootMux.Handle("/settings/", gameMux)
	rootMux.Handle("/campaigns", gameMux)
	rootMux.Handle("/campaigns/", gameMux)
	rootMux.Handle("/invites", gameMux)
	rootMux.Handle("/invites/", gameMux)
	rootMux.Handle("/notifications", gameMux)
	rootMux.Handle("/notifications/", gameMux)
	rootMux.Handle("/", publicMux)

	return rootMux, nil
}

func (h *handler) registerGameRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/dashboard", h.handleAppHome)
	mux.HandleFunc("/profile", h.handleAppProfile)
	mux.HandleFunc("/settings", h.handleAppSettings)
	mux.HandleFunc("/settings/", h.handleAppSettingsRoutes)
	mux.HandleFunc("/campaigns", h.handleAppCampaigns)
	mux.HandleFunc("/campaigns/create", h.handleAppCampaignCreate)
	mux.HandleFunc("/campaigns/", h.handleAppCampaignDetail)
	mux.HandleFunc("/invites", h.handleAppInvites)
	mux.HandleFunc("/invites/claim", h.handleAppInviteClaim)
	mux.HandleFunc("/notifications", h.handleAppNotifications)
	mux.HandleFunc("/notifications/", h.handleAppNotificationsRoutes)
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
			AppName:      appName,
			PendingID:    pendingID,
			ClientName:   clientName,
			Error:        errorMessage,
			Lang:         lang,
			Loc:          printer,
			CurrentPath:  r.URL.Path,
			CurrentQuery: r.URL.RawQuery,
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

	var authClients authGRPCClients
	if strings.TrimSpace(config.AuthAddr) != "" {
		authClients, err = dialAuthGRPC(ctx, config)
		if err != nil {
			if cacheStore != nil {
				_ = cacheStore.Close()
			}
			return nil, fmt.Errorf("dial auth grpc: %w", err)
		}
	}
	var connectionsClients connectionsGRPCClients
	if strings.TrimSpace(config.ConnectionsAddr) != "" {
		connectionsClients, err = dialConnectionsGRPC(ctx, config)
		if err != nil {
			log.Printf("connections gRPC dial failed, invite contact options disabled: %v", err)
		}
	}

	var gameClients gameGRPCClients
	if strings.TrimSpace(config.GameAddr) != "" {
		gameClients, err = dialGameGRPC(ctx, config)
		if err != nil {
			log.Printf("game gRPC dial failed, campaign access checks disabled: %v", err)
		}
	}
	var notificationsClients notificationsGRPCClients
	if strings.TrimSpace(config.NotificationsAddr) != "" {
		notificationsClients, err = dialNotificationsGRPC(ctx, config)
		if err != nil {
			log.Printf("notifications gRPC dial failed, notifications routes disabled: %v", err)
		}
	}
	var aiClients aiGRPCClients
	if strings.TrimSpace(config.AIAddr) != "" {
		aiClients, err = dialAIGRPC(ctx, config)
		if err != nil {
			log.Printf("ai gRPC dial failed, settings ai keys disabled: %v", err)
		}
	}
	campaignAccess := newCampaignAccessChecker(config, gameClients.participantClient)
	handler, err := NewHandlerWithCampaignAccess(config, authClients.authClient, handlerDependencies{
		campaignAccess:     campaignAccess,
		cacheStore:         cacheStore,
		accountClient:      authClients.accountClient,
		connectionsClient:  connectionsClients.connectionsClient,
		credentialClient:   aiClients.credentialClient,
		campaignClient:     gameClients.campaignClient,
		eventClient:        gameClients.eventClient,
		sessionClient:      gameClients.sessionClient,
		participantClient:  gameClients.participantClient,
		characterClient:    gameClients.characterClient,
		inviteClient:       gameClients.inviteClient,
		notificationClient: notificationsClients.notificationClient,
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

	invalidationStop, invalidationDone := startCacheInvalidationWorker(cacheStore, gameClients.eventClient)
	campaignUpdateStop, campaignUpdateDone := startCampaignProjectionSubscriptionWorker(cacheStore, gameClients.eventClient)

	return &Server{
		httpAddr:                       httpAddr,
		httpServer:                     httpServer,
		authConn:                       authClients.conn,
		connectionsConn:                connectionsClients.conn,
		gameConn:                       gameClients.conn,
		notificationsConn:              notificationsClients.conn,
		aiConn:                         aiClients.conn,
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
	if s.connectionsConn != nil {
		if err := s.connectionsConn.Close(); err != nil {
			log.Printf("close connections gRPC connection: %v", err)
		}
	}
	if s.gameConn != nil {
		if err := s.gameConn.Close(); err != nil {
			log.Printf("close game gRPC connection: %v", err)
		}
	}
	if s.notificationsConn != nil {
		if err := s.notificationsConn.Close(); err != nil {
			log.Printf("close notifications gRPC connection: %v", err)
		}
	}
	if s.aiConn != nil {
		if err := s.aiConn.Close(); err != nil {
			log.Printf("close ai gRPC connection: %v", err)
		}
	}
	if s.cacheStore != nil {
		if err := s.cacheStore.Close(); err != nil {
			log.Printf("close web cache store: %v", err)
		}
	}
}
