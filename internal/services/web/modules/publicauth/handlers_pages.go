package publicauth

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/a-h/templ"
	sharedi18n "github.com/louisbranch/fracturing.space/internal/services/shared/i18nhttp"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"golang.org/x/text/language"
)

// handleRoot handles this route in the module transport layer.
func (h handlers) handleRoot(w http.ResponseWriter, r *http.Request) {
	if h.redirectAuthenticatedToApp(w, r) {
		return
	}
	langTag := h.resolveAuthTag(w, r)
	copy := webi18n.Auth(langTag)
	h.writeAuthPage(w, r, copy.LandingTitle, copy.MetaDescription, langTag.String(), webtemplates.PublicRootFragment(copy))
}

// handleLogin handles this route in the module transport layer.
func (h handlers) handleLogin(w http.ResponseWriter, r *http.Request) {
	if h.redirectAuthenticatedToApp(w, r) {
		return
	}
	langTag := h.resolveAuthTag(w, r)
	copy := webi18n.Auth(langTag)
	h.writeAuthPage(w, r, copy.LoginTitle, copy.MetaDescription, langTag.String(), webtemplates.PasskeyLoginPage(copy, h.recoveryPageURL(r), h.pendingID(r)))
}

// handleRecoveryGet handles this route in the module transport layer.
func (h handlers) handleRecoveryGet(w http.ResponseWriter, r *http.Request) {
	if h.redirectAuthenticatedToApp(w, r) {
		return
	}
	langTag := h.resolveAuthTag(w, r)
	copy := webi18n.Auth(langTag)
	h.writeAuthPage(w, r, copy.RecoveryTitle, copy.MetaDescription, langTag.String(), webtemplates.PasskeyRecoveryPage(copy, h.pendingID(r)))
}

// handleAuthLogin handles this route in the module transport layer.
func (h handlers) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if h.redirectAuthenticatedToApp(w, r) {
		return
	}
	httpx.WriteRedirect(w, r, routepath.Login)
}

// handleHealth handles this route in the module transport layer.
func (h handlers) handleHealth(w http.ResponseWriter, r *http.Request) {
	_ = httpx.WriteHTML(w, http.StatusOK, h.service.HealthBody())
}

// writeAuthPage centralizes this web behavior in one helper seam.
func (h handlers) writeAuthPage(w http.ResponseWriter, r *http.Request, title string, metaDesc string, lang string, body templ.Component) {
	h.WritePublicPage(w, r, title, metaDesc, lang, http.StatusOK, body)
}

// writeNotFoundPage centralizes this web behavior in one helper seam.
func (h handlers) writeNotFoundPage(w http.ResponseWriter, r *http.Request) {
	h.WriteNotFound(w, r)
}

// resolveAuthTag resolves request-scoped values needed by this package.
func (h handlers) resolveAuthTag(w http.ResponseWriter, r *http.Request) language.Tag {
	langTag, persist := sharedi18n.ResolveTag(r)
	if persist {
		sharedi18n.SetLanguageCookie(w, langTag)
	}
	return langTag
}

// pendingID extracts an optional pending OAuth authorization ID from the query.
func (h handlers) pendingID(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	return strings.TrimSpace(r.URL.Query().Get("pending_id"))
}

// recoveryPageURL builds the login-to-recovery link, preserving pending auth state.
func (h handlers) recoveryPageURL(r *http.Request) string {
	pendingID := h.pendingID(r)
	if pendingID == "" {
		return routepath.LoginRecovery
	}
	values := url.Values{}
	values.Set("pending_id", pendingID)
	return routepath.LoginRecovery + "?" + values.Encode()
}
