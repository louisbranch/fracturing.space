package profile

import (
	"net/http"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

type handlers struct {
	publichandler.Base
	service service
}

func newHandlers(s service, base publichandler.Base) handlers {
	return handlers{Base: base, service: s}
}

func (h handlers) handleProfile(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.PathValue("username"))
	profile, err := h.service.loadProfile(httpx.RequestContext(r), username)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.renderProfilePage(w, r, profile)
}

func (h handlers) handleNotFound(w http.ResponseWriter, r *http.Request) {
	h.WriteError(w, r, apperrors.E(apperrors.KindNotFound, profileNotFoundMessage))
}

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
