package settings

import (
	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	settingsgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

func newSettingsConfigFromGateways(
	accountGateway settingsapp.AccountGateway,
	aiGateway settingsapp.AIGateway,
	base modulehandler.Base,
	opts ...func(*Config),
) Config {
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
	config := Config{
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
		Base: base,
	}
	for _, opt := range opts {
		opt(&config)
	}
	return config
}

func newSettingsModuleFromGateways(
	accountGateway settingsapp.AccountGateway,
	aiGateway settingsapp.AIGateway,
	base modulehandler.Base,
	opts ...func(*Config),
) Module {
	return New(newSettingsConfigFromGateways(accountGateway, aiGateway, base, opts...))
}

func withFlashMeta(policy requestmeta.SchemePolicy) func(*Config) {
	return func(config *Config) {
		config.FlashMeta = policy
	}
}

func withDashboardSync(sync DashboardSync) func(*Config) {
	return func(config *Config) {
		config.DashboardSync = sync
	}
}

func testSettingsGateway(
	social settingsgateway.SocialClient,
	account settingsgateway.AccountClient,
	passkey settingsgateway.PasskeyClient,
	credential settingsgateway.CredentialClient,
	agent settingsgateway.AgentClient,
) settingsgateway.GRPCGateway {
	return settingsgateway.GRPCGateway{
		SocialClient:     social,
		AccountClient:    account,
		PasskeyClient:    passkey,
		CredentialClient: credential,
		AgentClient:      agent,
	}
}
