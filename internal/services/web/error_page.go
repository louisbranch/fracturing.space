package web

import (
	"net/http"

	websupport "github.com/louisbranch/fracturing.space/internal/services/web/support"
)

// renderErrorPage converts internal transport and auth errors to a localized web
// error template, so failure states stay in one shared UX surface.
func (h *handler) renderErrorPage(w http.ResponseWriter, r *http.Request, status int, title string, message string) {
	websupport.RenderErrorPage(w, r, status, title, message, h.pageContext(w, r), websupport.ErrorPageRenderer{
		WriteContentType: writeGameContentType,
		WritePage:        h.writePage,
		ComposeTitle:     composeHTMXTitleForPage,
		LocalizeText:     websupport.LocalizeErrorText,
		LocalizeHTTP:     localizeHTTPError,
	})
}

func localizeHTTPError(w http.ResponseWriter, r *http.Request, status int, key string, args ...any) {
	websupport.LocalizeHTTPError(w, r, status, key, args...)
}
