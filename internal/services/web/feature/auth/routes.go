package auth

import (
	"net/http"

	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// PublicService is the public/auth surface contract consumed by public route registration.
type PublicService interface {
	HandleRoot(w http.ResponseWriter, r *http.Request)
	HandleLogin(w http.ResponseWriter, r *http.Request)
	HandleAuthLogin(w http.ResponseWriter, r *http.Request)
	HandleAuthCallback(w http.ResponseWriter, r *http.Request)
	HandleAuthLogout(w http.ResponseWriter, r *http.Request)
	HandleMagicLink(w http.ResponseWriter, r *http.Request)
	HandlePasskeyRegisterStart(w http.ResponseWriter, r *http.Request)
	HandlePasskeyRegisterFinish(w http.ResponseWriter, r *http.Request)
	HandlePasskeyLoginStart(w http.ResponseWriter, r *http.Request)
	HandlePasskeyLoginFinish(w http.ResponseWriter, r *http.Request)
	HandleHealth(w http.ResponseWriter, r *http.Request)
}

// RegisterPublicRoutes wires public and auth routes into the public mux.
func RegisterPublicRoutes(mux *http.ServeMux, service PublicService) {
	if mux == nil || service == nil {
		return
	}
	mux.HandleFunc(routepath.Root, service.HandleRoot)
	mux.HandleFunc(routepath.AuthLogin, service.HandleAuthLogin)
	mux.HandleFunc(routepath.AuthCallback, service.HandleAuthCallback)
	mux.HandleFunc(routepath.AuthLogout, service.HandleAuthLogout)
	mux.HandleFunc(routepath.Login, service.HandleLogin)
	mux.HandleFunc(routepath.MagicLink, service.HandleMagicLink)
	mux.HandleFunc(routepath.PasskeyRegisterStart, service.HandlePasskeyRegisterStart)
	mux.HandleFunc(routepath.PasskeyRegisterFinish, service.HandlePasskeyRegisterFinish)
	mux.HandleFunc(routepath.PasskeyLoginStart, service.HandlePasskeyLoginStart)
	mux.HandleFunc(routepath.PasskeyLoginFinish, service.HandlePasskeyLoginFinish)
	mux.HandleFunc(routepath.Health, service.HandleHealth)
}
