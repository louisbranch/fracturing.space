package public

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func registerRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	registerShellRoutes(mux, h)
	registerPasskeyRoutes(mux, h)
	registerAuthRedirectRoutes(mux, h)
}

func registerShellRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.Root+"{$}", h.handleRoot)
	mux.HandleFunc(http.MethodGet+" "+routepath.Login, h.handleLogin)
	mux.HandleFunc(http.MethodGet+" "+routepath.Health, h.handleHealth)

	mux.HandleFunc(http.MethodPost+" "+routepath.Logout, h.handleLogout)
	mux.HandleFunc(http.MethodGet+" "+routepath.Logout, httpx.MethodNotAllowed(http.MethodPost))
	mux.HandleFunc(http.MethodGet+" "+routepath.Root+"{rest...}", h.handleNotFound)
}

func registerPasskeyRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodPost+" "+routepath.PasskeyLoginStart, h.handlePasskeyLoginStart)
	mux.HandleFunc(http.MethodGet+" "+routepath.PasskeyLoginStart, httpx.MethodNotAllowed(http.MethodPost))

	mux.HandleFunc(http.MethodPost+" "+routepath.PasskeyLoginFinish, h.handlePasskeyLoginFinish)
	mux.HandleFunc(http.MethodGet+" "+routepath.PasskeyLoginFinish, httpx.MethodNotAllowed(http.MethodPost))

	mux.HandleFunc(http.MethodPost+" "+routepath.PasskeyRegisterStart, h.handlePasskeyRegisterStart)
	mux.HandleFunc(http.MethodGet+" "+routepath.PasskeyRegisterStart, httpx.MethodNotAllowed(http.MethodPost))

	mux.HandleFunc(http.MethodPost+" "+routepath.PasskeyRegisterFinish, h.handlePasskeyRegisterFinish)
	mux.HandleFunc(http.MethodGet+" "+routepath.PasskeyRegisterFinish, httpx.MethodNotAllowed(http.MethodPost))
	mux.HandleFunc(http.MethodGet+" "+routepath.PasskeysPrefix+"{rest...}", h.handleNotFound)
}

func registerAuthRedirectRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AuthLogin, h.handleAuthLogin)
}
