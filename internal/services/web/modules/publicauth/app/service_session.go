package app

import (
	"context"
	"strings"
)

// sessionService owns session-backed redirect and logout behavior.
type sessionService struct {
	session     SessionGateway
	authBaseURL string
}

// NewSessionService wires session-only public auth flows behind input validation.
func NewSessionService(gateway SessionGateway, authBaseURL string) SessionService {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return sessionService{
		session:     gateway,
		authBaseURL: normalizeAuthBaseURL(authBaseURL),
	}
}

// ResolvePostAuthRedirect returns the auth consent URL, validated continuation,
// or dashboard.
func (s sessionService) ResolvePostAuthRedirect(pendingID string, nextPath string) string {
	return resolvePostAuthRedirect(s.authBaseURL, pendingID, nextPath)
}

// RevokeWebSession treats blank cookie values as already-cleared sessions.
func (s sessionService) RevokeWebSession(ctx context.Context, sessionID string) error {
	resolvedSessionID := strings.TrimSpace(sessionID)
	if resolvedSessionID == "" {
		return nil
	}
	return s.session.RevokeWebSession(ctx, resolvedSessionID)
}
