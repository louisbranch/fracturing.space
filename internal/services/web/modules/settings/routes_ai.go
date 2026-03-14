package settings

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// registerAIRoutes wires AI key and agent settings routes.
func registerAIRoutes(mux *http.ServeMux, h handlers) {
	mux.HandleFunc(http.MethodGet+" "+routepath.AppSettingsAIKeys, h.handleAIKeysGet)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppSettingsAIKeys, h.handleAIKeysCreate)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppSettingsAIKeyRevokePattern, httpx.MethodNotAllowed(http.MethodPost))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppSettingsAIKeyRevokePattern, h.withCredentialID(h.handleAIKeyRevoke))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppSettingsAIAgents, h.handleAIAgentsGet)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppSettingsAIAgents, h.handleAIAgentsCreate)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppSettingsAIAgentDeletePattern, httpx.MethodNotAllowed(http.MethodPost))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppSettingsAIAgentDeletePattern, h.withAgentID(h.handleAIAgentDelete))
}
