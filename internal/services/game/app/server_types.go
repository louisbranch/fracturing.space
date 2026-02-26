package server

import (
	"context"
	"log"
	"net"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	storagesqlite "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
)

type Server struct {
	listener                                 net.Listener
	grpcServer                               *grpc.Server
	health                                   *health.Server
	stores                                   *storageBundle
	authConn                                 *grpc.ClientConn
	socialConn                               *grpc.ClientConn
	projectionApplyOutboxWorkerEnabled       bool
	projectionApplyOutboxApply               func(context.Context, event.Event) error
	projectionApplyOutboxShadowWorkerEnabled bool
}

type authGRPCClients struct {
	conn       *grpc.ClientConn
	authClient authv1.AuthServiceClient
}

type socialGRPCClients struct {
	conn         *grpc.ClientConn
	socialClient socialv1.SocialServiceClient
}

// projectionApplyOutboxShadowProcessor drains queue rows for environments where the
// main apply worker is intentionally delayed or disabled.
type projectionApplyOutboxShadowProcessor interface {
	ProcessProjectionApplyOutboxShadow(context.Context, time.Time, int) (int, error)
}

// projectionApplyOutboxProcessor is responsible for applying queued events to projections.
//
// It keeps event ingestion and projection side effects separated from request path
// responsiveness while still converging read models in the background.
type projectionApplyOutboxProcessor interface {
	ProcessProjectionApplyOutbox(context.Context, time.Time, int, func(context.Context, event.Event) error) (int, error)
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
