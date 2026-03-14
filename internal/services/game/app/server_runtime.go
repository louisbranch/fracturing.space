package server

import (
	"context"
	"log"
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
		s.startProjectionApplyOutboxWorker(ctx),
		s.startProjectionApplyOutboxShadowWorker(ctx),
		s.startStatusReporter(ctx),
		s.startCatalogAvailabilityMonitor(ctx),
	)
	defer stopAncillaryWorkers()

	log.Printf("game server listening at %v", s.listener.Addr())
	return runGRPCServeLoop(
		ctx,
		func() error {
			return s.grpcServer.Serve(s.listener)
		},
		func() {
			if s.health != nil {
				s.health.Shutdown()
			}
			s.grpcServer.GracefulStop()
		},
	)
}

// startProjectionApplyOutboxShadowWorker launches an optional background shadow worker.
//
// This keeps pending queue items progressing when projection updates are not
// processed inline.
func (s *Server) startProjectionApplyOutboxShadowWorker(ctx context.Context) func() {
	if s == nil || !s.projectionApplyOutboxShadowWorkerEnabled || s.stores == nil || s.stores.events == nil {
		return func() {}
	}

	return startCancelableLoop(ctx, func(workerCtx context.Context) {
		runProjectionApplyOutboxShadowWorker(
			workerCtx,
			s.stores.events,
			projectionApplyOutboxShadowWorkerInterval,
			projectionApplyOutboxShadowWorkerBatch,
			time.Now,
			log.Printf,
		)
	})
}

// startProjectionApplyOutboxWorker launches an optional background projection worker.
//
// The worker applies queued projection rows independently from request handling.
func (s *Server) startProjectionApplyOutboxWorker(ctx context.Context) func() {
	if s == nil || !s.projectionApplyOutboxWorkerEnabled || s.stores == nil || s.stores.events == nil || s.projectionApplyOutboxApply == nil {
		return func() {}
	}

	return startCancelableLoop(ctx, func(workerCtx context.Context) {
		runProjectionApplyOutboxWorker(
			workerCtx,
			s.stores.events,
			s.projectionApplyOutboxApply,
			projectionApplyOutboxWorkerInterval,
			projectionApplyOutboxWorkerBatch,
			time.Now,
			log.Printf,
		)
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
	if s.statusBindCancel != nil {
		s.statusBindCancel()
	}
	if s.statusBindDone != nil {
		<-s.statusBindDone
	}
	closeManagedConn(s.statusMc, "status")
	closeManagedConn(s.aiMc, "ai")
	closeManagedConn(s.socialMc, "social")
	closeManagedConn(s.notificationsMc, "notifications")
	closeManagedConn(s.authMc, "auth")
}

func closeManagedConn(mc *platformgrpc.ManagedConn, name string) {
	if mc == nil {
		return
	}
	if err := mc.Close(); err != nil {
		log.Printf("close %s managed conn: %v", name, err)
	}
}
