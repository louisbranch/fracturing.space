package publicauth

import (
	"net/http"

	publicauthapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

// handlers defines an internal contract used at this web package boundary.
type handlers struct {
	publichandler.Base
	service     publicauthapp.Service
	requestMeta requestmeta.SchemePolicy
}

// newHandlers builds package wiring for this web seam.
func newHandlers(s publicauthapp.Service, policy requestmeta.SchemePolicy) handlers {
	return handlers{service: s, requestMeta: policy}
}

// handleNotFound handles this route in the module transport layer.
func (h handlers) handleNotFound(w http.ResponseWriter, r *http.Request) {
	h.writeNotFoundPage(w, r)
}
