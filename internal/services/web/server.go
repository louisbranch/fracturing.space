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
	"strings"
	"time"

	"github.com/a-h/templ"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/branding"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/i18n"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"golang.org/x/text/message"
	"google.golang.org/grpc"
)

var subStaticFS = func() (fs.FS, error) {
	return fs.Sub(assetsFS, "static")
}

// Config defines the inputs for the web login server.
type Config struct {
	HTTPAddr        string
	AuthBaseURL     string
	AuthAddr        string
	GameAddr        string
	AppName         string
	GRPCDialTimeout time.Duration
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
	httpAddr   string
	httpServer *http.Server
	authConn   *grpc.ClientConn
	gameConn   *grpc.ClientConn
}

type handler struct {
	config            Config
	authClient        authv1.AuthServiceClient
	sessions          *sessionStore
	pendingFlows      *pendingFlowStore
	campaignClient    statev1.CampaignServiceClient
	sessionClient     statev1.SessionServiceClient
	participantClient statev1.ParticipantServiceClient
	characterClient   statev1.CharacterServiceClient
	inviteClient      statev1.InviteServiceClient
	campaignAccess    campaignAccessChecker
}

type handlerDependencies struct {
	campaignAccess    campaignAccessChecker
	campaignClient    statev1.CampaignServiceClient
	sessionClient     statev1.SessionServiceClient
	participantClient statev1.ParticipantServiceClient
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

// NewHandler creates the HTTP handler for the login UX.
//
// This function is the test-oriented entrypoint that assembles route handlers
// while keeping gRPC dependencies injectable via NewHandlerWithCampaignAccess.
func NewHandler(config Config, authClient authv1.AuthServiceClient) http.Handler {
	handler, err := NewHandlerWithCampaignAccess(config, authClient, handlerDependencies{})
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "web handler unavailable", http.StatusInternalServerError)
		})
	}
	return handler
}

