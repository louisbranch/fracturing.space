package campaigns

import (
	"net/http"
)

// routeSurface defines one explicit campaigns route-owner surface.
type routeSurface struct {
	id       string
	register func(*http.ServeMux, handlers)
}

// registerStableRoutes wires stable campaigns route surfaces.
func registerStableRoutes(mux *http.ServeMux, h handlers) {
	registerRouteSurfaces(mux, h, stableRouteSurfaces())
}

// registerRouteSurfaces applies ordered route surface registrations to one mux.
func registerRouteSurfaces(mux *http.ServeMux, h handlers, surfaces []routeSurface) {
	if mux == nil {
		return
	}
	for _, surface := range surfaces {
		if surface.register == nil {
			continue
		}
		surface.register(mux, h)
	}
}

// stableRouteSurfaces returns stable route-owner surfaces in mount order.
func stableRouteSurfaces() []routeSurface {
	return []routeSurface{
		stableCampaignOverviewRoutes(),
		stableCampaignStarterRoutes(),
		stableCampaignParticipantRoutes(),
		stableCampaignCharacterRoutes(),
		stableCampaignCharacterCreationRoutes(),
		stableCampaignSessionGameRoutes(),
		stableCampaignInviteRoutes(),
	}
}
