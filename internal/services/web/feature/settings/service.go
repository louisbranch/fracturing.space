package settings

import (
	"net/http"

	routing "github.com/louisbranch/fracturing.space/internal/services/web/feature/routing"
)

// Handlers configures callback-backed settings service construction.
type Handlers struct {
	Settings            http.HandlerFunc
	UserProfileSettings http.HandlerFunc
	AIKeys              http.HandlerFunc
	AIKeyRevoke         routing.StringParamHandler
}

type callbackService struct {
	handlers Handlers
}

// NewService builds a settings Service backed by handler callbacks.
func NewService(handlers Handlers) Service {
	return callbackService{handlers: handlers}
}

func (s callbackService) HandleSettings(w http.ResponseWriter, r *http.Request) {
	routing.CallOrNotFound(w, r, s.handlers.Settings)
}

func (s callbackService) HandleUserProfileSettings(w http.ResponseWriter, r *http.Request) {
	routing.CallOrNotFound(w, r, s.handlers.UserProfileSettings)
}

func (s callbackService) HandleAIKeys(w http.ResponseWriter, r *http.Request) {
	routing.CallOrNotFound(w, r, s.handlers.AIKeys)
}

func (s callbackService) HandleAIKeyRevoke(w http.ResponseWriter, r *http.Request, credentialID string) {
	routing.CallStringOrNotFound(w, r, s.handlers.AIKeyRevoke, credentialID)
}