// NewHandlerWithCampaignAccess creates the HTTP handler with campaign access checks.
func NewHandlerWithCampaignAccess(config Config, authClient authv1.AuthServiceClient, deps handlerDependencies) (http.Handler, error) {
	mux := http.NewServeMux()
	staticFS, err := subStaticFS()
	if err != nil {
		return nil, fmt.Errorf("resolve static assets: %w", err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	appName := strings.TrimSpace(config.AppName)
	if appName == "" {
		appName = branding.AppName
	}
	h := &handler{
		config:            config,
		authClient:        authClient,
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		campaignClient:    deps.campaignClient,
		sessionClient:     deps.sessionClient,
		participantClient: deps.participantClient,
		characterClient:   deps.characterClient,
		inviteClient:      deps.inviteClient,
		campaignAccess:    deps.campaignAccess,
	}

	mux.HandleFunc("/app", h.handleAppHome)
	mux.HandleFunc("/app/campaigns", h.handleAppCampaigns)
	mux.HandleFunc("/app/campaigns/create", h.handleAppCampaignCreate)
	mux.HandleFunc("/app/campaigns/", h.handleAppCampaignDetail)
	mux.HandleFunc("/app/invites", h.handleAppInvites)
	mux.HandleFunc("/app/invites/claim", h.handleAppInviteClaim)

	// Register OAuth client routes when configured.
	if strings.TrimSpace(config.OAuthClientID) != "" {
		mux.HandleFunc("/auth/login", h.handleAuthLogin)
		mux.HandleFunc("/auth/callback", h.handleAuthCallback)
		mux.HandleFunc("/auth/logout", h.handleAuthLogout)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		printer, lang := localizer(w, r)
		page := webtemplates.PageContext{
			Lang:         lang,
			Loc:          printer,
			CurrentPath:  r.URL.Path,
			CurrentQuery: r.URL.RawQuery,
		}
		params := webtemplates.LandingParams{}
		if strings.TrimSpace(config.OAuthClientID) != "" {
			params.SignInURL = "/auth/login"
		}
		if sess := sessionFromRequest(r, h.sessions); sess != nil {
			name := sess.displayName
			if name == "" {
				name = "User"
			}
			params.UserName = name
		}
		templ.Handler(webtemplates.LandingPage(page, appName, params)).ServeHTTP(w, r)
	})

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		pendingID := strings.TrimSpace(r.URL.Query().Get("pending_id"))
		if pendingID == "" {
			if strings.TrimSpace(config.OAuthClientID) != "" {
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}
			http.Error(w, "pending_id is required", http.StatusBadRequest)
			return
		}
		clientID := strings.TrimSpace(r.URL.Query().Get("client_id"))
		clientName := strings.TrimSpace(r.URL.Query().Get("client_name"))
		errorMessage := strings.TrimSpace(r.URL.Query().Get("error"))
		if clientName == "" {
			if clientID != "" {
				clientName = clientID
			} else {
				clientName = "Unknown Client"
			}
		}

		printer, lang := localizer(w, r)
		params := webtemplates.LoginParams{
			AppName:    appName,
			PendingID:  pendingID,
			ClientID:   clientID,
			ClientName: clientName,
			Error:      errorMessage,
			Lang:       lang,
			Loc:        printer,
		}
		templ.Handler(webtemplates.LoginPage(params)).ServeHTTP(w, r)
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

	return mux, nil
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

	var authConn *grpc.ClientConn
	var authClient authv1.AuthServiceClient
	if strings.TrimSpace(config.AuthAddr) != "" {
		conn, client, err := dialAuthGRPC(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("dial auth grpc: %w", err)
		}
		authConn = conn
		authClient = client
	}

	var gameConn *grpc.ClientConn
	var participantClient statev1.ParticipantServiceClient
	var campaignClient statev1.CampaignServiceClient
	var sessionClient statev1.SessionServiceClient
	var characterClient statev1.CharacterServiceClient
	var inviteClient statev1.InviteServiceClient
	if strings.TrimSpace(config.GameAddr) != "" {
		conn, participantServiceClient, campaignServiceClient, sessionServiceClient, characterServiceClient, inviteServiceClient, err := dialGameGRPC(ctx, config)
		if err != nil {
			log.Printf("game gRPC dial failed, campaign access checks disabled: %v", err)
		} else {
			gameConn = conn
			participantClient = participantServiceClient
			campaignClient = campaignServiceClient
			sessionClient = sessionServiceClient
			characterClient = characterServiceClient
			inviteClient = inviteServiceClient
		}
	}
	campaignAccess := newCampaignAccessChecker(config, participantClient)
	handler, err := NewHandlerWithCampaignAccess(config, authClient, handlerDependencies{
		campaignAccess:    campaignAccess,
		campaignClient:    campaignClient,
		sessionClient:     sessionClient,
		participantClient: participantClient,
		characterClient:   characterClient,
		inviteClient:      inviteClient,
	})
	if err != nil {
		return nil, fmt.Errorf("build handler: %w", err)
	}
	httpServer := &http.Server{
		Addr:              httpAddr,
		Handler:           handler,
		ReadHeaderTimeout: timeouts.ReadHeader,
	}

	return &Server{
		httpAddr:   httpAddr,
		httpServer: httpServer,
		authConn:   authConn,
		gameConn:   gameConn,
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
func dialAuthGRPC(ctx context.Context, config Config) (*grpc.ClientConn, authv1.AuthServiceClient, error) {
	authAddr := strings.TrimSpace(config.AuthAddr)
	if authAddr == "" {
		return nil, nil, nil
	}
	if ctx == nil {
		return nil, nil, errors.New("context is required")
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
				return nil, nil, fmt.Errorf("auth gRPC health check failed for %s: %w", authAddr, dialErr.Err)
			}
			return nil, nil, fmt.Errorf("dial auth gRPC %s: %w", authAddr, dialErr.Err)
		}
		return nil, nil, fmt.Errorf("dial auth gRPC %s: %w", authAddr, err)
	}
	client := authv1.NewAuthServiceClient(conn)
	return conn, client, nil
}

// dialGameGRPC returns clients for campaign/character/session/invite operations.
// This dependency is optional by design so campaign routes can degrade gracefully
// during partial service outages.
func dialGameGRPC(ctx context.Context, config Config) (*grpc.ClientConn, statev1.ParticipantServiceClient, statev1.CampaignServiceClient, statev1.SessionServiceClient, statev1.CharacterServiceClient, statev1.InviteServiceClient, error) {
	gameAddr := strings.TrimSpace(config.GameAddr)
	if gameAddr == "" {
		return nil, nil, nil, nil, nil, nil, nil
	}
	if ctx == nil {
		return nil, nil, nil, nil, nil, nil, errors.New("context is required")
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
				return nil, nil, nil, nil, nil, nil, fmt.Errorf("game gRPC health check failed for %s: %w", gameAddr, dialErr.Err)
			}
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("dial game gRPC %s: %w", gameAddr, dialErr.Err)
		}
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("dial game gRPC %s: %w", gameAddr, err)
	}
	participantClient := statev1.NewParticipantServiceClient(conn)
	campaignClient := statev1.NewCampaignServiceClient(conn)
	sessionClient := statev1.NewSessionServiceClient(conn)
	characterClient := statev1.NewCharacterServiceClient(conn)
	inviteClient := statev1.NewInviteServiceClient(conn)
	return conn, participantClient, campaignClient, sessionClient, characterClient, inviteClient, nil
}

