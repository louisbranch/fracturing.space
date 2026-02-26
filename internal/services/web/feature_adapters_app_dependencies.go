package web

import (
	"net/http"
	"strings"

	"github.com/a-h/templ"
	appfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	websupport "github.com/louisbranch/fracturing.space/internal/services/web/support"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

func (h *handler) appRootDependenciesImpl() appfeature.AppRootDependencies {
	return appfeature.AppRootDependencies{
		AppName: func() string {
			if h == nil {
				return ""
			}
			return h.resolvedAppName()
		},
		PageContext: func(w http.ResponseWriter, r *http.Request) webtemplates.PageContext {
			if h == nil {
				return webtemplates.PageContext{}
			}
			return h.pageContext(w, r)
		},
		IsAuthenticated: func(r *http.Request) bool {
			if h == nil || h.sessions == nil {
				return false
			}
			return sessionFromRequest(r, h.sessions) != nil
		},
		IsLoginEnabled: func() bool {
			if h == nil {
				return false
			}
			return strings.TrimSpace(h.config.OAuthClientID) != ""
		},
		LandingParams: func() webtemplates.LandingParams {
			if h == nil || strings.TrimSpace(h.config.OAuthClientID) == "" {
				return webtemplates.LandingParams{}
			}
			return webtemplates.LandingParams{
				SignInURL: routepath.AuthLogin,
			}
		},
		RenderLanding: func(page webtemplates.PageContext, params webtemplates.LandingParams) (templ.Component, error) {
			return webtemplates.LandingPage(page, page.AppName, params), nil
		},
		RenderDashboard: func(page webtemplates.PageContext) (templ.Component, error) {
			return webtemplates.DashboardPage(webtemplates.DashboardPageParams{
				AppName:                page.AppName,
				Lang:                   page.Lang,
				UserName:               page.UserName,
				UserAvatarURL:          page.UserAvatarURL,
				HasUnreadNotifications: page.HasUnreadNotifications,
				CurrentPath:            page.CurrentPath,
			}), nil
		},
		ComposeTitle: func(page webtemplates.PageContext, title string, args ...any) string {
			return composeHTMXTitleForPage(page, title, args...)
		},
		WritePage: func(w http.ResponseWriter, r *http.Request, page templ.Component, title string) error {
			if h == nil {
				return websupport.WritePage(w, r, page, title)
			}
			return h.writePage(w, r, page, title)
		},
		LocalizeError: localizeHTTPError,
	}
}

func (h *handler) appHomeDependenciesImpl() appfeature.AppHomeDependencies {
	return appfeature.AppHomeDependencies{
		Authenticate: func(r *http.Request) bool {
			if h == nil || h.sessions == nil {
				return false
			}
			return sessionFromRequest(r, h.sessions) != nil
		},
		AuthLoginPath: routepath.AuthLogin,
		LocalizeError: localizeHTTPError,
	}
}
