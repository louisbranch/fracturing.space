package web

import (
	"net/http"

	appfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/app"
	authfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/auth"
)

func (h *handler) publicAuthRouteHandlers() authfeature.PublicHandlers {
	deps := h.publicAuthRouteDependencies()
	return authfeature.PublicHandlers{
		Root: func(w http.ResponseWriter, r *http.Request) {
			appfeature.HandleAppRoot(deps.appRootDependencies, w, r)
		},
		Login: func(w http.ResponseWriter, r *http.Request) {
			authfeature.HandleAuthLoginPage(deps.authFlowDependencies, w, r)
		},
		AuthLogin: func(w http.ResponseWriter, r *http.Request) {
			authfeature.HandleAuthLogin(deps.authFlowDependencies, w, r)
		},
		AuthCallback: func(w http.ResponseWriter, r *http.Request) {
			authfeature.HandleAuthCallback(deps.authFlowDependencies, w, r)
		},
		AuthLogout: func(w http.ResponseWriter, r *http.Request) {
			authfeature.HandleAuthLogout(deps.authFlowDependencies, w, r)
		},
		MagicLink: func(w http.ResponseWriter, r *http.Request) {
			authfeature.HandleMagicLink(deps.authFlowDependencies, w, r)
		},
		PasskeyRegisterStart: func(w http.ResponseWriter, r *http.Request) {
			authfeature.HandlePasskeyRegisterStart(deps.authFlowDependencies, w, r)
		},
		PasskeyRegisterFinish: func(w http.ResponseWriter, r *http.Request) {
			authfeature.HandlePasskeyRegisterFinish(deps.authFlowDependencies, w, r)
		},
		PasskeyLoginStart: func(w http.ResponseWriter, r *http.Request) {
			authfeature.HandlePasskeyLoginStart(deps.authFlowDependencies, w, r)
		},
		PasskeyLoginFinish: func(w http.ResponseWriter, r *http.Request) {
			authfeature.HandlePasskeyLoginFinish(deps.authFlowDependencies, w, r)
		},
		Health: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		},
	}
}
