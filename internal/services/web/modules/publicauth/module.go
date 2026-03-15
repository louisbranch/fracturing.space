package publicauth

import (
	"net/http"
	"strings"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	publicauthapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides unauthenticated root/auth routes.
type Module struct {
	pageService    publicauthapp.PageService
	sessionService publicauthapp.SessionService
	passkeyService publicauthapp.PasskeyService
	recovery       publicauthapp.RecoveryService
	principal      principal.PrincipalResolver
	requestMeta    requestmeta.SchemePolicy
	id             string
	prefix         string
	routeRegister  routeRegisterFunc
}

// routeRegisterFunc is the route registration function for one public surface.
type routeRegisterFunc func(*http.ServeMux, handlers)

// Config defines constructor dependencies for a publicauth module.
type Config struct {
	PageService    publicauthapp.PageService
	SessionService publicauthapp.SessionService
	PasskeyService publicauthapp.PasskeyService
	Recovery       publicauthapp.RecoveryService
	Principal      principal.PrincipalResolver
	RequestMeta    requestmeta.SchemePolicy
}

// NewShell constructs the shell/public routes surface.
func NewShell(config Config) Module {
	return newModule("public", routepath.Root, registerShellRoutes, config)
}

// NewPasskeys constructs the passkey-focused routes surface.
func NewPasskeys(config Config) Module {
	return newModule("public-passkeys", routepath.PasskeysPrefix, registerPasskeyRoutes, config)
}

// NewAuthRedirect constructs the auth-redirect surface.
func NewAuthRedirect(config Config) Module {
	return newModule("public-auth-redirect", routepath.AuthPrefix, registerAuthRedirectRoutes, config)
}

// newModule wires a publicauth module from composition dependencies and route
// registration.
func newModule(id string, prefix string, routeRegister routeRegisterFunc, config Config) Module {
	return Module{
		pageService:    config.PageService,
		sessionService: config.SessionService,
		passkeyService: config.PasskeyService,
		recovery:       config.Recovery,
		principal:      config.Principal,
		requestMeta:    config.RequestMeta,
		id:             id,
		prefix:         prefix,
		routeRegister:  routeRegister,
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
	h := newHandlers(handlersConfig{
		Pages:     m.pageService,
		Session:   m.sessionService,
		Passkeys:  m.passkeyService,
		Recovery:  m.recovery,
		Policy:    m.requestMeta,
		Principal: m.principal,
	})
	if m.routeRegister != nil {
		m.routeRegister(mux, h)
	}
	prefix := strings.TrimSpace(m.prefix)
	if prefix == "" {
		prefix = routepath.Root
	}
	return module.Mount{Prefix: prefix, Handler: mux}, nil
}
