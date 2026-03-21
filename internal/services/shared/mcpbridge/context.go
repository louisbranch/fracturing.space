package mcpbridge

import (
	"context"
	"strings"
)

type contextKey string

const sessionContextKey contextKey = "fracturing-space-mcp-bridge-session-context"

// SessionContext carries fixed authority for one orchestration session.
type SessionContext struct {
	CampaignID    string
	SessionID     string
	ParticipantID string
}

// Normalize trims whitespace from all fixed authority fields.
func (c SessionContext) Normalize() SessionContext {
	return SessionContext{
		CampaignID:    strings.TrimSpace(c.CampaignID),
		SessionID:     strings.TrimSpace(c.SessionID),
		ParticipantID: strings.TrimSpace(c.ParticipantID),
	}
}

// Valid reports whether the bridge context has all required fixed authority.
func (c SessionContext) Valid() bool {
	c = c.Normalize()
	return c.CampaignID != "" && c.SessionID != "" && c.ParticipantID != ""
}

// WithSessionContext stores one fixed orchestration context on ctx.
func WithSessionContext(ctx context.Context, sessionCtx SessionContext) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, sessionContextKey, sessionCtx.Normalize())
}

// SessionContextFromContext loads the fixed orchestration context from ctx.
func SessionContextFromContext(ctx context.Context) SessionContext {
	if ctx == nil {
		return SessionContext{}
	}
	sessionCtx, _ := ctx.Value(sessionContextKey).(SessionContext)
	return sessionCtx.Normalize()
}
