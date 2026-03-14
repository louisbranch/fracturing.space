package principal

import (
	"context"
	"net/http"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/sessioncookie"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/userid"
)

// ResolveSignedIn reports whether the request carries a valid authenticated
// user.
func (r Resolver) ResolveSignedIn(request *http.Request) bool {
	return userid.Normalize(r.ResolveUserID(request)) != ""
}

// ResolveUserID returns the authenticated user id for the request when one can
// be resolved safely.
func (r Resolver) ResolveUserID(request *http.Request) string {
	if snapshot := snapshotFromRequest(request); snapshot != nil {
		snapshot.userIDOnce.Do(func() {
			snapshot.userID = r.resolveUserIDUncached(request)
		})
		return snapshot.userID
	}
	return r.resolveUserIDUncached(request)
}

// resolveSessionUserID validates the session cookie value and normalizes the
// resulting user id for downstream browser use.
func (r authResolver) resolveSessionUserID(ctx context.Context, sessionID string) (string, bool) {
	if r.sessionClient == nil {
		return "", false
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", false
	}
	resp, err := r.sessionClient.GetWebSession(ctx, &authv1.GetWebSessionRequest{SessionId: sessionID})
	if err != nil || resp == nil || resp.GetSession() == nil {
		return "", false
	}
	userID := userid.Normalize(resp.GetSession().GetUserId())
	if userID == "" {
		return "", false
	}
	return userID, true
}

// resolveUserIDUncached reads and validates the request session cookie.
func (r Resolver) resolveUserIDUncached(request *http.Request) string {
	if request == nil {
		return ""
	}
	sessionID, ok := sessioncookie.Read(request)
	if !ok {
		return ""
	}
	userID, ok := r.auth.resolveSessionUserID(request.Context(), sessionID)
	if !ok {
		return ""
	}
	return userID
}
