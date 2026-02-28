package profile

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/weberror"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

type handlers struct {
	service service
	deps    runtimeDependencies
}

type runtimeDependencies struct {
	resolveViewer module.ResolveViewer
}

func newRuntimeDependencies(deps module.Dependencies) runtimeDependencies {
	return runtimeDependencies{resolveViewer: deps.ResolveViewer}
}

func (d runtimeDependencies) isViewerSignedIn(r *http.Request) bool {
	if d.resolveViewer == nil {
		return false
	}
	viewer := d.resolveViewer(r)
	return strings.TrimSpace(viewer.DisplayName) != ""
}

func newHandlers(s service, deps module.Dependencies) handlers {
	return handlers{service: s, deps: newRuntimeDependencies(deps)}
}

func (h handlers) handleProfile(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.PathValue("username"))
	profile, err := h.service.loadProfile(requestContext(r), username)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.renderProfilePage(w, r, profile)
}

func (h handlers) handleNotFound(w http.ResponseWriter, r *http.Request) {
	h.writeError(w, r, apperrors.E(apperrors.KindNotFound, profileNotFoundMessage))
}

func (h handlers) renderProfilePage(w http.ResponseWriter, r *http.Request, profile Profile) {
	loc, lang := webi18n.ResolveLocalizer(w, r, nil)
	h.writeAuthPageWithStatus(
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
			ViewerSignedIn: h.deps.isViewerSignedIn(r),
		}, loc),
	)
}

func (h handlers) writeError(w http.ResponseWriter, r *http.Request, err error) {
	if w == nil {
		return
	}
	statusCode := apperrors.HTTPStatus(err)
	if weberror.ShouldRenderAppError(statusCode) {
		h.writeAuthErrorPage(w, r, statusCode)
		return
	}
	loc, _ := webi18n.ResolveLocalizer(w, r, nil)
	http.Error(w, weberror.PublicMessage(loc, err), statusCode)
}

func (h handlers) writeAuthErrorPage(w http.ResponseWriter, r *http.Request, statusCode int) {
	loc, lang := webi18n.ResolveLocalizer(w, r, nil)
	h.writeAuthPageWithStatus(
		w,
		r,
		webtemplates.AppErrorPageTitle(statusCode, loc),
		webtemplates.T(loc, "layout.meta_description"),
		lang,
		statusCode,
		webtemplates.AppErrorState(statusCode, loc),
	)
}

func (handlers) writeAuthPageWithStatus(w http.ResponseWriter, r *http.Request, title string, metaDesc string, lang string, statusCode int, body templ.Component) {
	if w == nil {
		return
	}
	if statusCode <= 0 {
		statusCode = http.StatusOK
	}
	if body == nil {
		body = templ.ComponentFunc(func(httpContext context.Context, writer io.Writer) error {
			return nil
		})
	}

	ctx := templ.WithChildren(requestContext(r), body)
	var rendered bytes.Buffer
	if err := webtemplates.AuthLayout(title, metaDesc, lang, requestPath(r), requestQuery(r)).Render(ctx, &rendered); err != nil {
		http.Error(w, weberror.PublicMessage(nil, err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	_, _ = w.Write(rendered.Bytes())
}

func requestContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}

func requestPath(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	return r.URL.Path
}

func requestQuery(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	return r.URL.RawQuery
}
