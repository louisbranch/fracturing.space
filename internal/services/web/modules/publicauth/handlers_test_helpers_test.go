package publicauth

import (
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver"
)

func newHandlersFromGateway(
	gateway gatewayServices,
	authBaseURL string,
	policy requestmeta.SchemePolicy,
	principal ...requestresolver.PrincipalResolver,
) handlers {
	var resolver requestresolver.PrincipalResolver
	if len(principal) > 0 {
		resolver = principal[0]
	}
	return newHandlers(handlersConfig{
		Services:  newHandlerServicesFromGateway(gateway, authBaseURL),
		Policy:    policy,
		Principal: resolver,
	})
}
