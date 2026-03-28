package participants

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// RegisterStableRoutes registers stable participant routes.
func RegisterStableRoutes(mux *http.ServeMux, h Handler) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignParticipantsPattern, h.WithCampaignID(h.HandleParticipants))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignParticipantCreatePattern, h.WithCampaignID(h.HandleParticipantCreatePage))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignParticipantCreatePattern, h.WithCampaignID(h.HandleParticipantCreate))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignParticipantEditPattern, h.WithCampaignAndParticipantID(h.HandleParticipantEdit))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignParticipantEditPattern, h.WithCampaignAndParticipantID(h.HandleParticipantUpdate))
}
