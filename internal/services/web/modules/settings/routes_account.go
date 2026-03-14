package settings

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// registerAccountRoutes wires profile, locale, and security settings routes.
func registerAccountRoutes(mux *http.ServeMux, h handlers) {
	mux.HandleFunc(http.MethodGet+" "+routepath.AppSettings, h.redirectSettingsRoot)
	mux.HandleFunc(http.MethodGet+" "+routepath.SettingsPrefix+"{$}", h.redirectSettingsRoot)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppSettingsProfile, h.handleProfileGet)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppSettingsProfile, h.handleProfilePost)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppSettingsLocale, h.handleLocaleGet)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppSettingsLocale, h.handleLocalePost)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppSettingsSecurity, h.handleSecurityGet)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppSettingsSecurityPasskeysStart, h.handleSecurityPasskeyStart)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppSettingsSecurityPasskeysStart, httpx.MethodNotAllowed(http.MethodPost))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppSettingsSecurityPasskeysFinish, h.handleSecurityPasskeyFinish)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppSettingsSecurityPasskeysFinish, httpx.MethodNotAllowed(http.MethodPost))
}
