package web

func (h *handler) publicRouteHandlersImpl() publicRouteHandlers {
	return publicRouteHandlers{
		PublicAuthHandlers:    h.publicAuthRouteHandlers(),
		PublicProfileHandlers: h.publicProfileRouteHandlers(),
		DiscoveryHandlers:     h.publicDiscoveryRouteHandlers(),
	}
}
