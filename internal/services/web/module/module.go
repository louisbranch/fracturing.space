// Package module defines the feature contract used by web composition.
package module

import "net/http"

// Viewer contains user-facing chrome data for authenticated app pages.
type Viewer struct {
	DisplayName            string
	AvatarURL              string
	ProfileURL             string
	HasUnreadNotifications bool
}

// ResolveViewer resolves app chrome viewer state for a request.
type ResolveViewer func(*http.Request) Viewer

// ResolveSignedIn reports whether the request is associated with a signed-in actor.
type ResolveSignedIn func(*http.Request) bool

// ResolveUserID resolves the authenticated user id for a request.
type ResolveUserID func(*http.Request) string

// ResolveLanguage returns the effective request language.
type ResolveLanguage func(*http.Request) string

// Mount describes a module route mount.
type Mount struct {
	Prefix  string
	Handler http.Handler
}

// Module declares the minimum contract required by web composition.
type Module interface {
	ID() string
	Mount() (Mount, error)
}

// HealthReporter is an optional interface for modules that can report their
// operational availability. Modules with gateway dependencies implement this
// so the registry can derive service health without centralizing client knowledge.
type HealthReporter interface {
	Healthy() bool
}
