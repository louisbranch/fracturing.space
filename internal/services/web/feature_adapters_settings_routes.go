package web

import (
	"net/http"

	featuresettings "github.com/louisbranch/fracturing.space/internal/services/web/feature/settings"
)

func (h *handler) appSettingsRouteHandlers() featuresettings.Handlers {
	return featuresettings.Handlers{
		Settings: func(w http.ResponseWriter, r *http.Request) {
			if h == nil {
				http.NotFound(w, r)
				return
			}
			featuresettings.HandleAppSettings(h.appSettingsRouteDependencies(w, r), w, r)
		},
		UserProfileSettings: func(w http.ResponseWriter, r *http.Request) {
			if h == nil {
				http.NotFound(w, r)
				return
			}
			featuresettings.HandleAppUserProfileSettings(h.appSettingsRouteDependencies(w, r), w, r)
		},
		AIKeys: func(w http.ResponseWriter, r *http.Request) {
			if h == nil {
				http.NotFound(w, r)
				return
			}
			featuresettings.HandleAppAIKeys(h.appSettingsRouteDependencies(w, r), w, r)
		},
		AIKeyRevoke: func(w http.ResponseWriter, r *http.Request, credentialID string) {
			if h == nil {
				http.NotFound(w, r)
				return
			}
			featuresettings.HandleAppAIKeyRevoke(h.appSettingsRouteDependencies(w, r), w, r, credentialID)
		},
	}
}
