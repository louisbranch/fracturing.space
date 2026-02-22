package settings

import (
	"net/http"
	"strings"

	sharedroute "github.com/louisbranch/fracturing.space/internal/services/shared/route"
	moduleruntime "github.com/louisbranch/fracturing.space/internal/services/web/module/runtime"
	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Service is the settings surface contract consumed by settings route registration.
type Service interface {
	HandleSettings(w http.ResponseWriter, r *http.Request)
	HandleUserProfileSettings(w http.ResponseWriter, r *http.Request)
	HandleAIKeys(w http.ResponseWriter, r *http.Request)
	HandleAIKeyRevoke(w http.ResponseWriter, r *http.Request, credentialID string)
}

// RegisterRoutes wires settings routes into the app mux.
func RegisterRoutes(mux *http.ServeMux, service Service) {
	if mux == nil || service == nil {
		return
	}
	mux.HandleFunc(routepath.AppSettings, service.HandleSettings)
	mux.HandleFunc(routepath.AppSettingsPrefix, func(w http.ResponseWriter, r *http.Request) {
		HandleSettingsSubpath(w, r, service)
	})
}

// HandleSettingsSubpath parses settings subpaths and dispatches to settings handlers.
func HandleSettingsSubpath(w http.ResponseWriter, r *http.Request, service Service) {
	if service == nil {
		http.NotFound(w, r)
		return
	}
	if sharedroute.RedirectTrailingSlash(w, r) {
		return
	}

	path := moduleruntime.TrimSubpath(r.URL.Path, routepath.AppSettingsPrefix)
	parts := moduleruntime.SplitParts(path)
	if len(parts) == 1 && parts[0] == "user-profile" {
		service.HandleUserProfileSettings(w, r)
		return
	}
	if len(parts) == 1 && parts[0] == "ai-keys" {
		service.HandleAIKeys(w, r)
		return
	}
	if len(parts) == 3 && parts[0] == "ai-keys" && parts[2] == "revoke" {
		credentialID := strings.TrimSpace(parts[1])
		if credentialID == "" {
			http.NotFound(w, r)
			return
		}
		service.HandleAIKeyRevoke(w, r, credentialID)
		return
	}
	http.NotFound(w, r)
}
