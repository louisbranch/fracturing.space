package discovery

import (
	"net/http"

	moduleruntime "github.com/louisbranch/fracturing.space/internal/services/web/module/runtime"
)

// Handlers configures callback-backed discovery service construction.
type Handlers struct {
	Discover         http.HandlerFunc
	DiscoverCampaign http.HandlerFunc
}

type callbackService struct {
	handlers Handlers
}

// NewService builds a discovery Service backed by handler callbacks.
func NewService(handlers Handlers) Service {
	return callbackService{handlers: handlers}
}

func (s callbackService) HandleDiscover(w http.ResponseWriter, r *http.Request) {
	moduleruntime.CallOrNotFound(w, r, s.handlers.Discover)
}

func (s callbackService) HandleDiscoverCampaign(w http.ResponseWriter, r *http.Request) {
	moduleruntime.CallOrNotFound(w, r, s.handlers.DiscoverCampaign)
}
