package server

import (
	"context"
	"log/slog"
	"net"
	"time"

	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
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

type eventBackend interface {
	storage.EventStore
	storage.AuditEventStore
	storage.EventIntegrityVerifier
	Close() error
	ProjectionApplyOutboxStore() storage.ProjectionApplyOutboxStore
	IntegrationOutboxStore() storage.IntegrationOutboxStore
}

type projectionBackend interface {
	storage.ProjectionStore
	storage.SessionGateStore
	storage.SessionSpotlightStore
	storage.SessionInteractionStore
	storage.SceneStore
	storage.SceneCharacterStore
	storage.SceneGateStore
	storage.SceneSpotlightStore
	storage.SceneInteractionStore
	storage.ProjectionApplyExactlyOnceStore
	Close() error
	DaggerheartProjectionStore() projectionstore.Store
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
// Events are the source of truth, projections feed APIs, and content stores
// enrich projection reads for system-specific metadata.
type storageBundle struct {
	events      eventBackend
	projections projectionBackend
	content     contentBackend
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
