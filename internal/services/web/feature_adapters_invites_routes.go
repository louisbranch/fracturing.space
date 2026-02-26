package web

import (
	"net/http"

	featureinvites "github.com/louisbranch/fracturing.space/internal/services/web/feature/invites"
)

func (h *handler) appInvitesRouteHandlers() featureinvites.Handlers {
	return featureinvites.Handlers{
		Invites: func(w http.ResponseWriter, r *http.Request) {
			if h == nil {
				http.NotFound(w, r)
				return
			}
			featureinvites.HandleAppInvites(h.appInvitesRouteDependencies(w, r), w, r)
		},
		InviteClaim: func(w http.ResponseWriter, r *http.Request) {
			if h == nil {
				http.NotFound(w, r)
				return
			}
			featureinvites.HandleAppInviteClaim(h.appInviteClaimRouteDependencies(w, r), w, r)
		},
	}
}
