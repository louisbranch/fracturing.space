package profile

import (
	"net/http"

	moduleruntime "github.com/louisbranch/fracturing.space/internal/services/web/module/runtime"
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
	moduleruntime.CallOrNotFound(w, r, s.handlers.Profile)
}
