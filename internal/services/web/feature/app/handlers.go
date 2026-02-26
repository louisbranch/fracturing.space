package app

import (
	"net/http"

	"github.com/a-h/templ"
	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// AppRootDependencies captures the contracts required by the app root renderer.
type AppRootDependencies struct {
	AppName         func() string
	PageContext     func(http.ResponseWriter, *http.Request) webtemplates.PageContext
	IsAuthenticated func(*http.Request) bool
	IsLoginEnabled  func() bool
	LandingParams   func() webtemplates.LandingParams
	RenderLanding   func(webtemplates.PageContext, webtemplates.LandingParams) (templ.Component, error)
	RenderDashboard func(webtemplates.PageContext) (templ.Component, error)
	ComposeTitle    func(webtemplates.PageContext, string, ...any) string
	WritePage       func(http.ResponseWriter, *http.Request, templ.Component, string) error
	LocalizeError   func(http.ResponseWriter, *http.Request, int, string, ...any)
}

// HandleAppRoot renders authenticated dashboard or landing state for the web root route.
func HandleAppRoot(d AppRootDependencies, w http.ResponseWriter, r *http.Request) {
	if d.PageContext == nil || d.IsAuthenticated == nil || d.ComposeTitle == nil || d.WritePage == nil || d.LocalizeError == nil {
		http.NotFound(w, r)
		return
	}
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	appName := ""
	if d.AppName != nil {
		appName = d.AppName()
	}

	page := d.PageContext(w, r)
	page.AppName = appName

	if d.IsAuthenticated(r) {
		component, err := d.RenderDashboard(page)
		if err != nil {
			d.LocalizeError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
			return
		}
		if err := d.WritePage(w, r, component, d.ComposeTitle(page, "dashboard.title")); err != nil {
			d.LocalizeError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
			return
		}
		return
	}

	params := webtemplates.LandingParams{}
	if d.LandingParams != nil {
		params = d.LandingParams()
	}
	if params.SignInURL == "" && d.IsLoginEnabled != nil && d.IsLoginEnabled() {
		params.SignInURL = routepath.AuthLogin
	}

	component, err := d.RenderLanding(page, params)
	if err != nil {
		d.LocalizeError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
		return
	}
	if err := d.WritePage(w, r, component, d.ComposeTitle(page, "title.landing")); err != nil {
		d.LocalizeError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
	}
}

// AppHomeDependencies captures the dependencies for /app route shaping.
type AppHomeDependencies struct {
	Authenticate  func(*http.Request) bool
	AuthLoginPath string
	LocalizeError func(http.ResponseWriter, *http.Request, int, string, ...any)
}

// HandleAppHome keeps unauthenticated users away from app shell routes.
func HandleAppHome(d AppHomeDependencies, w http.ResponseWriter, r *http.Request) {
	if d.Authenticate == nil || d.LocalizeError == nil {
		http.NotFound(w, r)
		return
	}
	if r.URL.Path == routepath.AppRootPrefix {
		http.Redirect(w, r, routepath.AppRoot, http.StatusMovedPermanently)
		return
	}
	if r.URL.Path != routepath.AppRoot {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if !d.Authenticate(r) {
		http.Redirect(w, r, d.AuthLoginPath, http.StatusFound)
		return
	}
	http.Redirect(w, r, routepath.AppCampaigns, http.StatusFound)
}

// AppDashboardDependencies captures the dependencies for /app shell dashboard checks.
type AppDashboardDependencies struct {
	Authenticate  func(*http.Request) bool
	AuthLoginPath string
	LocalizeError func(http.ResponseWriter, *http.Request, int, string, ...any)
}

// HandleAppDashboard keeps unauthorized users on the auth entrypoint.
func HandleAppDashboard(d AppDashboardDependencies, w http.ResponseWriter, r *http.Request) {
	if d.Authenticate == nil || d.LocalizeError == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if !d.Authenticate(r) {
		http.Redirect(w, r, d.AuthLoginPath, http.StatusFound)
		return
	}
	http.Redirect(w, r, routepath.AppCampaigns, http.StatusFound)
}
