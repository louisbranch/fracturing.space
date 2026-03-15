package adversarytransport

import "github.com/louisbranch/fracturing.space/internal/platform/id"

// NewHandler builds an adversary transport handler from explicit reads and
// write callbacks.
func NewHandler(deps Dependencies) *Handler {
	if deps.GenerateID == nil {
		deps.GenerateID = id.NewID
	}
	return &Handler{deps: deps}
}
