package gametools

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/shared/mcpbridge"
)

// DirectDialer implements orchestration.Dialer by constructing a DirectSession
// bound to the campaign/session/participant authority from context.
type DirectDialer struct {
	clients Clients
}

// NewDirectDialer creates a dialer that builds direct gRPC sessions.
func NewDirectDialer(clients Clients) *DirectDialer {
	return &DirectDialer{clients: clients}
}

// Dial extracts the session context from ctx and returns a DirectSession.
func (d *DirectDialer) Dial(ctx context.Context) (orchestration.Session, error) {
	sc := mcpbridge.SessionContextFromContext(ctx)
	return NewDirectSession(d.clients, sessionContext{
		CampaignID:    sc.CampaignID,
		SessionID:     sc.SessionID,
		ParticipantID: sc.ParticipantID,
	}), nil
}
