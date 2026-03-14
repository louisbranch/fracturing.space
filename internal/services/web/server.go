// Package web hosts the clean-slate browser-facing service.
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
	"github.com/louisbranch/fracturing.space/internal/services/web/composition"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/observability"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
	webstatic "github.com/louisbranch/fracturing.space/internal/services/web/static"
)

// Config defines startup inputs for the web service.
type Config struct {
	HTTPAddr     string
	ChatHTTPAddr string
	// Logger receives request and lifecycle logs. Nil uses slog.Default().
	Logger *slog.Logger

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
	logger     *slog.Logger
}

// NewHandler builds a root handler from default module registry groups.
func NewHandler(cfg Config) (http.Handler, error) {
	deps := DependencyBundle{}
	if cfg.Dependencies != nil {
		deps = *cfg.Dependencies
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	principalResolver := principal.New(deps.Principal)
	h, err := composition.ComposeAppHandler(composition.ComposeInput{
		Principal:           principalResolver,
		ModuleDependencies:  deps.Modules,
		ChatHTTPAddr:        cfg.ChatHTTPAddr,
		RequestSchemePolicy: cfg.RequestSchemePolicy,
	})
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
		logger: loggerOrDefault(cfg.Logger),
	}, nil
}

// loggerOrDefault normalizes nil logger inputs to the process default logger.
func loggerOrDefault(logger *slog.Logger) *slog.Logger {
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
	logger := loggerOrDefault(s.logger)
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
