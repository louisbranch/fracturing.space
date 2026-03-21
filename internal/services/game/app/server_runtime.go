package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func (s *Server) Addr() string {
	if s == nil || s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Run creates and serves a game server until the context ends.
func Run(ctx context.Context, port int) error {
	grpcServer, err := NewContext(ctx, port)
	if err != nil {
		return err
	}
	return grpcServer.Serve(ctx)
}

// RunWithAddr creates and serves a game server until the context ends.
func RunWithAddr(ctx context.Context, addr string) error {
	grpcServer, err := NewWithAddrContext(ctx, addr)
	if err != nil {
		return err
	}
	return grpcServer.Serve(ctx)
}

// Serve starts the game server and blocks until it stops or the context ends.
func (s *Server) Serve(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	defer s.closeResources()
	stopAncillaryWorkers := composeRuntimeStops(
		s.startProjectionWorker(ctx, projectionApplyWorkerKind),
		s.startProjectionWorker(ctx, projectionShadowWorkerKind),
		s.startStatusReporter(ctx),
		s.startCatalogAvailabilityMonitor(ctx),
	)
	defer stopAncillaryWorkers()

	slog.Info("game server listening", "addr", s.listener.Addr())
	return runGRPCServeLoop(
		ctx,
		func() error {
			return s.transport.grpcServer.Serve(s.listener)
		},
		func() {
			if s.transport.health != nil {
				s.transport.health.Shutdown()
			}
			platformgrpc.GracefulStopWithTimeout(s.transport.grpcServer, platformgrpc.DefaultGracefulStopTimeout)
		},
	)
}

// projectionWorkerKind selects the apply or shadow worker variant.
type projectionWorkerKind int

const (
	projectionApplyWorkerKind projectionWorkerKind = iota
	projectionShadowWorkerKind
)

// startProjectionWorker launches an optional background projection worker of
// the given kind. It returns a stop function that blocks until the worker
// goroutine exits. The two worker variants (apply and shadow) share identical
// lifecycle logic and differ only in the processing function they invoke.
func (s *Server) startProjectionWorker(ctx context.Context, kind projectionWorkerKind) func() {
	if s == nil || s.stores == nil || s.stores.events == nil {
		return func() {}
	}

	var enabled bool
	switch kind {
	case projectionApplyWorkerKind:
		enabled = s.workers.applyWorkerEnabled && s.workers.applyFunc != nil
	case projectionShadowWorkerKind:
		enabled = s.workers.shadowWorkerEnabled
	}
	if !enabled {
		return func() {}
	}

	processor := s.stores.events.ProjectionApplyOutboxStore()
	if processor == nil {
		return func() {}
	}

	return startCancelableLoop(ctx, func(workerCtx context.Context) {
		switch kind {
		case projectionApplyWorkerKind:
			runProjectionApplyOutboxWorker(
				workerCtx,
				processor,
				s.workers.applyFunc,
				projectionApplyOutboxWorkerInterval,
				projectionApplyOutboxWorkerBatch,
				time.Now,
				slogInfof,
			)
		case projectionShadowWorkerKind:
			runProjectionApplyOutboxShadowWorker(
				workerCtx,
				processor,
				projectionApplyOutboxShadowWorkerInterval,
				projectionApplyOutboxShadowWorkerBatch,
				time.Now,
				slogInfof,
			)
		}
	})
}

// runProjectionApplyOutboxShadowWorker drains projection outbox shadow entries.
//
// It is intentionally lightweight: the purpose is progress cleanup, not full
// projection mutation.
func runProjectionApplyOutboxShadowWorker(
	ctx context.Context,
	processor projectionApplyOutboxShadowProcessor,
	interval time.Duration,
	limit int,
	now func() time.Time,
	logf func(string, ...any),
) {
	if processor == nil || interval <= 0 || limit <= 0 {
		return
	}
	if now == nil {
		now = time.Now
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}

	runPass := func() int {
		processed, err := processor.ProcessProjectionApplyOutboxShadow(ctx, now().UTC(), limit)
		if err != nil {
			logf("projection apply outbox shadow worker pass failed: %v", err)
			return 0
		}
		if processed > 0 {
			logf("projection apply outbox shadow worker observed %d rows", processed)
		}
		return processed
	}

	runBatchedPollingLoop(ctx, interval, limit, runPass)
}

// runProjectionApplyOutboxWorker drains projection outbox entries into projections.
//
// It loops in bounded batches until no rows remain, then waits for timer ticks.
func runProjectionApplyOutboxWorker(
	ctx context.Context,
	processor projectionApplyOutboxProcessor,
	apply func(context.Context, event.Event) error,
	interval time.Duration,
	limit int,
	now func() time.Time,
	logf func(string, ...any),
) {
	if processor == nil || apply == nil || interval <= 0 || limit <= 0 {
		return
	}
	if now == nil {
		now = time.Now
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}

	runPass := func() int {
		processed, err := processor.ProcessProjectionApplyOutbox(ctx, now().UTC(), limit, apply)
		if err != nil {
			logf("projection apply outbox worker pass failed: %v", err)
			return 0
		}
		if processed > 0 {
			logf("projection apply outbox worker applied %d rows", processed)
		}
		return processed
	}

	runBatchedPollingLoop(ctx, interval, limit, runPass)
}

// closeResources closes store handles and outbound connections.
//
// It is safe to call on a nil Server and is intended for shutdown paths.
func (s *Server) closeResources() {
	if s == nil {
		return
	}
	s.stores.Close()
	// Cancel and join the status-bind goroutine before closing the connection
	// it references.
	if s.conns.statusBindCancel != nil {
		s.conns.statusBindCancel()
	}
	if s.conns.statusBindDone != nil {
		<-s.conns.statusBindDone
	}
	closeManagedConn(s.conns.statusMc, "status")
	closeManagedConn(s.conns.aiMc, "ai")
	closeManagedConn(s.conns.socialMc, "social")
	closeManagedConn(s.conns.authMc, "auth")
}

func closeManagedConn(mc *platformgrpc.ManagedConn, name string) {
	if mc == nil {
		return
	}
	if err := mc.Close(); err != nil {
		slog.Error("close managed conn", "name", name, "error", err)
	}
}

// slogInfof adapts slog.Info for callers that accept a printf-style callback.
func slogInfof(format string, args ...any) {
	slog.Info(fmt.Sprintf(format, args...))
}
