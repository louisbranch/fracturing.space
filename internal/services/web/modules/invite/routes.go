package invite

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// registerRoutes owns the public invite URL contract for landing and actions.
func registerRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.PublicInvitePattern, h.withInviteID(h.handleInvite))
	mux.HandleFunc(http.MethodPost+" "+routepath.PublicInviteAcceptPattern, h.withInviteID(h.handleAccept))
	mux.HandleFunc(http.MethodGet+" "+routepath.PublicInviteAcceptPattern, httpx.MethodNotAllowed(http.MethodPost))
	mux.HandleFunc(http.MethodPost+" "+routepath.PublicInviteDeclinePattern, h.withInviteID(h.handleDecline))
	mux.HandleFunc(http.MethodGet+" "+routepath.PublicInviteDeclinePattern, httpx.MethodNotAllowed(http.MethodPost))
	mux.HandleFunc(http.MethodGet+" "+routepath.InvitePrefix+"{$}", h.handleNotFound)
	mux.HandleFunc(http.MethodGet+" "+routepath.PublicInviteRestPattern, h.handleNotFound)
	mux.HandleFunc(http.MethodPost+" "+routepath.PublicInviteRestPattern, h.handleNotFound)
}
