package web

import (
	"net/http"

	feature_auth "github.com/louisbranch/fracturing.space/internal/services/web/feature/auth"
	feature_discovery "github.com/louisbranch/fracturing.space/internal/services/web/feature/discovery"
	feature_publicprofile "github.com/louisbranch/fracturing.space/internal/services/web/feature/publicprofile"
)

func (h *handler) buildPublicAuthRouteMux(routes feature_auth.PublicHandlers) *http.ServeMux {
	authMux := http.NewServeMux()
	feature_auth.RegisterPublicRoutes(authMux, feature_auth.NewPublicService(routes))
	return authMux
}

func (h *handler) buildPublicProfileRouteMux(routes feature_publicprofile.Handlers) *http.ServeMux {
	profileMux := http.NewServeMux()
	feature_publicprofile.RegisterRoutes(profileMux, feature_publicprofile.NewService(routes))
	return profileMux
}

func (h *handler) buildPublicDiscoveryRouteMux(routes feature_discovery.Handlers) *http.ServeMux {
	discoveryMux := http.NewServeMux()
	feature_discovery.RegisterRoutes(discoveryMux, feature_discovery.NewService(routes))
	return discoveryMux
}
