package settings

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	settingsgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

// CompositionConfig owns the startup wiring required to construct the
// production settings module without leaking gateway internals into the
// registry package.
type CompositionConfig struct {
	Base          modulehandler.Base
	FlashMeta     requestmeta.SchemePolicy
	DashboardSync DashboardSync

	SocialClient     settingsgateway.SocialClient
	AccountClient    settingsgateway.AccountClient
	PasskeyClient    settingsgateway.PasskeyClient
	CredentialClient settingsgateway.CredentialClient
	AgentClient      settingsgateway.AgentClient
}

// ProtectedSurfaceOptions carries the cross-cutting inputs the protected registry is
// allowed to pass into settings composition.
type ProtectedSurfaceOptions struct {
	Base          modulehandler.Base
	FlashMeta     requestmeta.SchemePolicy
	DashboardSync DashboardSync
}

// Compose builds the production settings module from area-owned startup
// dependencies.
func Compose(config CompositionConfig) module.Module {
	accountGateway := settingsgateway.NewAccountGateway(
		config.SocialClient,
		config.AccountClient,
		config.PasskeyClient,
	)
	aiGateway := settingsgateway.NewAIGateway(
		config.CredentialClient,
		config.AgentClient,
	)
	return New(Config{
		Services: handlerServices{
			Account: settingsapp.NewAccountService(settingsapp.AccountServiceConfig{
				ProfileGateway:  accountGateway,
				LocaleGateway:   accountGateway,
				SecurityGateway: accountGateway,
			}),
			AI: settingsapp.NewAIService(settingsapp.AIServiceConfig{
				AIKeyGateway:   aiGateway,
				AIAgentGateway: aiGateway,
			}),
		},
		Availability: settingsSurfaceAvailability{
			profile:  settingsapp.IsProfileGatewayHealthy(accountGateway),
			locale:   settingsapp.IsLocaleGatewayHealthy(accountGateway),
			security: settingsapp.IsSecurityGatewayHealthy(accountGateway),
			aiKeys:   settingsapp.IsAIKeyGatewayHealthy(aiGateway),
			aiAgents: settingsapp.IsAIAgentGatewayHealthy(aiGateway),
		},
		Base:          config.Base,
		FlashMeta:     config.FlashMeta,
		DashboardSync: config.DashboardSync,
	})
}

// ComposeProtected composes the protected settings surface from module-owned
// startup dependencies and shared cross-cutting inputs.
func ComposeProtected(options ProtectedSurfaceOptions, deps Dependencies) module.Module {
	return Compose(newCompositionConfig(options, deps))
}

// newCompositionConfig projects startup dependencies into settings composition
// input.
func newCompositionConfig(options ProtectedSurfaceOptions, deps Dependencies) CompositionConfig {
	return CompositionConfig{
		Base:             options.Base,
		FlashMeta:        options.FlashMeta,
		DashboardSync:    options.DashboardSync,
		SocialClient:     deps.SocialClient,
		AccountClient:    deps.AccountClient,
		PasskeyClient:    deps.PasskeyClient,
		CredentialClient: deps.CredentialClient,
		AgentClient:      deps.AgentClient,
	}
}
