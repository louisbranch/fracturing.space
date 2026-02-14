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

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/branding"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config defines the inputs for the web login server.
type Config struct {
	HTTPAddr        string
	AuthBaseURL     string
	AuthAddr        string
	AppName         string
	GRPCDialTimeout time.Duration
}

// Server hosts the web login HTTP server.
type Server struct {
	httpAddr   string
	httpServer *http.Server
	authConn   *grpc.ClientConn
}

type loginView struct {
	AppName      string
	PendingID    string
	ClientID     string
	ClientName   string
	AuthLoginURL string
	Error        string
}

type magicView struct {
	AppName   string
	Title     string
	Message   string
	Detail    string
	Success   bool
	LinkURL   string
	LinkLabel string
}

type handler struct {
	config     Config
	authClient authv1.AuthServiceClient
}

// NewHandler creates the HTTP handler for the login UX.
func NewHandler(config Config, authClient authv1.AuthServiceClient) http.Handler {
	mux := http.NewServeMux()
	staticFS, err := fs.Sub(assetsFS, "static")
	if err != nil {
		panic(err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	authLoginURL := buildAuthLoginURL(config.AuthBaseURL)
	appName := strings.TrimSpace(config.AppName)
	if appName == "" {
		appName = branding.AppName
	}
	h := &handler{config: config, authClient: authClient}

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		pendingID := strings.TrimSpace(r.URL.Query().Get("pending_id"))
		if pendingID == "" {
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

		view := loginView{
			AppName:      appName,
			PendingID:    pendingID,
			ClientID:     clientID,
			ClientName:   clientName,
			AuthLoginURL: authLoginURL,
			Error:        errorMessage,
		}
		if err := templates.ExecuteTemplate(w, "login.html", view); err != nil {
			http.Error(w, "failed to render login", http.StatusInternalServerError)
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

	return mux
}

// NewServer builds a configured web server.
func NewServer(config Config) (*Server, error) {
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
		conn, client, err := dialAuthGRPC(context.Background(), config)
		if err != nil {
			return nil, fmt.Errorf("dial auth grpc: %w", err)
		}
		authConn = conn
		authClient = client
	}

	handler := NewHandler(config, authClient)
	httpServer := &http.Server{
		Addr:              httpAddr,
		Handler:           handler,
		ReadHeaderTimeout: timeouts.ReadHeader,
	}

	return &Server{
		httpAddr:   httpAddr,
		httpServer: httpServer,
		authConn:   authConn,
	}, nil
}

// ListenAndServe runs the HTTP server until the context ends.
func (s *Server) ListenAndServe(ctx context.Context) error {
	if s == nil {
		return errors.New("web server is nil")
	}
	if ctx == nil {
		ctx = context.Background()
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
	if s == nil || s.authConn == nil {
		return
	}
	if err := s.authConn.Close(); err != nil {
		log.Printf("close auth gRPC connection: %v", err)
	}
}

func buildAuthLoginURL(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return "/authorize/login"
	}
	return strings.TrimRight(base, "/") + "/authorize/login"
}

func buildAuthConsentURL(base string, pendingID string) string {
	base = strings.TrimSpace(base)
	encoded := url.QueryEscape(pendingID)
	if base == "" {
		return "/authorize/consent?pending_id=" + encoded
	}
	return strings.TrimRight(base, "/") + "/authorize/consent?pending_id=" + encoded
}

func dialAuthGRPC(ctx context.Context, config Config) (*grpc.ClientConn, authv1.AuthServiceClient, error) {
	authAddr := strings.TrimSpace(config.AuthAddr)
	if authAddr == "" {
		return nil, nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	dialCtx, cancel := context.WithTimeout(ctx, config.GRPCDialTimeout)
	defer cancel()

	conn, err := grpc.DialContext(
		dialCtx,
		authAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, nil, err
	}
	client := authv1.NewAuthServiceClient(conn)
	return conn, client, nil
}

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
		DisplayName string `json:"display_name"`
		PendingID   string `json:"pending_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(payload.DisplayName) == "" {
		http.Error(w, "display_name is required", http.StatusBadRequest)
		return
	}

	createResp, err := h.authClient.CreateUser(r.Context(), &authv1.CreateUserRequest{DisplayName: payload.DisplayName})
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

func (h *handler) handleMagicLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h == nil || h.authClient == nil {
		renderMagicPage(w, http.StatusInternalServerError, magicView{
			AppName: branding.AppName,
			Title:   "Magic link unavailable",
			Message: "We could not reach the authentication service.",
			Detail:  "Please try again in a moment.",
			Success: false,
		})
		return
	}

	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		renderMagicPage(w, http.StatusBadRequest, magicView{
			AppName: branding.AppName,
			Title:   "Magic link missing",
			Message: "This link is missing its token.",
			Detail:  "Please request a new magic link and try again.",
			Success: false,
		})
		return
	}

	resp, err := h.authClient.ConsumeMagicLink(r.Context(), &authv1.ConsumeMagicLinkRequest{Token: token})
	if err != nil {
		renderMagicPage(w, http.StatusBadRequest, magicView{
			AppName: branding.AppName,
			Title:   "Magic link invalid",
			Message: "We could not validate this magic link.",
			Detail:  "It may have expired or already been used.",
			Success: false,
		})
		return
	}
	if pendingID := strings.TrimSpace(resp.GetPendingId()); pendingID != "" {
		redirectURL := buildAuthConsentURL(h.config.AuthBaseURL, pendingID)
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}

	renderMagicPage(w, http.StatusOK, magicView{
		AppName:   branding.AppName,
		Title:     "Magic link verified",
		Message:   "Your link is valid and your email has been confirmed.",
		Detail:    "You can return to the app and continue sign in.",
		Success:   true,
		LinkURL:   "/",
		LinkLabel: "Return to the app",
	})
}

func renderMagicPage(w http.ResponseWriter, status int, view magicView) {
	w.WriteHeader(status)
	if err := templates.ExecuteTemplate(w, "magic.html", view); err != nil {
		http.Error(w, "failed to render magic link page", http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	_ = encoder.Encode(payload)
}
