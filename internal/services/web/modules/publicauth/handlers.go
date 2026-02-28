package publicauth

import (
	"encoding/json"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/a-h/templ"
	sharedi18n "github.com/louisbranch/fracturing.space/internal/services/shared/i18nhttp"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/sessioncookie"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/weberror"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"golang.org/x/text/language"
)

type handlers struct {
	publichandler.Base
	service     service
	requestMeta requestmeta.SchemePolicy
}

func newHandlers(s service, policy requestmeta.SchemePolicy) handlers {
	return handlers{service: s, requestMeta: policy}
}

func (h handlers) handleRoot(w http.ResponseWriter, r *http.Request) {
	if h.redirectAuthenticatedToApp(w, r) {
		return
	}
	langTag := h.resolveAuthTag(w, r)
	copy := webi18n.Auth(langTag)
	h.writeAuthPage(w, r, copy.LandingTitle, copy.MetaDescription, langTag.String(), webtemplates.PublicRootFragment(copy))
}

func (h handlers) handleLogin(w http.ResponseWriter, r *http.Request) {
	if h.redirectAuthenticatedToApp(w, r) {
		return
	}
	langTag := h.resolveAuthTag(w, r)
	copy := webi18n.Auth(langTag)
	h.writeAuthPage(w, r, copy.LoginTitle, copy.MetaDescription, langTag.String(), webtemplates.PasskeyLoginPage(copy))
}

func (h handlers) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if h.redirectAuthenticatedToApp(w, r) {
		return
	}
	httpx.WriteRedirect(w, r, routepath.Login)
}

func (h handlers) handleLogout(w http.ResponseWriter, r *http.Request) {
	sessionID, hasSession := sessioncookie.Read(r)
	if hasSession && !h.hasSameOriginProof(r) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	h.clearSessionCookie(w, r)
	if hasSession {
		_ = h.service.revokeWebSession(r.Context(), sessionID)
	}
	httpx.WriteRedirect(w, r, routepath.Root)
}

func (h handlers) handlePasskeyLoginStart(w http.ResponseWriter, r *http.Request) {
	start, err := h.service.passkeyLoginStart(r.Context())
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	_ = httpx.WriteJSON(w, http.StatusOK, map[string]any{"session_id": start.SessionID, "public_key": start.PublicKey})
}

func (h handlers) handlePasskeyLoginFinish(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		SessionID  string          `json:"session_id"`
		Credential json.RawMessage `json:"credential"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeJSONError(w, r, apperrors.E(apperrors.KindInvalidInput, "invalid json body"))
		return
	}
	finished, err := h.service.passkeyLoginFinish(r.Context(), payload.SessionID, payload.Credential)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	h.writeSessionCookie(w, r, finished.SessionID)
	_ = httpx.WriteJSON(w, http.StatusOK, map[string]any{"redirect_url": routepath.AppDashboard})
}

func (h handlers) handlePasskeyRegisterStart(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeJSONError(w, r, apperrors.E(apperrors.KindInvalidInput, "invalid json body"))
		return
	}
	start, err := h.service.passkeyRegisterStart(r.Context(), payload.Email)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	_ = httpx.WriteJSON(w, http.StatusOK, map[string]any{"session_id": start.SessionID, "public_key": start.PublicKey, "user_id": start.UserID})
}

func (h handlers) handlePasskeyRegisterFinish(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		SessionID  string          `json:"session_id"`
		Credential json.RawMessage `json:"credential"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeJSONError(w, r, apperrors.E(apperrors.KindInvalidInput, "invalid json body"))
		return
	}
	finished, err := h.service.passkeyRegisterFinish(r.Context(), payload.SessionID, payload.Credential)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	_ = httpx.WriteJSON(w, http.StatusOK, map[string]any{"user_id": finished.UserID})
}

func (h handlers) handleHealth(w http.ResponseWriter, r *http.Request) {
	_ = httpx.WriteHTML(w, http.StatusOK, h.service.healthBody())
}

