package publicauth

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// registerRoutes centralizes this web behavior in one helper seam.
func registerRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	registerShellRoutes(mux, h)
	registerPasskeyRoutes(mux, h)
	registerAuthRedirectRoutes(mux, h)
}

// registerShellRoutes centralizes this web behavior in one helper seam.
func registerShellRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.Root+"{$}", h.handleRoot)
	mux.HandleFunc(http.MethodGet+" "+routepath.Login, h.handleLogin)
	mux.HandleFunc(http.MethodGet+" "+routepath.LoginRecovery, h.handleRecoveryGet)
	mux.HandleFunc(http.MethodGet+" "+routepath.LoginRecoveryCode, h.handleRecoveryCodeGet)
	mux.HandleFunc(http.MethodGet+" "+routepath.Health, h.handleHealth)

	mux.HandleFunc(http.MethodPost+" "+routepath.LoginRecoveryCodeAcknowledge, h.handleRecoveryCodeAcknowledge)
	mux.HandleFunc(http.MethodGet+" "+routepath.LoginRecoveryCodeAcknowledge, httpx.MethodNotAllowed(http.MethodPost))
	mux.HandleFunc(http.MethodPost+" "+routepath.Logout, h.handleLogout)
	mux.HandleFunc(http.MethodGet+" "+routepath.Logout, httpx.MethodNotAllowed(http.MethodPost))
	mux.HandleFunc(http.MethodGet+" "+routepath.Root+"{rest...}", h.handleNotFound)
}

// registerPasskeyRoutes centralizes this web behavior in one helper seam.
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

	mux.HandleFunc(http.MethodPost+" "+routepath.PasskeyRecoveryStart, h.handleRecoveryStart)
	mux.HandleFunc(http.MethodGet+" "+routepath.PasskeyRecoveryStart, httpx.MethodNotAllowed(http.MethodPost))

	mux.HandleFunc(http.MethodPost+" "+routepath.PasskeyRecoveryFinish, h.handleRecoveryFinish)
	mux.HandleFunc(http.MethodGet+" "+routepath.PasskeyRecoveryFinish, httpx.MethodNotAllowed(http.MethodPost))
	mux.HandleFunc(http.MethodGet+" "+routepath.PasskeysPrefix+"{rest...}", h.handleNotFound)
}

// registerAuthRedirectRoutes centralizes this web behavior in one helper seam.
func registerAuthRedirectRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AuthLogin, h.handleAuthLogin)
}
