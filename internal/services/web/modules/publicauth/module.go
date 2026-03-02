package publicauth

import (
	"net/http"
	"strings"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	publicauthapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides unauthenticated root/auth routes.
type Module struct {
	gateway        publicauthapp.Gateway
	requestMeta    requestmeta.SchemePolicy
	id             string
	prefix         string
	registerRoutes func(*http.ServeMux, handlers)
}

// NewWithGateway returns a public/auth module with an explicit gateway.
func NewWithGateway(gateway publicauthapp.Gateway) Module {
	return NewWithGatewayAndPolicy(gateway, requestmeta.SchemePolicy{})
}

// NewWithGatewayAndPolicy returns a public/auth module with explicit request metadata policy.
func NewWithGatewayAndPolicy(gateway publicauthapp.Gateway, policy requestmeta.SchemePolicy) Module {
	return newPublicModule("public", routepath.Root, registerRoutes, gateway, policy)
}

// NewShellWithGateway returns an auth shell module with explicit routes and policy.
func NewShellWithGateway(gateway publicauthapp.Gateway) Module {
	return NewShellWithGatewayAndPolicy(gateway, requestmeta.SchemePolicy{})
}

// NewShellWithGatewayAndPolicy returns the shell/public entrypoint routes module.
func NewShellWithGatewayAndPolicy(gateway publicauthapp.Gateway, policy requestmeta.SchemePolicy) Module {
	return newPublicModule("public", routepath.Root, registerShellRoutes, gateway, policy)
}

// NewPasskeysWithGateway returns a module containing passkey JSON endpoints.
func NewPasskeysWithGateway(gateway publicauthapp.Gateway) Module {
	return NewPasskeysWithGatewayAndPolicy(gateway, requestmeta.SchemePolicy{})
}

// NewPasskeysWithGatewayAndPolicy returns the passkey routes module.
func NewPasskeysWithGatewayAndPolicy(gateway publicauthapp.Gateway, policy requestmeta.SchemePolicy) Module {
	return newPublicModule("public-passkeys", routepath.PasskeysPrefix, registerPasskeyRoutes, gateway, policy)
}

// NewAuthRedirectWithGateway returns an auth-redirect module.
func NewAuthRedirectWithGateway(gateway publicauthapp.Gateway) Module {
	return NewAuthRedirectWithGatewayAndPolicy(gateway, requestmeta.SchemePolicy{})
}

// NewAuthRedirectWithGatewayAndPolicy returns the auth-redirect route module.
func NewAuthRedirectWithGatewayAndPolicy(gateway publicauthapp.Gateway, policy requestmeta.SchemePolicy) Module {
	return newPublicModule("public-auth-redirect", routepath.AuthPrefix, registerAuthRedirectRoutes, gateway, policy)
}

// newPublicModule builds package wiring for this web seam.
func newPublicModule(id string, prefix string, registerRoutes func(*http.ServeMux, handlers), gateway publicauthapp.Gateway, policy requestmeta.SchemePolicy) Module {
	return Module{
		gateway:        gateway,
		requestMeta:    policy,
		id:             strings.TrimSpace(id),
		prefix:         strings.TrimSpace(prefix),
		registerRoutes: registerRoutes,
	}
}

// ID returns a stable identifier for diagnostics and startup logs.
func (m Module) ID() string {
	id := strings.TrimSpace(m.id)
	if id == "" {
		return "public"
	}
	return id
}

// Mount wires public routes under the auth/root prefix.
func (m Module) Mount() (module.Mount, error) {
	mux := http.NewServeMux()
	svc := publicauthapp.NewService(m.gateway)
	h := newHandlers(svc, m.requestMeta)
	if m.registerRoutes != nil {
		m.registerRoutes(mux, h)
	} else {
		registerRoutes(mux, h)
	}
	prefix := strings.TrimSpace(m.prefix)
	if prefix == "" {
		prefix = routepath.Root
	}
	return module.Mount{Prefix: prefix, Handler: mux}, nil
}
