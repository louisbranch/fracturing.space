package app

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
)

// registerRoutes wires the browser-facing HTTP surface without leaking route
// assembly back into the server lifecycle file.
func (s *Server) registerRoutes(rootMux *http.ServeMux, launchGrantCfg playlaunchgrant.Config) {
	for _, route := range s.playRoutes(launchGrantCfg) {
		route.register(rootMux)
	}
}

// playRoutes is the single browser-surface index for the play service. The
// interaction mutation subset is delegated to interactionRoutes so both the
// broad route map and the focused mutation inventory stay reader-visible.
func (s *Server) playRoutes(launchGrantCfg playlaunchgrant.Config) []interactionRoute {
	routes := []interactionRoute{
		{
			pattern: "GET /up",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("OK"))
			}),
		},
		{
			pattern: "GET /{$}",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				s.handleRootShell(w, r)
			}),
		},
		{
			pattern: "GET /campaigns/{campaignID}",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				s.handleCampaignShell(w, r, launchGrantCfg)
			}),
		},
		{
			pattern: "GET /api/campaigns/{campaignID}/bootstrap",
			handler: http.HandlerFunc(s.handleBootstrap),
		},
		{
			pattern: "GET /api/campaigns/{campaignID}/chat/history",
			handler: http.HandlerFunc(s.handleChatHistory),
		},
		{
			pattern: "GET /realtime",
			handler: s.realtime.handler(),
		},
	}
	return append(routes, interactionRoutes(s)...)
}
