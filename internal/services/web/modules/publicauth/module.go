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
	authBaseURL    string
	requestMeta    requestmeta.SchemePolicy
	id             string
	prefix         string
	registerRoutes func(*http.ServeMux, handlers)
}

// Surface classifies which route subset this module instance mounts.
// Composition selects the surface directly so route ownership stays in the root
// publicauth package instead of wrapper packages.
type Surface string

const (
	SurfaceAll          Surface = "all"
	SurfaceShell        Surface = "shell"
	SurfacePasskeys     Surface = "passkeys"
	SurfaceAuthRedirect Surface = "auth-redirect"
)

// Config defines constructor dependencies for a publicauth module.
type Config struct {
	Gateway     publicauthapp.Gateway
	RequestMeta requestmeta.SchemePolicy
	AuthBaseURL string
	Surface     Surface
}

// New returns a publicauth module with explicit dependencies.
func New(config Config) Module {
	id, prefix, register := resolveSurface(config.Surface)
	return Module{
		gateway:        config.Gateway,
		authBaseURL:    config.AuthBaseURL,
		requestMeta:    config.RequestMeta,
		id:             id,
		prefix:         prefix,
		registerRoutes: register,
	}
}

// resolveSurface converts surface selection into mount metadata.
func resolveSurface(surface Surface) (string, string, func(*http.ServeMux, handlers)) {
	switch surface {
	case SurfaceShell:
		return "public", routepath.Root, registerShellRoutes
	case SurfacePasskeys:
		return "public-passkeys", routepath.PasskeysPrefix, registerPasskeyRoutes
	case SurfaceAuthRedirect:
		return "public-auth-redirect", routepath.AuthPrefix, registerAuthRedirectRoutes
	case SurfaceAll, "":
		return "public", routepath.Root, registerRoutes
	default:
		return "public", routepath.Root, registerRoutes
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
	svc := publicauthapp.NewService(m.gateway, m.authBaseURL)
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
