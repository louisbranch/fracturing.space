package web

import (
	"log"
	"net/http"
	"strings"

	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

func (h *handler) handleAppLoginPage(w http.ResponseWriter, r *http.Request, appName string) {
	if r.Method != http.MethodGet {
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	printer, lang := localizer(w, r)

	pendingID := strings.TrimSpace(r.URL.Query().Get("pending_id"))
	if pendingID == "" {
		if strings.TrimSpace(h.config.OAuthClientID) != "" {
			http.Redirect(w, r, routepath.AuthLogin, http.StatusFound)
			return
		}
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.pending_id_is_required")
		return
	}
	clientID := strings.TrimSpace(r.URL.Query().Get("client_id"))
	clientName := strings.TrimSpace(r.URL.Query().Get("client_name"))
	errorMessage := strings.TrimSpace(r.URL.Query().Get("error"))
	if clientName == "" {
		if clientID != "" {
			clientName = clientID
		} else {
			clientName = webtemplates.T(printer, "web.login.unknown_client")
		}
	}

	params := webtemplates.LoginParams{
		AppName:      appName,
		PendingID:    pendingID,
		ClientName:   clientName,
		Error:        errorMessage,
		Lang:         lang,
		Loc:          printer,
		CurrentPath:  r.URL.Path,
		CurrentQuery: r.URL.RawQuery,
	}
	loginPage := webtemplates.PageContext{
		Lang:    lang,
		Loc:     printer,
		AppName: appName,
	}
	if err := h.writePage(w, r, webtemplates.LoginPage(params), composeHTMXTitleForPage(loginPage, "title.login")); err != nil {
		log.Printf("web: failed to render login page: %v", err)
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
		return
	}
}
