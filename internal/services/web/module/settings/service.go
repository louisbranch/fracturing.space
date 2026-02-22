package settings

import (
	"net/http"

	moduleruntime "github.com/louisbranch/fracturing.space/internal/services/web/module/runtime"
)

// Handlers configures callback-backed settings service construction.
type Handlers struct {
	Settings            http.HandlerFunc
	UserProfileSettings http.HandlerFunc
	AIKeys              http.HandlerFunc
	AIKeyRevoke         moduleruntime.StringParamHandler
}

type callbackService struct {
	handlers Handlers
}

// NewService builds a settings Service backed by handler callbacks.
func NewService(handlers Handlers) Service {
	return callbackService{handlers: handlers}
}

func (s callbackService) HandleSettings(w http.ResponseWriter, r *http.Request) {
	moduleruntime.CallOrNotFound(w, r, s.handlers.Settings)
}

func (s callbackService) HandleUserProfileSettings(w http.ResponseWriter, r *http.Request) {
	moduleruntime.CallOrNotFound(w, r, s.handlers.UserProfileSettings)
}

func (s callbackService) HandleAIKeys(w http.ResponseWriter, r *http.Request) {
	moduleruntime.CallOrNotFound(w, r, s.handlers.AIKeys)
}

func (s callbackService) HandleAIKeyRevoke(w http.ResponseWriter, r *http.Request, credentialID string) {
	moduleruntime.CallStringOrNotFound(w, r, s.handlers.AIKeyRevoke, credentialID)
}
