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
	gateway   settingsapp.Gateway
	base      modulehandler.Base
	flashMeta requestmeta.SchemePolicy
	sync      DashboardSync
}

// Config defines constructor dependencies for a settings module.
type Config struct {
	Gateway       settingsapp.Gateway
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
	account := settingsapp.NewAccountService(settingsapp.AccountServiceConfig{
		ProfileGateway:  m.gateway,
		LocaleGateway:   m.gateway,
		SecurityGateway: m.gateway,
	})
	ai := settingsapp.NewAIService(settingsapp.AIServiceConfig{
		AIKeyGateway:   m.gateway,
		AIAgentGateway: m.gateway,
	})
	availability := settingsSurfaceAvailability{
		profile:  settingsapp.IsProfileGatewayHealthy(m.gateway),
		locale:   settingsapp.IsLocaleGatewayHealthy(m.gateway),
		security: settingsapp.IsSecurityGatewayHealthy(m.gateway),
		aiKeys:   settingsapp.IsAIKeyGatewayHealthy(m.gateway),
		aiAgents: settingsapp.IsAIAgentGatewayHealthy(m.gateway),
	}
	h := newHandlers(handlersConfig{
		Services: handlerServices{
			Account: account,
			AI:      ai,
		},
		Availability: availability,
		Base:         m.base,
		Policy:       m.flashMeta,
		Sync:         m.sync,
	})
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.SettingsPrefix, Handler: mux}, nil
}
