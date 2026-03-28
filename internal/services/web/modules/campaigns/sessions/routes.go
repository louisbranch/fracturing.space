package sessions

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// RegisterStableRoutes registers stable session and game routes.
func RegisterStableRoutes(mux *http.ServeMux, h Handler) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignSessionsPattern, h.WithCampaignID(h.HandleSessions))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignSessionCreatePattern, h.WithCampaignID(h.HandleSessionCreatePage))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignSessionCreatePattern, h.WithCampaignID(h.HandleSessionCreate))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignSessionPattern, h.WithCampaignAndSessionID(h.HandleSessionDetail))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignSessionEndPattern, h.WithCampaignID(h.HandleSessionEnd))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignGamePattern, h.WithCampaignID(h.HandleGame))
}
