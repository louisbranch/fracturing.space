package server

import (
	"context"
	"log"
	"net"
	"time"

	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	storagesqlite "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
)

type Server struct {
	listener                                 net.Listener
	grpcServer                               *grpc.Server
	health                                   *health.Server
	stores                                   *storageBundle
	authMc                                   *platformgrpc.ManagedConn
	socialMc                                 *platformgrpc.ManagedConn
	aiMc                                     *platformgrpc.ManagedConn
	statusMc                                 *platformgrpc.ManagedConn
	statusBindDone                           <-chan struct{}
	statusBindCancel                         context.CancelFunc
	projectionApplyOutboxWorkerEnabled       bool
	projectionApplyOutboxApply               func(context.Context, event.Event) error
	projectionApplyOutboxShadowWorkerEnabled bool
	statusReporter                           *platformstatus.Reporter
	catalogReadyAtStartup                    bool
}

// projectionApplyOutboxShadowProcessor and projectionApplyOutboxProcessor use
// storage contracts directly so runtime worker orchestration does not depend on
// SQLite-specific types.
type projectionApplyOutboxShadowProcessor = storage.ProjectionApplyOutboxShadowProcessor
type projectionApplyOutboxProcessor = storage.ProjectionApplyOutboxProcessor

// projectionApplyStore is the projection-store contract needed by outbox apply
// wiring. It combines exactly-once apply with system-adapter store binding.
type projectionApplyStore interface {
	storage.ProjectionApplyExactlyOnceStore
}

// Projection worker defaults balance recovery speed versus DB churn.
const (
	projectionApplyOutboxWorkerInterval       = 2 * time.Second
	projectionApplyOutboxWorkerBatch          = 64
	projectionApplyOutboxShadowWorkerInterval = 2 * time.Second
	projectionApplyOutboxShadowWorkerBatch    = 64
)

// storageBundle groups the three SQLite stores and manages their lifecycle.
//
// Events are the source of truth, projections feed APIs, and content stores
// enrich projection reads for system-specific metadata.
type storageBundle struct {
	events      *storagesqlite.Store
	projections *storagesqlite.Store
	content     *storagesqlite.Store
}

// Close closes all stores in the bundle, logging any errors.
func (b *storageBundle) Close() {
	if b == nil {
		return
	}
	if b.events != nil {
		if err := b.events.Close(); err != nil {
			log.Printf("close event store: %v", err)
		}
	}
	if b.projections != nil {
		if err := b.projections.Close(); err != nil {
			log.Printf("close projection store: %v", err)
		}
	}
	if b.content != nil {
		if err := b.content.Close(); err != nil {
			log.Printf("close content store: %v", err)
		}
	}
}
