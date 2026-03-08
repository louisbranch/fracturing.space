package settings

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/module"
	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides authenticated settings routes.
type Module struct {
	gateway   SettingsGateway
	base      modulehandler.Base
	flashMeta requestmeta.SchemePolicy
	sync      DashboardSync
}

// Config defines constructor dependencies for a settings module.
type Config struct {
	Gateway       SettingsGateway
	Base          modulehandler.Base
	FlashMeta     requestmeta.SchemePolicy
	DashboardSync DashboardSync
}

// New returns a settings module with explicit dependencies.
func New(config Config) Module {
	return Module{
		gateway:   config.Gateway,
		base:      config.Base,
		flashMeta: config.FlashMeta,
		sync:      config.DashboardSync,
	}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "settings" }

// Healthy reports whether the settings module has an operational gateway.
func (m Module) Healthy() bool {
	return settingsapp.IsGatewayHealthy(m.gateway)
}

// Mount wires settings route handlers.
func (m Module) Mount() (module.Mount, error) {
	mux := http.NewServeMux()
	svc := settingsapp.NewService(m.gateway)
	h := newHandlers(svc, m.base, m.flashMeta, m.sync)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.SettingsPrefix, Handler: mux}, nil
}
