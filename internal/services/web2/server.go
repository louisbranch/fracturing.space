// Package web2 hosts the clean-slate browser-facing service.
package web2

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	web2app "github.com/louisbranch/fracturing.space/internal/services/web2/app"
	module "github.com/louisbranch/fracturing.space/internal/services/web2/module"
	"github.com/louisbranch/fracturing.space/internal/services/web2/modules"
	"github.com/louisbranch/fracturing.space/internal/services/web2/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web2/platform/observability"
	web2static "github.com/louisbranch/fracturing.space/internal/services/web2/static"
)

// Config defines startup inputs for the web2 service.
type Config struct {
	HTTPAddr                  string
	AssetBaseURL              string
	ChatHTTPAddr              string
	EnableExperimentalModules bool
	CampaignClient            module.CampaignClient
	ParticipantClient         module.ParticipantClient
	CharacterClient           module.CharacterClient
	SessionClient             module.SessionClient
	InviteClient              module.InviteClient
	AuthClient                module.AuthClient
	AccountClient             module.AccountClient
	CredentialClient          module.CredentialClient
	ConnectionsClient         connectionsv1.ConnectionsServiceClient
}

// Server hosts the web2 HTTP surface and lifecycle.
type Server struct {
	httpAddr   string
	httpServer *http.Server
}

// NewHandler builds a root handler from default module registry groups.
func NewHandler(cfg Config) (http.Handler, error) {
	principal := newPrincipalResolver(cfg)
	deps := module.Dependencies{
		CampaignClient:    cfg.CampaignClient,
		ParticipantClient: cfg.ParticipantClient,
		CharacterClient:   cfg.CharacterClient,
		SessionClient:     cfg.SessionClient,
		InviteClient:      cfg.InviteClient,
		AuthClient:        cfg.AuthClient,
		AccountClient:     cfg.AccountClient,
		CredentialClient:  cfg.CredentialClient,
		ConnectionsClient: cfg.ConnectionsClient,
		ResolveViewer:     principal.resolveViewer,
		ResolveUserID:     principal.resolveRequestUserID,
		ResolveLanguage:   principal.resolveRequestLanguage,
		AssetBaseURL:      cfg.AssetBaseURL,
		ChatFallbackPort:  websupport.ResolveChatFallbackPort(cfg.ChatHTTPAddr),
	}
	publicModules := modules.DefaultPublicModules()
	protectedModules := modules.DefaultProtectedModules(deps)
	// TODO(web2-cutover): revisit stable registry composition as parity gaps close so default surfaces never expose scaffold-only behavior.
	if cfg.EnableExperimentalModules {
		protectedModules = modules.DefaultProtectedModulesWithExperimentalCampaignRoutes(deps)
		publicModules = append(publicModules, modules.ExperimentalPublicModules()...)
		protectedModules = append(protectedModules, modules.ExperimentalProtectedModules(deps)...)
	}
	h, err := web2app.BuildRootHandler(web2app.Config{
		Dependencies:     deps,
		PublicModules:    publicModules,
		ProtectedModules: protectedModules,
	}, principal.authRequired())
	if err != nil {
		return nil, err
	}
	rootMux := http.NewServeMux()
	rootMux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(web2static.FS))))
	rootMux.Handle("/", h)
	return httpx.Chain(rootMux,
		httpx.RecoverPanic(),
		httpx.RequestID(),
		withRequestPrincipalState(),
		observability.RequestLogger(log.Default()),
	), nil
}

func withRequestPrincipalState() httpx.Middleware {
	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.NotFoundHandler()
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r == nil {
				next.ServeHTTP(w, r)
				return
			}
			state := &requestPrincipalState{}
			ctx := context.WithValue(r.Context(), requestPrincipalStateKey{}, state)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func requestPrincipalStateFromRequest(r *http.Request) *requestPrincipalState {
	if r == nil {
		return nil
	}
	return requestPrincipalStateFromContext(r.Context())
}

func requestPrincipalStateFromContext(ctx context.Context) *requestPrincipalState {
	if ctx == nil {
		return nil
	}
	state, _ := ctx.Value(requestPrincipalStateKey{}).(*requestPrincipalState)
	return state
}

// NewServer validates config and constructs a web2 server.
func NewServer(_ context.Context, cfg Config) (*Server, error) {
	httpAddr := strings.TrimSpace(cfg.HTTPAddr)
	if httpAddr == "" {
		return nil, errors.New("http address is required")
	}
	handler, err := NewHandler(cfg)
	if err != nil {
		return nil, fmt.Errorf("compose web2 handler: %w", err)
	}
	return &Server{
		httpAddr: httpAddr,
		httpServer: &http.Server{
			Addr:              httpAddr,
			Handler:           handler,
			ReadHeaderTimeout: timeouts.ReadHeader,
		},
	}, nil
}

// ListenAndServe serves HTTP traffic until context cancellation or server stop.
func (s *Server) ListenAndServe(ctx context.Context) error {
	if s == nil {
		return errors.New("web2 server is nil")
	}
	if ctx == nil {
		return errors.New("context is required")
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- s.httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), timeouts.Shutdown)
		err := s.httpServer.Shutdown(shutdownCtx)
		cancel()
		if err != nil {
			return fmt.Errorf("shutdown web2 http server: %w", err)
		}
		return nil
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve web2 http: %w", err)
	}
}

// Close closes open server resources.
func (s *Server) Close() {
	if s == nil || s.httpServer == nil {
		return
	}
	_ = s.httpServer.Close()
}
