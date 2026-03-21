package ai

import (
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// AccessRequestHandlers serves access-request and audit-event RPCs.
type AccessRequestHandlers struct {
	aiv1.UnimplementedAccessRequestServiceServer

	agentStore         storage.AgentStore
	accessRequestStore storage.AccessRequestStore
	auditEventStore    storage.AuditEventStore
	clock              func() time.Time
	idGenerator        func() (string, error)
}

// AccessRequestHandlersConfig declares the dependencies for access-request RPCs.
type AccessRequestHandlersConfig struct {
	AgentStore         storage.AgentStore
	AccessRequestStore storage.AccessRequestStore
	AuditEventStore    storage.AuditEventStore
	Clock              func() time.Time
	IDGenerator        func() (string, error)
}

// NewAccessRequestHandlers builds an access-request RPC server from explicit deps.
func NewAccessRequestHandlers(cfg AccessRequestHandlersConfig) *AccessRequestHandlers {
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	idGenerator := cfg.IDGenerator
	if idGenerator == nil {
		idGenerator = id.NewID
	}
	return &AccessRequestHandlers{
		agentStore:         cfg.AgentStore,
		accessRequestStore: cfg.AccessRequestStore,
		auditEventStore:    cfg.AuditEventStore,
		clock:              clock,
		idGenerator:        idGenerator,
	}
}
