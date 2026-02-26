package publicprofile

import (
	"net/http"

	routing "github.com/louisbranch/fracturing.space/internal/services/web/feature/routing"
)

// Handlers configures callback-backed public profile service construction.
type Handlers struct {
	PublicProfile http.HandlerFunc
}

type callbackService struct {
	handlers Handlers
}

// NewService builds a public profile Service backed by handler callbacks.
func NewService(handlers Handlers) Service {
	return callbackService{handlers: handlers}
}

func (s callbackService) HandlePublicProfile(w http.ResponseWriter, r *http.Request) {
	routing.CallOrNotFound(w, r, s.handlers.PublicProfile)
}
