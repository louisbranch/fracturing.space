// Package web hosts the clean-slate browser-facing service.
package web

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"github.com/louisbranch/fracturing.space/internal/services/web/composition"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/observability"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	webstatic "github.com/louisbranch/fracturing.space/internal/services/web/static"
)

// Config defines startup inputs for the web service.
type Config struct {
	HTTPAddr                  string
	ChatHTTPAddr              string
	EnableExperimentalModules bool

	// RequestSchemePolicy controls scheme resolution for proxy headers.
	RequestSchemePolicy requestmeta.SchemePolicy

	// Dependencies carries startup dependencies in one place for principal resolution
	// and module registry construction.
	Dependencies *DependencyBundle
}

// Server hosts the web HTTP surface and lifecycle.
type Server struct {
	httpAddr   string
	httpServer *http.Server
}

// NewHandler builds a root handler from default module registry groups.
func NewHandler(cfg Config) (http.Handler, error) {
	deps := DependencyBundle{}
	if cfg.Dependencies != nil {
		deps = *cfg.Dependencies
	}

	session := newSessionResolver(deps.Principal.SessionClient)
	viewer := newViewerResolver(deps.Principal.SocialClient, deps.Principal.NotificationClient, deps.Principal.AssetBaseURL, session.resolveRequestUserID)
	lang := newLanguageResolver(deps.Principal.AccountClient, session.resolveRequestUserID)
	h, err := composition.ComposeAppHandler(composition.ComposeInput{
		Principal: composition.PrincipalResolvers{
			AuthRequired:    session.authRequired(),
			ResolveViewer:   viewer.resolveViewer,
			ResolveSignedIn: session.resolveRequestSignedIn,
			ResolveUserID:   session.resolveRequestUserID,
			ResolveLanguage: lang.resolveRequestLanguage,
		},
		ModuleDependencies:        deps.Modules,
		EnableExperimentalModules: cfg.EnableExperimentalModules,
		ChatHTTPAddr:              cfg.ChatHTTPAddr,
		RequestSchemePolicy:       cfg.RequestSchemePolicy,
	})
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
