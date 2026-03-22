package module

import "net/http"

// Viewer contains user-facing chrome data for authenticated app pages.
type Viewer struct {
	DisplayName            string
	AvatarURL              string
	ProfileURL             string
	NotificationsAvailable bool
	HasUnreadNotifications bool
}

// Mount describes a module route mount.
type Mount struct {
	Prefix string
	// CanonicalRoot controls trailing-slash canonicalization. When true,
	// requests to the bare prefix (e.g. /app/campaigns/) are redirected to
	// the slashless canonical form (e.g. /app/campaigns). The slashless
	// path is also claimed as a route so both forms resolve to this module.
	CanonicalRoot bool
	Handler       http.Handler
}

// Module declares the minimum contract required by web composition.
type Module interface {
	ID() string
	Mount() (Mount, error)
}
