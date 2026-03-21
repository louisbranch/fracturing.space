package recoverytransport

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
)

// Handler owns Daggerheart recovery and life-state mutation transport.
type Handler struct {
	deps Dependencies
}

// NewHandler builds a recovery transport handler from explicit reads and
// callback seams.
func NewHandler(deps Dependencies) *Handler {
	if deps.ResolveSeed == nil {
		deps.ResolveSeed = random.ResolveSeed
	}
	return &Handler{deps: deps}
}
