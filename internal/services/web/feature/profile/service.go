package profile

import (
	"net/http"

	routing "github.com/louisbranch/fracturing.space/internal/services/web/feature/routing"
)

// Handlers configures callback-backed profile service construction.
type Handlers struct {
	Profile http.HandlerFunc
}

type callbackService struct {
	handlers Handlers
}

// NewService builds a profile Service backed by handler callbacks.
func NewService(handlers Handlers) Service {
	return callbackService{handlers: handlers}
}

func (s callbackService) HandleProfile(w http.ResponseWriter, r *http.Request) {
	routing.CallOrNotFound(w, r, s.handlers.Profile)
}
