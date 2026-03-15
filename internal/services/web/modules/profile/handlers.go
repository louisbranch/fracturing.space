package profile

import (
	"net/http"

	profileapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/routeparam"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// handlers defines an internal contract used at this web package boundary.
type handlers struct {
	publichandler.Base
	assetBaseURL string
	service      profileapp.Service
}

// newHandlers builds package wiring for this web seam.
func newHandlers(s profileapp.Service, assetBaseURL string, base publichandler.Base) handlers {
	return handlers{Base: base, assetBaseURL: assetBaseURL, service: s}
}

// withUsername extracts the username path param and delegates to fn, returning
// the module not-found flow when the param is missing.
func (h handlers) withUsername(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return routeparam.WithRequired("username", h.handleNotFound, fn)
}

// handleProfile handles this route in the module transport layer.
func (h handlers) handleProfile(w http.ResponseWriter, r *http.Request, username string) {
	profile, err := h.service.LoadProfile(httpx.RequestContext(r), username)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.renderProfilePage(w, r, profile)
}

// handleNotFound handles this route in the module transport layer.
func (h handlers) handleNotFound(w http.ResponseWriter, r *http.Request) {
	h.WriteError(w, r, apperrors.E(apperrors.KindNotFound, profileapp.ProfileNotFoundMessage))
}

// renderProfilePage centralizes this web behavior in one helper seam.
func (h handlers) renderProfilePage(w http.ResponseWriter, r *http.Request, profile profileapp.Profile) {
	loc, lang := h.PageLocalizer(w, r)
	h.WritePublicPage(
		w,
		r,
		profile.Username,
		webtemplates.T(loc, "layout.meta_description"),
		lang,
		http.StatusOK,
		PublicProfilePage(mapPublicProfileTemplateView(profile, h.assetBaseURL, h.IsViewerSignedIn(r)), loc),
	)
}
