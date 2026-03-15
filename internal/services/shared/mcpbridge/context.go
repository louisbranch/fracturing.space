package mcpbridge

import (
	"context"
	"net/http"
	"strings"
)

const (
	// CampaignIDHeader fixes the campaign authority for one internal MCP session.
	CampaignIDHeader = "X-Fracturing-Space-MCP-Campaign-Id"
	// SessionIDHeader fixes the active session authority for one internal MCP session.
	SessionIDHeader = "X-Fracturing-Space-MCP-Session-Id"
	// ParticipantIDHeader fixes the acting participant authority for one internal MCP session.
	ParticipantIDHeader = "X-Fracturing-Space-MCP-Participant-Id"
)

type contextKey string

const sessionContextKey contextKey = "fracturing-space-mcp-bridge-session-context"

// SessionContext carries fixed authority for one internal MCP session.
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

// WithSessionContext stores one fixed MCP bridge context on ctx.
func WithSessionContext(ctx context.Context, sessionCtx SessionContext) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, sessionContextKey, sessionCtx.Normalize())
}

// SessionContextFromContext loads the fixed bridge context from ctx.
func SessionContextFromContext(ctx context.Context) SessionContext {
	if ctx == nil {
		return SessionContext{}
	}
	sessionCtx, _ := ctx.Value(sessionContextKey).(SessionContext)
	return sessionCtx.Normalize()
}

// SessionContextFromHeaders parses the fixed bridge context from request headers.
func SessionContextFromHeaders(header http.Header) SessionContext {
	if len(header) == 0 {
		return SessionContext{}
	}
	return SessionContext{
		CampaignID:    strings.TrimSpace(header.Get(CampaignIDHeader)),
		SessionID:     strings.TrimSpace(header.Get(SessionIDHeader)),
		ParticipantID: strings.TrimSpace(header.Get(ParticipantIDHeader)),
	}.Normalize()
}

// ApplyToHeader writes the fixed bridge context into header.
func (c SessionContext) ApplyToHeader(header http.Header) {
	if header == nil {
		return
	}
	c = c.Normalize()
	if c.CampaignID != "" {
		header.Set(CampaignIDHeader, c.CampaignID)
	}
	if c.SessionID != "" {
		header.Set(SessionIDHeader, c.SessionID)
	}
	if c.ParticipantID != "" {
		header.Set(ParticipantIDHeader, c.ParticipantID)
	}
}