func (h handlers) handleNotFound(w http.ResponseWriter, r *http.Request) {
	h.writeNotFoundPage(w, r)
}

func (h handlers) writeAuthPage(w http.ResponseWriter, r *http.Request, title string, metaDesc string, lang string, body templ.Component) {
	h.WritePublicPage(w, r, title, metaDesc, lang, http.StatusOK, body)
}

func (h handlers) writeNotFoundPage(w http.ResponseWriter, r *http.Request) {
	h.WriteNotFound(w, r)
}

func (h handlers) resolveAuthTag(w http.ResponseWriter, r *http.Request) language.Tag {
	langTag, persist := sharedi18n.ResolveTag(r)
	if persist {
		sharedi18n.SetLanguageCookie(w, langTag)
	}
	return langTag
}

func (h handlers) writeJSONError(w http.ResponseWriter, r *http.Request, err error) {
	loc, _ := webi18n.ResolveLocalizer(w, r, nil)
	_ = httpx.WriteJSONError(w, apperrors.HTTPStatus(err), weberror.PublicMessage(loc, err))
}

func (h handlers) redirectAuthenticatedToApp(w http.ResponseWriter, r *http.Request) bool {
	if r == nil {
		return false
	}
	sessionID, ok := sessioncookie.Read(r)
	if !ok {
		return false
	}
	if !h.service.hasValidWebSession(r.Context(), sessionID) {
		return false
	}
	httpx.WriteRedirect(w, r, resolveAppRedirectPath(r.URL.Query().Get("next")))
	return true
}

func (h handlers) writeSessionCookie(w http.ResponseWriter, r *http.Request, sessionID string) {
	sessioncookie.WriteWithPolicy(w, r, sessionID, h.requestMeta)
}

func (h handlers) clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	sessioncookie.ClearWithPolicy(w, r, h.requestMeta)
}

func (h handlers) hasSameOriginProof(r *http.Request) bool {
	return requestmeta.HasSameOriginProofWithPolicy(r, h.requestMeta)
}

func resolveAppRedirectPath(raw string) string {
	next := strings.TrimSpace(raw)
	if next == "" {
		return routepath.AppDashboard
	}
	parsed, err := url.Parse(next)
	if err != nil || parsed.Scheme != "" || parsed.Host != "" || parsed.Opaque != "" {
		return routepath.AppDashboard
	}
	rawPath := strings.TrimSpace(parsed.EscapedPath())
	if hasEncodedSlash(rawPath) {
		return routepath.AppDashboard
	}
	decodedPath, err := url.PathUnescape(strings.TrimSpace(parsed.Path))
	if err != nil {
		return routepath.AppDashboard
	}
	if hasDotSegment(decodedPath) {
		return routepath.AppDashboard
	}
	canonicalPath := path.Clean(decodedPath)
	if strings.TrimSpace(canonicalPath) == "." {
		canonicalPath = "/"
	}
	canonicalPath = ensureLeadingSlash(canonicalPath)
	if !strings.HasPrefix(canonicalPath, routepath.AppPrefix) {
		return routepath.AppDashboard
	}
	if canonicalPath == routepath.AppPrefix {
		return routepath.AppDashboard
	}
	if parsed.RawQuery != "" {
		return canonicalPath + "?" + parsed.RawQuery
	}
	return canonicalPath
}

func hasDotSegment(rawPath string) bool {
	for _, part := range strings.Split(rawPath, "/") {
		if part == "." || part == ".." {
			return true
		}
	}
	return false
}

func hasEncodedSlash(rawPath string) bool {
	lower := strings.ToLower(rawPath)
	return strings.Contains(lower, "%2f") || strings.Contains(lower, "%5c")
}

func ensureLeadingSlash(pathValue string) string {
	pathValue = strings.TrimSpace(pathValue)
	if pathValue == "" {
		return "/"
	}
	if strings.HasPrefix(pathValue, "/") {
		return pathValue
	}
	return "/" + pathValue
}
