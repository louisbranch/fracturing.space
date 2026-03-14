package publicauth

import (
	"net/http"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
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
func newHandlers(s publicauthapp.Service, policy requestmeta.SchemePolicy, resolveSignedIn ...module.ResolveSignedIn) handlers {
	var signedIn module.ResolveSignedIn
	if len(resolveSignedIn) > 0 {
		signedIn = resolveSignedIn[0]
	}
	return handlers{
		Base:        publichandler.NewBase(publichandler.WithResolveViewerSignedIn(signedIn)),
		service:     s,
		requestMeta: policy,
	}
}

// handleNotFound handles this route in the module transport layer.
func (h handlers) handleNotFound(w http.ResponseWriter, r *http.Request) {
	h.writeNotFoundPage(w, r)
}
