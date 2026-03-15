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
	accountGateway settingsapp.AccountGateway
	aiGateway      settingsapp.AIGateway
	base           modulehandler.Base
	flashMeta      requestmeta.SchemePolicy
	sync           DashboardSync
}

// Config defines constructor dependencies for a settings module.
type Config struct {
	AccountGateway settingsapp.AccountGateway
	AIGateway      settingsapp.AIGateway
	Base           modulehandler.Base
	FlashMeta      requestmeta.SchemePolicy
	DashboardSync  DashboardSync
}

// New returns a settings module with explicit dependencies.
func New(config Config) Module {
	accountGateway := config.AccountGateway
	aiGateway := config.AIGateway
	if accountGateway == nil {
		if shared, ok := aiGateway.(settingsapp.AccountGateway); ok {
			accountGateway = shared
		}
	}
	if aiGateway == nil {
		if shared, ok := accountGateway.(settingsapp.AIGateway); ok {
			aiGateway = shared
		}
	}
	return Module{
		accountGateway: accountGateway,
		aiGateway:      aiGateway,
		base:           config.Base,
		flashMeta:      config.FlashMeta,
		sync:           config.DashboardSync,
	}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "settings" }

// Healthy reports whether the settings module has an operational gateway.
func (m Module) Healthy() bool {
	return settingsapp.IsAccountGatewayHealthy(m.accountGateway) || settingsapp.IsAIGatewayHealthy(m.aiGateway)
}

// Mount wires settings route handlers.
func (m Module) Mount() (module.Mount, error) {
	mux := http.NewServeMux()
	account := settingsapp.NewAccountService(settingsapp.AccountServiceConfig{
		ProfileGateway:  m.accountGateway,
		LocaleGateway:   m.accountGateway,
		SecurityGateway: m.accountGateway,
	})
	ai := settingsapp.NewAIService(settingsapp.AIServiceConfig{
		AIKeyGateway:   m.aiGateway,
		AIAgentGateway: m.aiGateway,
	})
	availability := settingsSurfaceAvailability{
		profile:  settingsapp.IsProfileGatewayHealthy(m.accountGateway),
		locale:   settingsapp.IsLocaleGatewayHealthy(m.accountGateway),
		security: settingsapp.IsSecurityGatewayHealthy(m.accountGateway),
		aiKeys:   settingsapp.IsAIKeyGatewayHealthy(m.aiGateway),
		aiAgents: settingsapp.IsAIAgentGatewayHealthy(m.aiGateway),
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
	return module.Mount{Prefix: routepath.SettingsPrefix, CanonicalRoot: true, Handler: mux}, nil
}
