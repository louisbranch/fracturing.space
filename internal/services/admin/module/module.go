// Package module defines the feature contract used by admin composition.
package module

import "net/http"

// Mount describes one module route mount.
type Mount struct {
	Prefix  string
	Handler http.Handler
}

// Module is the minimum contract required by admin app composition.
type Module interface {
	ID() string
	Mount() (Mount, error)
}
