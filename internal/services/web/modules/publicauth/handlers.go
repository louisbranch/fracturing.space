package publicauth

import (
	"net/http"

	publicauthapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver"
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

// handlerServices groups the publicauth app seams consumed by transport.
type handlerServices struct {
	Pages    publicauthapp.PageService
	Session  publicauthapp.SessionService
	Passkeys publicauthapp.PasskeyService
	Recovery publicauthapp.RecoveryService
}

// newHandlerServicesFromGateway keeps publicauth-owned app-service assembly in
// the publicauth package while transport depends on the narrower route seams.
func newHandlerServicesFromGateway(gateway publicauthapp.Gateway, authBaseURL string) handlerServices {
	return handlerServices{
		Pages:    publicauthapp.NewPageService(gateway, authBaseURL),
		Session:  publicauthapp.NewSessionService(gateway, authBaseURL),
		Passkeys: publicauthapp.NewPasskeyService(gateway, authBaseURL),
		Recovery: publicauthapp.NewRecoveryService(gateway, authBaseURL),
	}
}

// normalizeHandlerServices ensures zero-value module configs fail closed
// instead of leaving nil route services behind.
func normalizeHandlerServices(services handlerServices) handlerServices {
	if services.Pages == nil {
		services.Pages = publicauthapp.NewPageService(nil, "")
	}
	if services.Session == nil {
		services.Session = publicauthapp.NewSessionService(nil, "")
	}
	if services.Passkeys == nil {
		services.Passkeys = publicauthapp.NewPasskeyService(nil, "")
	}
	if services.Recovery == nil {
		services.Recovery = publicauthapp.NewRecoveryService(nil, "")
	}
	return services
}

// handlersConfig keeps transport wiring explicit by owned route surface.
type handlersConfig struct {
	Services  handlerServices
	Policy    requestmeta.SchemePolicy
	Principal requestresolver.PrincipalResolver
}

// newHandlers builds package wiring for this web seam.
func newHandlers(config handlersConfig) handlers {
	return handlers{
		Base:        publichandler.NewBaseFromPrincipal(config.Principal),
		pages:       config.Services.Pages,
		session:     config.Services.Session,
		passkeys:    config.Services.Passkeys,
		recovery:    config.Services.Recovery,
		requestMeta: config.Policy,
	}
}

// handleNotFound handles this route in the module transport layer.
func (h handlers) handleNotFound(w http.ResponseWriter, r *http.Request) {
	h.writeNotFoundPage(w, r)
}