// handlePasskeyLoginStart begins a passkey authentication round trip and returns
// the credential challenge expected by browser/WebAuth clients.
func (h *handler) handlePasskeyLoginStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h == nil || h.authClient == nil {
		http.Error(w, "auth client not configured", http.StatusInternalServerError)
		return
	}

	var payload struct {
		PendingID string `json:"pending_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(payload.PendingID) == "" {
		http.Error(w, "pending_id is required", http.StatusBadRequest)
		return
	}

	resp, err := h.authClient.BeginPasskeyLogin(r.Context(), &authv1.BeginPasskeyLoginRequest{})
	if err != nil {
		http.Error(w, "failed to start passkey login", http.StatusBadRequest)
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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h == nil || h.authClient == nil {
		http.Error(w, "auth client not configured", http.StatusInternalServerError)
		return
	}

	var payload struct {
		PendingID  string          `json:"pending_id"`
		SessionID  string          `json:"session_id"`
		Credential json.RawMessage `json:"credential"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(payload.PendingID) == "" {
		http.Error(w, "pending_id is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(payload.SessionID) == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}
	if len(payload.Credential) == 0 {
		http.Error(w, "credential is required", http.StatusBadRequest)
		return
	}

	_, err := h.authClient.FinishPasskeyLogin(r.Context(), &authv1.FinishPasskeyLoginRequest{
		SessionId:              payload.SessionID,
		CredentialResponseJson: payload.Credential,
		PendingId:              payload.PendingID,
	})
	if err != nil {
		http.Error(w, "failed to finish passkey login", http.StatusBadRequest)
		return
	}

	redirectURL := buildAuthConsentURL(h.config.AuthBaseURL, payload.PendingID)
	writeJSON(w, http.StatusOK, map[string]any{"redirect_url": redirectURL})
}

// handlePasskeyRegisterStart creates a new passkey credential request so users can
// onboard a new WebAuth identity without leaving the current auth flow.
func (h *handler) handlePasskeyRegisterStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h == nil || h.authClient == nil {
		http.Error(w, "auth client not configured", http.StatusInternalServerError)
		return
	}

	var payload struct {
		Email     string `json:"email"`
		PendingID string `json:"pending_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(payload.Email) == "" {
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}

	createResp, err := h.authClient.CreateUser(r.Context(), &authv1.CreateUserRequest{PrimaryEmail: payload.Email})
	if err != nil || createResp.GetUser() == nil {
		http.Error(w, "failed to create user", http.StatusBadRequest)
		return
	}

	beginResp, err := h.authClient.BeginPasskeyRegistration(r.Context(), &authv1.BeginPasskeyRegistrationRequest{UserId: createResp.GetUser().GetId()})
	if err != nil {
		http.Error(w, "failed to start passkey registration", http.StatusBadRequest)
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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h == nil || h.authClient == nil {
		http.Error(w, "auth client not configured", http.StatusInternalServerError)
		return
	}

	var payload struct {
		PendingID  string          `json:"pending_id"`
		SessionID  string          `json:"session_id"`
		UserID     string          `json:"user_id"`
		Credential json.RawMessage `json:"credential"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(payload.SessionID) == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(payload.UserID) == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}
	if len(payload.Credential) == 0 {
		http.Error(w, "credential is required", http.StatusBadRequest)
		return
	}

	_, err := h.authClient.FinishPasskeyRegistration(r.Context(), &authv1.FinishPasskeyRegistrationRequest{
		SessionId:              payload.SessionID,
		CredentialResponseJson: payload.Credential,
	})
	if err != nil {
		http.Error(w, "failed to finish passkey registration", http.StatusBadRequest)
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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	printer, lang := localizer(w, r)
	if h == nil || h.authClient == nil {
		renderMagicPage(w, r, http.StatusInternalServerError, webtemplates.MagicParams{
			AppName: branding.AppName,
			Title:   printer.Sprintf("magic.unavailable.title"),
			Message: printer.Sprintf("magic.unavailable.message"),
			Detail:  printer.Sprintf("magic.unavailable.detail"),
			Success: false,
			Lang:    lang,
		})
		return
	}

	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		renderMagicPage(w, r, http.StatusBadRequest, webtemplates.MagicParams{
			AppName: branding.AppName,
			Title:   printer.Sprintf("magic.missing.title"),
			Message: printer.Sprintf("magic.missing.message"),
			Detail:  printer.Sprintf("magic.missing.detail"),
			Success: false,
			Lang:    lang,
		})
		return
	}

	resp, err := h.authClient.ConsumeMagicLink(r.Context(), &authv1.ConsumeMagicLinkRequest{Token: token})
	if err != nil {
		renderMagicPage(w, r, http.StatusBadRequest, webtemplates.MagicParams{
			AppName: branding.AppName,
			Title:   printer.Sprintf("magic.invalid.title"),
			Message: printer.Sprintf("magic.invalid.message"),
			Detail:  printer.Sprintf("magic.invalid.detail"),
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

	renderMagicPage(w, r, http.StatusOK, webtemplates.MagicParams{
		AppName:   branding.AppName,
		Title:     printer.Sprintf("magic.verified.title"),
		Message:   printer.Sprintf("magic.verified.message"),
		Detail:    printer.Sprintf("magic.verified.detail"),
		Success:   true,
		LinkURL:   "/",
		LinkLabel: printer.Sprintf("magic.verified.link"),
		Lang:      lang,
	})
}

// renderMagicPage writes the status code and renders the magic-link templ page.
func renderMagicPage(w http.ResponseWriter, r *http.Request, status int, params webtemplates.MagicParams) {
	w.WriteHeader(status)
	templ.Handler(webtemplates.MagicPage(params)).ServeHTTP(w, r)
}

// handleAuthLogin initiates the OAuth PKCE flow by redirecting to the auth server
// with state and challenge that ties browser and token exchange together.
func (h *handler) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	verifier, err := generateCodeVerifier()
	if err != nil {
		http.Error(w, "failed to generate PKCE verifier", http.StatusInternalServerError)
		return
	}
	challenge := computeS256Challenge(verifier)
	state := h.pendingFlows.create(verifier)

	authorizeURL := strings.TrimRight(strings.TrimSpace(h.config.AuthBaseURL), "/") + "/authorize"
	redirectURL, err := url.Parse(authorizeURL)
	if err != nil {
		http.Error(w, "invalid auth base url", http.StatusInternalServerError)
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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	state := strings.TrimSpace(r.URL.Query().Get("state"))

	if code == "" || state == "" {
		http.Error(w, "missing code or state", http.StatusBadRequest)
		return
	}

	flow := h.pendingFlows.consume(state)
	if flow == nil {
		http.Error(w, "invalid or expired state", http.StatusBadRequest)
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
		http.Error(w, "token exchange failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "token exchange returned "+resp.Status, http.StatusBadGateway)
		return
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		http.Error(w, "failed to decode token response", http.StatusBadGateway)
		return
	}

	if tokenResp.AccessToken == "" {
		http.Error(w, "empty access token", http.StatusBadGateway)
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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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
