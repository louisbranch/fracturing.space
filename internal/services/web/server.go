// Package web hosts the clean-slate browser-facing service.
package web

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	webapp "github.com/louisbranch/fracturing.space/internal/services/web/app"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/observability"
	webstatic "github.com/louisbranch/fracturing.space/internal/services/web/static"
)

// Config defines startup inputs for the web service.
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
	AuthorizationClient       module.AuthorizationClient
	AuthClient                module.AuthClient
	AccountClient             module.AccountClient
	CredentialClient          module.CredentialClient
	SocialClient              socialv1.SocialServiceClient
}

// Server hosts the web HTTP surface and lifecycle.
type Server struct {
	httpAddr   string
	httpServer *http.Server
}

// NewHandler builds a root handler from default module registry groups.
func NewHandler(cfg Config) (http.Handler, error) {
	principal := newPrincipalResolver(cfg)
	deps := module.Dependencies{
		CampaignClient:      cfg.CampaignClient,
		ParticipantClient:   cfg.ParticipantClient,
		CharacterClient:     cfg.CharacterClient,
		SessionClient:       cfg.SessionClient,
		InviteClient:        cfg.InviteClient,
		AuthorizationClient: cfg.AuthorizationClient,
		AuthClient:          cfg.AuthClient,
		AccountClient:       cfg.AccountClient,
		CredentialClient:    cfg.CredentialClient,
		SocialClient:        cfg.SocialClient,
		ResolveViewer:       principal.resolveViewer,
		ResolveUserID:       principal.resolveRequestUserID,
		ResolveLanguage:     principal.resolveRequestLanguage,
		AssetBaseURL:        cfg.AssetBaseURL,
		ChatFallbackPort:    websupport.ResolveChatFallbackPort(cfg.ChatHTTPAddr),
	}
	publicModules := modules.DefaultPublicModules()
	protectedModules := modules.DefaultProtectedModules(deps)
	// TODO(web-cutover): revisit stable registry composition as parity gaps close so default surfaces never expose scaffold-only behavior.
	if cfg.EnableExperimentalModules {
		protectedModules = modules.DefaultProtectedModulesWithExperimentalCampaignRoutes(deps)
		publicModules = append(publicModules, modules.ExperimentalPublicModules()...)
		protectedModules = append(protectedModules, modules.ExperimentalProtectedModules(deps)...)
	}
	h, err := webapp.BuildRootHandler(webapp.Config{
		Dependencies:     deps,
		PublicModules:    publicModules,
		ProtectedModules: protectedModules,
	}, principal.authRequired())
	if err != nil {
		return nil, err
	}
	rootMux := http.NewServeMux()
	rootMux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(webstatic.FS))))
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

// NewServer validates config and constructs a web server.
func NewServer(_ context.Context, cfg Config) (*Server, error) {
	httpAddr := strings.TrimSpace(cfg.HTTPAddr)
	if httpAddr == "" {
		return nil, errors.New("http address is required")
	}
	handler, err := NewHandler(cfg)
	if err != nil {
		return nil, fmt.Errorf("compose web handler: %w", err)
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
		return errors.New("web server is nil")
	}
	if ctx == nil {
		return errors.New("context is required")
	}

	serveErr := make(chan error, 1)
	log.Printf("web server listening on %s", s.httpAddr)
	go func() {
		serveErr <- s.httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), timeouts.Shutdown)
		err := s.httpServer.Shutdown(shutdownCtx)
		cancel()
		if err != nil {
			return fmt.Errorf("shutdown web http server: %w", err)
		}
		return nil
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve web http: %w", err)
	}
}

// Close closes open server resources.
func (s *Server) Close() {
	if s == nil || s.httpServer == nil {
		return
	}
	_ = s.httpServer.Close()
}
