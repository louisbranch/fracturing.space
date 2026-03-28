package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	sharedhttpx "github.com/louisbranch/fracturing.space/internal/services/shared/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	webapp "github.com/louisbranch/fracturing.space/internal/services/web/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/observability"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
	webstatic "github.com/louisbranch/fracturing.space/internal/services/web/static"
)

// Config defines startup inputs for the web service.
type Config struct {
	HTTPAddr     string
	PlayHTTPAddr string
	// Logger receives request and lifecycle logs. Nil uses slog.Default().
	Logger *slog.Logger

	// RequestSchemePolicy controls scheme resolution for proxy headers.
	RequestSchemePolicy requestmeta.SchemePolicy

	// PlayLaunchGrant signs web-to-play handoff grants for the game route.
	PlayLaunchGrant playlaunchgrant.Config

	// Dependencies carries startup dependencies in one place for principal resolution
	// and module registry construction.
	Dependencies *DependencyBundle

	// RegistryBuilder overrides the default module registry builder. Nil uses
	// the production default. Tests inject custom builders to control module
	// composition.
	RegistryBuilder modules.RegistryBuilder
}

// Server hosts the web HTTP surface and lifecycle.
type Server struct {
	httpAddr   string
	httpServer *http.Server
	logger     *slog.Logger
}

// composeHandler builds the root web handler from an explicit dependency
// bundle. Production constructors validate required dependencies before
// calling this helper; package tests may opt into partial dependency bundles
// through test-only helpers.
func composeHandler(cfg Config, deps DependencyBundle) (http.Handler, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	principalResolver := principal.New(deps.Principal)
	h, err := composeAppHandler(principalResolver, logger, cfg, deps)
	if err != nil {
		return nil, err
	}
	rootMux := http.NewServeMux()
	rootMux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(webstatic.FS))))
	rootMux.Handle("/", h)
	return sharedhttpx.Chain(rootMux,
		sharedhttpx.RecoverPanic(),
		sharedhttpx.RequestID("web"),
		principalResolver.Middleware(),
		observability.RequestLogger(logger),
	), nil
}

// composeAppHandler builds the web app handler by assembling the module
// registry and passing the resulting module sets to the root mux composer.
// This replaces the former composition/ package indirection.
func composeAppHandler(
	resolver principal.PrincipalResolver,
	logger *slog.Logger,
	cfg Config,
	deps DependencyBundle,
) (http.Handler, error) {
	registryBuilder := cfg.RegistryBuilder
	if registryBuilder == nil {
		registryBuilder = modules.NewRegistryBuilder()
	}
	var authRequired func(*http.Request) bool
	if resolver != nil {
		authRequired = resolver.AuthRequired
	}

	built := registryBuilder.Build(modules.RegistryInput{
		Dependencies: deps.Modules,
		Principal:    resolver,
		PublicOptions: modules.PublicModuleOptions{
			RequestSchemePolicy: cfg.RequestSchemePolicy,
			Logger:              logger,
		},
		ProtectedOptions: modules.ProtectedModuleOptions{
			PlayFallbackPort:    websupport.ResolveHTTPFallbackPort(cfg.PlayHTTPAddr),
			PlayLaunchGrant:     cfg.PlayLaunchGrant,
			RequestSchemePolicy: cfg.RequestSchemePolicy,
			Logger:              logger,
		},
	})

	return webapp.Compose(webapp.ComposeInput{
		AuthRequired:        authRequired,
		PublicModules:       built.Public,
		ProtectedModules:    built.Protected,
		RequestSchemePolicy: cfg.RequestSchemePolicy,
	})
}

// newServer wraps an already-composed root handler in an HTTP server with
// shared lifecycle defaults.
func newServer(cfg Config, handler http.Handler) (*Server, error) {
	httpAddr := strings.TrimSpace(cfg.HTTPAddr)
	if httpAddr == "" {
		return nil, errors.New("http address is required")
	}
	if handler == nil {
		return nil, errors.New("web handler is required")
	}
	return &Server{
		httpAddr: httpAddr,
		httpServer: &http.Server{
			Addr:              httpAddr,
			Handler:           handler,
			ReadHeaderTimeout: timeouts.ReadHeader,
		},
		logger: LoggerOrDefault(cfg.Logger),
	}, nil
}

// snapshotDependencyBundle returns a value copy of the configured dependency
// bundle, or the zero bundle when no dependencies were supplied.
func snapshotDependencyBundle(bundle *DependencyBundle) DependencyBundle {
	if bundle == nil {
		return DependencyBundle{}
	}
	return *bundle
}

// validatedDependencyBundle validates required startup dependencies and
// returns a value copy for handler composition. Production callers must use
// this path; test callers that accept partial bundles use snapshotDependencyBundle
// directly.
func validatedDependencyBundle(bundle *DependencyBundle) (DependencyBundle, error) {
	if err := validateRequiredDependencyBundle(bundle); err != nil {
		return DependencyBundle{}, err
	}
	return snapshotDependencyBundle(bundle), nil
}

// NewHandler builds a root handler from default module registry groups.
func NewHandler(cfg Config) (http.Handler, error) {
	deps, err := validatedDependencyBundle(cfg.Dependencies)
	if err != nil {
		return nil, err
	}
	return composeHandler(cfg, deps)
}

// NewServer validates config and constructs a web server.
func NewServer(_ context.Context, cfg Config) (*Server, error) {
	handler, err := NewHandler(cfg)
	if err != nil {
		return nil, err
	}
	return newServer(cfg, handler)
}

// LoggerOrDefault normalizes nil logger inputs to the process default logger.
func LoggerOrDefault(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		return slog.Default()
	}
	return logger
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
	logger := LoggerOrDefault(s.logger)
	logger.Info("web server listening", "addr", s.httpAddr)
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
