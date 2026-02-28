package web

import (
	"context"
	"net/http"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/authctx"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/sessioncookie"
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

func newSessionResolver(client PrincipalSessionClient) sessionResolver {
	return sessionResolver{authClient: client}
}

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
	userID := strings.TrimSpace(resp.GetSession().GetUserId())
	if userID == "" {
		return "", false
	}
	return userID, true
}

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

func (r sessionResolver) resolveRequestUserID(request *http.Request) string {
	if state := requestPrincipalStateFromRequest(request); state != nil {
		state.userIDOnce.Do(func() {
			state.userID = r.resolveRequestUserIDUncached(request)
		})
		return state.userID
	}
	return r.resolveRequestUserIDUncached(request)
}

func (r sessionResolver) resolveRequestSignedIn(request *http.Request) bool {
	return strings.TrimSpace(r.resolveRequestUserID(request)) != ""
}

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
