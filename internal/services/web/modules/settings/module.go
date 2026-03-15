package settings

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/module"
	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/dashboardsync"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides authenticated settings routes.
type Module struct {
	services     handlerServices
	availability settingsSurfaceAvailability
	base         modulehandler.Base
	flashMeta    requestmeta.SchemePolicy
	sync         DashboardSync
}

// Config defines constructor dependencies for a settings module.
type Config struct {
	Services      handlerServices
	Availability  settingsSurfaceAvailability
	Base          modulehandler.Base
	FlashMeta     requestmeta.SchemePolicy
	DashboardSync DashboardSync
}

// New returns a settings module with explicit dependencies.
func New(config Config) Module {
	services := config.Services
	if services.Account == nil {
		services.Account = settingsapp.NewAccountService(settingsapp.AccountServiceConfig{})
	}
	if services.AI == nil {
		services.AI = settingsapp.NewAIService(settingsapp.AIServiceConfig{})
	}
	sync := config.DashboardSync
	if sync == nil {
		sync = dashboardsync.Noop{}
	}
	return Module{
		services:     services,
		availability: config.Availability,
		base:         config.Base,
		flashMeta:    config.FlashMeta,
		sync:         sync,
	}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "settings" }

// Mount wires settings route handlers.
func (m Module) Mount() (module.Mount, error) {
	mux := http.NewServeMux()
	h := newHandlers(handlersConfig{
		Services:     m.services,
		Availability: m.availability,
		Base:         m.base,
		Policy:       m.flashMeta,
		Sync:         m.sync,
	})
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.SettingsPrefix, CanonicalRoot: true, Handler: mux}, nil
}
