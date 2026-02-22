package auth

import (
	"net/http"

	moduleruntime "github.com/louisbranch/fracturing.space/internal/services/web/module/runtime"
)

// PublicHandlers configures callback-backed public/auth service construction.
type PublicHandlers struct {
	Root                  http.HandlerFunc
	Login                 http.HandlerFunc
	AuthLogin             http.HandlerFunc
	AuthCallback          http.HandlerFunc
	AuthLogout            http.HandlerFunc
	MagicLink             http.HandlerFunc
	PasskeyRegisterStart  http.HandlerFunc
	PasskeyRegisterFinish http.HandlerFunc
	PasskeyLoginStart     http.HandlerFunc
	PasskeyLoginFinish    http.HandlerFunc
	Health                http.HandlerFunc
}

type callbackPublicService struct {
	handlers PublicHandlers
}

// NewPublicService builds a PublicService backed by handler callbacks.
func NewPublicService(handlers PublicHandlers) PublicService {
	return callbackPublicService{handlers: handlers}
}

func (s callbackPublicService) HandleRoot(w http.ResponseWriter, r *http.Request) {
	moduleruntime.CallOrNotFound(w, r, s.handlers.Root)
}

func (s callbackPublicService) HandleLogin(w http.ResponseWriter, r *http.Request) {
	moduleruntime.CallOrNotFound(w, r, s.handlers.Login)
}

func (s callbackPublicService) HandleAuthLogin(w http.ResponseWriter, r *http.Request) {
	moduleruntime.CallOrNotFound(w, r, s.handlers.AuthLogin)
}

func (s callbackPublicService) HandleAuthCallback(w http.ResponseWriter, r *http.Request) {
	moduleruntime.CallOrNotFound(w, r, s.handlers.AuthCallback)
}

func (s callbackPublicService) HandleAuthLogout(w http.ResponseWriter, r *http.Request) {
	moduleruntime.CallOrNotFound(w, r, s.handlers.AuthLogout)
}

func (s callbackPublicService) HandleMagicLink(w http.ResponseWriter, r *http.Request) {
	moduleruntime.CallOrNotFound(w, r, s.handlers.MagicLink)
}

func (s callbackPublicService) HandlePasskeyRegisterStart(w http.ResponseWriter, r *http.Request) {
	moduleruntime.CallOrNotFound(w, r, s.handlers.PasskeyRegisterStart)
}

func (s callbackPublicService) HandlePasskeyRegisterFinish(w http.ResponseWriter, r *http.Request) {
	moduleruntime.CallOrNotFound(w, r, s.handlers.PasskeyRegisterFinish)
}

func (s callbackPublicService) HandlePasskeyLoginStart(w http.ResponseWriter, r *http.Request) {
	moduleruntime.CallOrNotFound(w, r, s.handlers.PasskeyLoginStart)
}

func (s callbackPublicService) HandlePasskeyLoginFinish(w http.ResponseWriter, r *http.Request) {
	moduleruntime.CallOrNotFound(w, r, s.handlers.PasskeyLoginFinish)
}

func (s callbackPublicService) HandleHealth(w http.ResponseWriter, r *http.Request) {
	moduleruntime.CallOrNotFound(w, r, s.handlers.Health)
}
