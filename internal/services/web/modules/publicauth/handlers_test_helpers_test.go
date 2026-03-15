package publicauth

import (
	publicauthapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
)

type gatewayServices interface {
	publicauthapp.SessionGateway
	publicauthapp.PasskeyGateway
	publicauthapp.RecoveryGateway
}

func newHandlersFromGateway(
	gateway gatewayServices,
	authBaseURL string,
	policy requestmeta.SchemePolicy,
	resolvers ...principal.PrincipalResolver,
) handlers {
	var resolver principal.PrincipalResolver
	if len(resolvers) > 0 {
		resolver = resolvers[0]
	}
	return newHandlers(handlersConfig{
		Pages:     publicauthapp.NewPageService(authBaseURL),
		Session:   publicauthapp.NewSessionService(gateway, authBaseURL),
		Passkeys:  publicauthapp.NewPasskeyService(gateway, authBaseURL),
		Recovery:  publicauthapp.NewRecoveryService(gateway, authBaseURL),
		Policy:    policy,
		Principal: resolver,
	})
}
