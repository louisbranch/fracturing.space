package publicauth

import (
	"net/http"

	publicauthapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
)

// handlers defines an internal contract used at this web package boundary.
type handlers struct {
	publichandler.Base
	pages       publicauthapp.PageService
	session     publicauthapp.SessionService
	passkeys    publicauthapp.PasskeyService
	recovery    publicauthapp.RecoveryService
	requestMeta requestmeta.SchemePolicy
}

// handlersConfig keeps transport wiring explicit by owned route surface.
type handlersConfig struct {
	Pages     publicauthapp.PageService
	Session   publicauthapp.SessionService
	Passkeys  publicauthapp.PasskeyService
	Recovery  publicauthapp.RecoveryService
	Policy    requestmeta.SchemePolicy
	Principal principal.PrincipalResolver
}

// newHandlers builds package wiring for this web seam.
func newHandlers(config handlersConfig) handlers {
	pages := config.Pages
	if pages == nil {
		pages = publicauthapp.NewPageService("")
	}
	session := config.Session
	if session == nil {
		session = publicauthapp.NewSessionService(nil, "")
	}
	passkeys := config.Passkeys
	if passkeys == nil {
		passkeys = publicauthapp.NewPasskeyService(nil, "")
	}
	recovery := config.Recovery
	if recovery == nil {
		recovery = publicauthapp.NewRecoveryService(nil, "")
	}
	return handlers{
		Base:        publichandler.NewBaseFromPrincipal(config.Principal),
		pages:       pages,
		session:     session,
		passkeys:    passkeys,
		recovery:    recovery,
		requestMeta: config.Policy,
	}
}

// handleNotFound handles this route in the module transport layer.
func (h handlers) handleNotFound(w http.ResponseWriter, r *http.Request) {
	h.writeNotFoundPage(w, r)
}
