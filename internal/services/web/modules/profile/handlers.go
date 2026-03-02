package profile

import (
	"net/http"
	"strings"

	profileapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// profileService defines an internal contract used at this web package boundary.
type profileService = profileapp.Service

// handlers defines an internal contract used at this web package boundary.
type handlers struct {
	publichandler.Base
	service profileService
}

// newHandlers builds package wiring for this web seam.
func newHandlers(s profileService, base publichandler.Base) handlers {
	return handlers{Base: base, service: s}
}

// handleProfile handles this route in the module transport layer.
func (h handlers) handleProfile(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.PathValue("username"))
	profile, err := h.service.LoadProfile(httpx.RequestContext(r), username)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.renderProfilePage(w, r, profile)
}

// handleNotFound handles this route in the module transport layer.
func (h handlers) handleNotFound(w http.ResponseWriter, r *http.Request) {
	h.WriteError(w, r, apperrors.E(apperrors.KindNotFound, profileNotFoundMessage))
}

// renderProfilePage centralizes this web behavior in one helper seam.
func (h handlers) renderProfilePage(w http.ResponseWriter, r *http.Request, profile Profile) {
	loc, lang := webi18n.ResolveLocalizer(w, r, nil)
	h.WritePublicPage(
		w,
		r,
		profile.Username,
		webtemplates.T(loc, "layout.meta_description"),
		lang,
		http.StatusOK,
		webtemplates.PublicProfilePage(webtemplates.PublicProfileView{
			Username:       profile.Username,
			Name:           profile.Name,
			Pronouns:       profile.Pronouns,
			Bio:            profile.Bio,
			AvatarURL:      profile.AvatarURL,
			ViewerSignedIn: h.IsViewerSignedIn(r),
		}, loc),
	)
}
