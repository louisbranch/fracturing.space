package settings

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// registerRoutes centralizes this web behavior in one helper seam.
func registerRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppSettings, h.redirectSettingsRoot)
	mux.HandleFunc(http.MethodGet+" "+routepath.SettingsPrefix+"{$}", h.redirectSettingsRoot)

	mux.HandleFunc(http.MethodGet+" "+routepath.AppSettingsProfile, h.handleProfileGet)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppSettingsProfile, h.handleProfilePost)

	mux.HandleFunc(http.MethodGet+" "+routepath.AppSettingsLocale, h.handleLocaleGet)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppSettingsLocale, h.handleLocalePost)

	mux.HandleFunc(http.MethodGet+" "+routepath.AppSettingsAIKeys, h.handleAIKeysGet)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppSettingsAIKeys, h.handleAIKeysCreate)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppSettingsAIKeyRevokePattern, httpx.MethodNotAllowed(http.MethodPost))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppSettingsAIKeyRevokePattern, h.withCredentialID(h.handleAIKeyRevoke))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppSettingsAIAgents, h.handleAIAgentsGet)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppSettingsAIAgents, h.handleAIAgentsCreate)

	mux.HandleFunc(http.MethodGet+" "+routepath.AppSettingsRestPattern, h.WriteNotFound)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppSettingsRestPattern, h.WriteNotFound)

}
