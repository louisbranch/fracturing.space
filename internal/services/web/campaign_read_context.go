package web

import (
	"context"
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func (h *handler) campaignReadContext(w http.ResponseWriter, r *http.Request, unavailableTitle string) (context.Context, string, bool) {
	sess := sessionFromRequest(r, h.sessions)
	if sess == nil {
		http.Redirect(w, r, routepath.AuthLogin, http.StatusFound)
		return nil, "", false
	}

	userID, err := h.sessionUserIDForSession(r.Context(), sess)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, unavailableTitle, "failed to resolve current user")
		return nil, "", false
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		h.renderErrorPage(w, r, http.StatusUnauthorized, "Authentication required", "no user identity was resolved for this session")
		return nil, "", false
	}

	return grpcauthctx.WithUserID(r.Context(), userID), userID, true
}
