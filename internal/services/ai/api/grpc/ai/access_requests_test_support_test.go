package ai

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/ai/auditevent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// accessRequestTestHandlers bundles the service and handler so tests can
// configure clock/ID generator on the service while calling RPC methods on
// the handler.
type accessRequestTestHandlers struct {
	*AccessRequestHandlers
	svc *service.AccessRequestService
}

func newAccessRequestHandlersWithStores(t *testing.T, agentStore storage.AgentStore, accessRequestStore storage.AccessRequestStore, auditEventStore auditevent.Store) *accessRequestTestHandlers {
	t.Helper()
	svc, err := service.NewAccessRequestService(service.AccessRequestServiceConfig{
		AgentStore:         agentStore,
		AccessRequestStore: accessRequestStore,
		AuditEventStore:    auditEventStore,
	})
	if err != nil {
		t.Fatalf("NewAccessRequestService: %v", err)
	}
	h, err := NewAccessRequestHandlers(AccessRequestHandlersConfig{
		AccessRequestService: svc,
	})
	if err != nil {
		t.Fatalf("NewAccessRequestHandlers: %v", err)
	}
	return &accessRequestTestHandlers{AccessRequestHandlers: h, svc: svc}
}
