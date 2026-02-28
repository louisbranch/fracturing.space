package public

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/a-h/templ"
	sharedi18n "github.com/louisbranch/fracturing.space/internal/services/shared/i18nhttp"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/sessioncookie"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/weberror"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"golang.org/x/text/language"
)

type handlers struct {
	service service
}

func newHandlers(s service) handlers {
	return handlers{service: s}
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
	http.Redirect(w, r, routepath.Login, http.StatusFound)
}

func (h handlers) handleLogout(w http.ResponseWriter, r *http.Request) {
	sessionID, hasSession := sessioncookie.Read(r)
	if hasSession && !requestmeta.HasSameOriginProof(r) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	h.clearSessionCookie(w, r)
	if hasSession {
		_ = h.service.revokeWebSession(r.Context(), sessionID)
	}
	http.Redirect(w, r, routepath.Root, http.StatusFound)
}

func (h handlers) handlePasskeyLoginStart(w http.ResponseWriter, r *http.Request) {
	start, err := h.service.passkeyLoginStart(r.Context())
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"session_id": start.sessionID, "public_key": start.publicKey})
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
	h.writeSessionCookie(w, r, finished.sessionID)
	h.writeJSON(w, http.StatusOK, map[string]any{"redirect_url": routepath.AppDashboard})
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
	h.writeJSON(w, http.StatusOK, map[string]any{"session_id": start.sessionID, "public_key": start.publicKey, "user_id": start.userID})
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
	h.writeJSON(w, http.StatusOK, map[string]any{"user_id": finished.userID})
}

func (h handlers) handleHealth(w http.ResponseWriter, r *http.Request) {
	_ = httpx.WriteHTML(w, http.StatusOK, h.service.healthBody())
}

func (h handlers) handleNotFound(w http.ResponseWriter, r *http.Request) {
	h.writeNotFoundPage(w, r)
}

func (h handlers) writeAuthPage(w http.ResponseWriter, r *http.Request, title string, metaDesc string, lang string, body templ.Component) {
	h.writeAuthPageWithStatus(w, r, title, metaDesc, lang, http.StatusOK, body)
}

func (h handlers) writeAuthPageWithStatus(w http.ResponseWriter, r *http.Request, title string, metaDesc string, lang string, statusCode int, body templ.Component) {
	ctx := templ.WithChildren(r.Context(), body)
	var rendered bytes.Buffer
	if err := webtemplates.AuthLayout(title, metaDesc, lang, r.URL.Path, r.URL.RawQuery).Render(ctx, &rendered); err != nil {
		http.Error(w, weberror.PublicMessage(nil, err), http.StatusInternalServerError)
		return
	}
	if statusCode <= 0 {
		statusCode = http.StatusOK
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	_, _ = w.Write(rendered.Bytes())
}

func (h handlers) writeNotFoundPage(w http.ResponseWriter, r *http.Request) {
	loc, lang := webi18n.ResolveLocalizer(w, r, nil)
	h.writeAuthPageWithStatus(
		w,
		r,
		webtemplates.AppErrorPageTitle(http.StatusNotFound, loc),
		webtemplates.T(loc, "layout.meta_description"),
		lang,
		http.StatusNotFound,
		webtemplates.AppErrorState(http.StatusNotFound, loc),
	)
}

func (h handlers) resolveAuthTag(w http.ResponseWriter, r *http.Request) language.Tag {
	langTag, persist := sharedi18n.ResolveTag(r)
	if persist {
		sharedi18n.SetLanguageCookie(w, langTag)
	}
	return langTag
}

func (handlers) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h handlers) writeJSONError(w http.ResponseWriter, r *http.Request, err error) {
	loc, _ := webi18n.ResolveLocalizer(w, r, nil)
	h.writeJSON(w, apperrors.HTTPStatus(err), map[string]any{"error": weberror.PublicMessage(loc, err)})
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
	http.Redirect(w, r, resolveAppRedirectPath(r.URL.Query().Get("next")), http.StatusFound)
	return true
}

func (handlers) writeSessionCookie(w http.ResponseWriter, r *http.Request, sessionID string) {
	sessioncookie.Write(w, r, sessionID)
}

func (handlers) clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	sessioncookie.Clear(w, r)
}

func resolveAppRedirectPath(raw string) string {
	next := strings.TrimSpace(raw)
	if next == "" {
		return routepath.AppDashboard
	}
	parsed, err := url.Parse(next)
	if err != nil || parsed.Scheme != "" || parsed.Host != "" {
		return routepath.AppDashboard
	}
	if !strings.HasPrefix(parsed.Path, routepath.AppPrefix) {
		return routepath.AppDashboard
	}
	if parsed.Path == "" {
		return routepath.AppDashboard
	}
	if parsed.RawQuery != "" {
		return parsed.Path + "?" + parsed.RawQuery
	}
	return parsed.Path
}
