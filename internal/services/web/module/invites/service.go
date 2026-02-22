package invites

import (
	"net/http"

	moduleruntime "github.com/louisbranch/fracturing.space/internal/services/web/module/runtime"
)

// Handlers configures callback-backed invites service construction.
type Handlers struct {
	Invites     http.HandlerFunc
	InviteClaim http.HandlerFunc
}

type callbackService struct {
	handlers Handlers
}

// NewService builds an invites Service backed by handler callbacks.
func NewService(handlers Handlers) Service {
	return callbackService{handlers: handlers}
}

func (s callbackService) HandleInvites(w http.ResponseWriter, r *http.Request) {
	moduleruntime.CallOrNotFound(w, r, s.handlers.Invites)
}

func (s callbackService) HandleInviteClaim(w http.ResponseWriter, r *http.Request) {
	moduleruntime.CallOrNotFound(w, r, s.handlers.InviteClaim)
}
