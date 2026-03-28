package characters

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// RegisterStableRoutes registers stable character workspace and
// character-creation routes.
func RegisterStableRoutes(mux *http.ServeMux, h Handler) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharactersPattern, h.WithCampaignID(h.HandleCharacters))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterCreatePattern, h.WithCampaignID(h.HandleCharacterCreatePage))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterCreatePattern, h.WithCampaignID(h.HandleCharacterCreate))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterPattern, h.WithCampaignAndCharacterID(h.HandleCharacterDetail))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterEditPattern, h.WithCampaignAndCharacterID(h.HandleCharacterEdit))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterEditPattern, h.WithCampaignAndCharacterID(h.HandleCharacterUpdate))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterControlPattern, h.WithCampaignAndCharacterID(h.HandleCharacterControlSet))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterControlClaimPattern, h.WithCampaignAndCharacterID(h.HandleCharacterControlClaim))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterControlReleasePattern, h.WithCampaignAndCharacterID(h.HandleCharacterControlRelease))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterDeletePattern, h.WithCampaignAndCharacterID(h.HandleCharacterDelete))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterCreationPattern, h.WithCampaignAndCharacterID(h.HandleCharacterCreationPage))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterCreationStepPattern, h.WithCampaignAndCharacterID(h.HandleCharacterCreationStep))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterCreationResetPattern, h.WithCampaignAndCharacterID(h.HandleCharacterCreationReset))

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignRestPattern, h.WriteNotFound)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignRestPattern, h.WriteNotFound)
}
