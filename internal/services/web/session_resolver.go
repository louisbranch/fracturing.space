package web

import (
	"context"
	"net/http"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/authctx"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/sessioncookie"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/userid"
	"google.golang.org/grpc"
)

// PrincipalSessionClient is the narrow auth surface needed by session validation.
type PrincipalSessionClient interface {
	GetWebSession(context.Context, *authv1.GetWebSessionRequest, ...grpc.CallOption) (*authv1.GetWebSessionResponse, error)
}

// sessionResolver validates session cookies and resolves user identity.
type sessionResolver struct {
	authClient PrincipalSessionClient
}

// newSessionResolver builds package wiring for this web seam.
func newSessionResolver(client PrincipalSessionClient) sessionResolver {
	return sessionResolver{authClient: client}
}

// resolveSessionUserID resolves request-scoped values needed by this package.
func (r sessionResolver) resolveSessionUserID(ctx context.Context, sessionID string) (string, bool) {
	if r.authClient == nil {
		return "", false
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", false
	}
	resp, err := r.authClient.GetWebSession(ctx, &authv1.GetWebSessionRequest{SessionId: sessionID})
	if err != nil || resp == nil || resp.GetSession() == nil {
		return "", false
	}
	userID := userid.Normalize(resp.GetSession().GetUserId())
	if userID == "" {
		return "", false
	}
	return userID, true
}

// resolveRequestUserIDUncached resolves request-scoped values needed by this package.
func (r sessionResolver) resolveRequestUserIDUncached(req *http.Request) string {
	if req == nil {
		return ""
	}
	sessionID, ok := sessioncookie.Read(req)
	if !ok {
		return ""
	}
	userID, ok := r.resolveSessionUserID(req.Context(), sessionID)
	if !ok {
		return ""
	}
	return userID
}

// resolveRequestUserID resolves request-scoped values needed by this package.
func (r sessionResolver) resolveRequestUserID(request *http.Request) string {
	if state := requestPrincipalStateFromRequest(request); state != nil {
		state.userIDOnce.Do(func() {
			state.userID = r.resolveRequestUserIDUncached(request)
		})
		return state.userID
	}
	return r.resolveRequestUserIDUncached(request)
}

// resolveRequestSignedIn resolves request-scoped values needed by this package.
func (r sessionResolver) resolveRequestSignedIn(request *http.Request) bool {
	return userid.Normalize(r.resolveRequestUserID(request)) != ""
}

// authRequired centralizes this web behavior in one helper seam.
func (r sessionResolver) authRequired() func(*http.Request) bool {
	validated := authctx.ValidatedSessionAuth(func(ctx context.Context, sessionID string) bool {
		userID, ok := r.resolveSessionUserID(ctx, sessionID)
		if !ok {
			return false
		}
		if state := requestPrincipalStateFromContext(ctx); state != nil {
			state.userIDOnce.Do(func() {
				state.userID = userID
			})
		}
		return true
	})
	return func(request *http.Request) bool {
		return validated(request)
	}
}
