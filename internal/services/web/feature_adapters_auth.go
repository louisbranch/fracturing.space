package web

import (
	"net/http"
	"strings"
	"time"

	"github.com/a-h/templ"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	authfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/auth"
	webcache "github.com/louisbranch/fracturing.space/internal/services/web/infra/cache"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	websupport "github.com/louisbranch/fracturing.space/internal/services/web/support"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"golang.org/x/text/message"
)

var notificationsNow = func() time.Time {
	return time.Now().UTC()
}

func (h *handler) authFlowDependenciesImpl() authfeature.AuthFlowDependencies {
	cfg := resolveAuthConfigImpl(h)
	return authfeature.AuthFlowDependencies{
		AuthClient: func() authv1.AuthServiceClient {
			if h == nil {
				return nil
			}
			return h.authClient
		},
		AuthBaseURL:   cfg.AuthBaseURL,
		AuthTokenURL:  cfg.AuthTokenURL,
		OAuthClientID: cfg.OAuthClientID,
		CallbackURL:   cfg.CallbackURL,
		ConfigDomain:  cfg.Domain,
		AuthLoginPath: routepath.AuthLogin,
		ResolvedAppName: func() string {
			if h == nil {
				return ""
			}
			return h.resolvedAppName()
		},
		BuildAuthConsentURL: webcache.BuildAuthConsentURL,
		CreatePendingState: func(verifier string) string {
			if h == nil || h.pendingFlows == nil {
				return ""
			}
			return h.pendingFlows.create(verifier)
		},
		ConsumePendingState: func(state string) (string, bool) {
			state = strings.TrimSpace(state)
			if h == nil || h.pendingFlows == nil || state == "" {
				return "", false
			}
			flow := h.pendingFlows.consume(state)
			if flow == nil {
				return "", false
			}
			return flow.codeVerifier, true
		},
		CreateSession: func(accessToken, displayName string, expiry time.Time) string {
			if h == nil || h.sessions == nil {
				return ""
			}
			return h.sessions.create(accessToken, displayName, expiry)
		},
		DeleteSession: func(sessionID string) {
			if h == nil || h.sessions == nil || strings.TrimSpace(sessionID) == "" {
				return
			}
			h.sessions.delete(sessionID)
		},
		SessionIDFromRequest: func(r *http.Request) string {
			if r == nil {
				return ""
			}
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil {
				return ""
			}
			return strings.TrimSpace(cookie.Value)
		},
		TokenCookieDomainForRequest: func(requestHost string) string {
			if h == nil {
				return tokenCookieDomainForRequest("", requestHost)
			}
			return tokenCookieDomainForRequest(h.config.Domain, requestHost)
		},
		GenerateCodeVerifier: authfeature.GenerateCodeVerifier,
		ComputeS256Challenge: authfeature.ComputeS256Challenge,
		Localizer:            localizer,
		LocalizeError:        localizeHTTPError,
		RenderMagicPage: func(w http.ResponseWriter, r *http.Request, status int, params webtemplates.MagicParams) {
			authfeature.RenderMagicPage(authfeature.AuthFlowDependencies{}, w, r, status, params)
		},
		WriteJSON: authfeature.WriteJSON,
		WritePage: func(w http.ResponseWriter, r *http.Request, c templ.Component, title string) error {
			if h == nil {
				return websupport.WritePage(w, r, c, title)
			}
			return h.writePage(w, r, c, title)
		},
		ComposeHTMXTitle: func(printer *message.Printer, title string, args ...any) string {
			return websupport.ComposeHTMXTitle(printer, title, args...)
		},
		WriteSessionCookie: func(w http.ResponseWriter, sessionID string) {
			setSessionCookie(w, sessionID)
		},
		WriteTokenCookie: func(w http.ResponseWriter, token, domain string, maxAge int) {
			setTokenCookie(w, token, domain, maxAge)
		},
		ClearSessionCookie: func(w http.ResponseWriter) {
			clearSessionCookie(w)
		},
		ClearTokenCookie: func(w http.ResponseWriter, domain string) {
			clearTokenCookie(w, domain)
		},
	}
}

type authConfigCopy struct {
	AuthBaseURL   string
	AuthTokenURL  string
	OAuthClientID string
	CallbackURL   string
	Domain        string
}

func resolveAuthConfigImpl(h *handler) authConfigCopy {
	if h == nil {
		return authConfigCopy{}
	}
	return authConfigCopy{
		AuthBaseURL:   h.config.AuthBaseURL,
		AuthTokenURL:  h.config.AuthTokenURL,
		OAuthClientID: h.config.OAuthClientID,
		CallbackURL:   h.config.CallbackURL,
		Domain:        h.config.Domain,
	}
}
