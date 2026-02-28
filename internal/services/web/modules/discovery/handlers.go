package discovery

import (
	"net/http"

	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

type handlers struct {
	publichandler.Base
}

func newHandlers(base publichandler.Base) handlers {
	return handlers{Base: base}
}

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
