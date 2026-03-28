package app

import (
	"context"
	"log/slog"
	"net"
	"time"

	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
)

// Server is the top-level game gRPC server. Fields are grouped into embedded
// sub-structs by concern to keep each area's surface small.
type Server struct {
	listener  net.Listener
	stores    *storageBundle
	transport transportState
	conns     connectionState
	workers   projectionWorkerState
	status    statusState
}

// transportState holds gRPC server and health infrastructure.
type transportState struct {
	grpcServer *grpc.Server
	health     *health.Server
}

// connectionState holds outbound managed connections and their lifecycle hooks.
type connectionState struct {
	authMc           *platformgrpc.ManagedConn
	socialMc         *platformgrpc.ManagedConn
	aiMc             *platformgrpc.ManagedConn
	statusMc         *platformgrpc.ManagedConn
	statusBindDone   <-chan struct{}
	statusBindCancel context.CancelFunc
}

// projectionWorkerState holds configuration for optional background projection
// apply and shadow workers.
type projectionWorkerState struct {
	applyWorkerEnabled  bool
	applyFunc           func(context.Context, event.Event) error
	shadowWorkerEnabled bool
}

// statusState holds status reporting and catalog readiness state.
type statusState struct {
	reporter              *platformstatus.Reporter
	catalogReadyAtStartup bool
}

// projectionApplyOutboxShadowProcessor and projectionApplyOutboxProcessor use
// storage contracts directly so runtime worker orchestration does not depend on
// SQLite-specific types.
type projectionApplyOutboxShadowProcessor = storage.ProjectionApplyOutboxShadowProcessor
type projectionApplyOutboxProcessor = storage.ProjectionApplyOutboxProcessor

type eventBackend interface {
	storage.EventStore
	storage.AuditEventStore
	storage.EventIntegrityVerifier
	Close() error
	ProjectionApplyOutboxStore() storage.ProjectionApplyOutboxStore
	IntegrationOutboxStore() storage.IntegrationOutboxWorkerStore
}

type projectionBackend interface {
	storage.ProjectionStore
	projection.ExactlyOnceStore
	Close() error
}

type contentBackend interface {
	contentstore.DaggerheartContentReadStore
	contentstore.DaggerheartCatalogReadinessStore
	Close() error
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
// Events are the source of truth, projections feed APIs, systemStores bind the
// built-in system query backends from the opened projection database, and
// content stores enrich projection reads for system-specific metadata.
type storageBundle struct {
	events       eventBackend
	projections  projectionBackend
	systemStores systemmanifest.ProjectionStores
	content      contentBackend
}

// Close closes all stores in the bundle, logging any errors.
func (b *storageBundle) Close() {
	if b == nil {
		return
	}
	if b.events != nil {
		if err := b.events.Close(); err != nil {
			slog.Error("close event store", "error", err)
		}
	}
	if b.projections != nil {
		if err := b.projections.Close(); err != nil {
			slog.Error("close projection store", "error", err)
		}
	}
	if b.content != nil {
		if err := b.content.Close(); err != nil {
			slog.Error("close content store", "error", err)
		}
	}
}
