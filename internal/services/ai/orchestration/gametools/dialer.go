package gametools

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/shared/mcpbridge"
)

// DirectDialer implements orchestration.Dialer by constructing a DirectSession
// bound to the campaign/session/participant authority from context.
type DirectDialer struct {
	clients  Clients
	registry productionToolRegistry
}

// NewDirectDialer creates a dialer that builds direct gRPC sessions using the
// default production tool registry.
func NewDirectDialer(clients Clients) *DirectDialer {
	return &DirectDialer{clients: clients, registry: defaultRegistry}
}

// Dial extracts the session context from ctx and returns a DirectSession.
func (d *DirectDialer) Dial(ctx context.Context) (orchestration.Session, error) {
	sc := mcpbridge.SessionContextFromContext(ctx)
	return newDirectSession(d.clients, d.registry, SessionContext{
		CampaignID:    sc.CampaignID,
		SessionID:     sc.SessionID,
		ParticipantID: sc.ParticipantID,
	}), nil
}
