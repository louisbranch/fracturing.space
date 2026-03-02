package discovery

import (
	"net/http"

	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// handlers defines an internal contract used at this web package boundary.
type handlers struct {
	publichandler.Base
}

// newHandlers builds package wiring for this web seam.
func newHandlers(base publichandler.Base) handlers {
	return handlers{Base: base}
}

// handleIndex handles this route in the module transport layer.
func (h handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	loc, lang := webi18n.ResolveLocalizer(w, r, nil)
	h.WritePublicPage(
		w,
		r,
		webtemplates.T(loc, "web.discovery.title"),
		webtemplates.T(loc, "layout.meta_description"),
		lang,
		http.StatusOK,
		webtemplates.DiscoveryFragment(loc),
	)
}
