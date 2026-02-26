package support

import (
	"net/http"

	"github.com/a-h/templ"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// ErrorPageRenderer captures dependencies needed to render a localized web error page.
type ErrorPageRenderer struct {
	WriteContentType func(http.ResponseWriter)
	WritePage        func(http.ResponseWriter, *http.Request, templ.Component, string) error
	ComposeTitle     func(webtemplates.PageContext, string, ...any) string
	LocalizeText     func(webtemplates.Localizer, string, map[string]string) string
	LocalizeHTTP     func(http.ResponseWriter, *http.Request, int, string, ...any)
}

// RenderErrorPage renders a shared localized error page using caller-provided seams.
//
// The renderer intentionally stays dependency-inverted so calling web handlers can
// keep behavior-neutral orchestration and unit-test the context assembly separately.
func RenderErrorPage(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	title string,
	message string,
	page webtemplates.PageContext,
	renderer ErrorPageRenderer,
) {
	if renderer.WritePage == nil {
		LocalizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
		return
	}

	localizedTitle := title
	localizedMessage := message
	if renderer.LocalizeText != nil {
		localizedTitle = renderer.LocalizeText(page.Loc, title, ErrorPageTitleTextKeys)
		localizedMessage = renderer.LocalizeText(page.Loc, message, ErrorPageMessageTextKeys)
	}

	if renderer.WriteContentType != nil {
		renderer.WriteContentType(w)
	}
	w.WriteHeader(status)
	if err := renderer.WritePage(
		w,
		r,
		webtemplates.ErrorPage(page, localizedTitle, localizedMessage),
		composeErrorTitle(page, localizedTitle, renderer.ComposeTitle),
	); err != nil {
		if renderer.LocalizeHTTP != nil {
			renderer.LocalizeHTTP(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
		}
	}
}

func composeErrorTitle(page webtemplates.PageContext, title string, composeTitle func(webtemplates.PageContext, string, ...any) string) string {
	if composeTitle != nil {
		return composeTitle(page, title)
	}
	return title
}
