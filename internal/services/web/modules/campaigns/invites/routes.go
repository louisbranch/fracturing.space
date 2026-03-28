package invites

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// RegisterStableRoutes registers stable invite routes.
func RegisterStableRoutes(mux *http.ServeMux, h Handler) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignInvitesPattern, h.WithCampaignID(h.HandleInvites))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignInviteCreatePattern, h.WithCampaignID(h.HandleInviteCreatePage))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignInviteSearchPattern, h.WithCampaignID(h.HandleInviteSearch))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignInviteCreatePattern, h.WithCampaignID(h.HandleInviteCreate))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignInviteRevokePattern, h.WithCampaignID(h.HandleInviteRevoke))
}
