package publicauth

import (
	publicauthapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
)

func newConfigFromGateway(gateway gatewayServices, authBaseURL string) Config {
	return Config{
		PageService:    publicauthapp.NewPageService(authBaseURL),
		SessionService: publicauthapp.NewSessionService(gateway, authBaseURL),
		PasskeyService: publicauthapp.NewPasskeyService(gateway),
		Recovery:       publicauthapp.NewRecoveryService(gateway),
	}
}

func withRequestMeta(policy requestmeta.SchemePolicy) func(*Config) {
	return func(config *Config) {
		config.RequestMeta = policy
	}
}

func withPrincipal(requestPrincipal principal.PrincipalResolver) func(*Config) {
	return func(config *Config) {
		config.Principal = requestPrincipal
	}
}

func newModuleFromGateway(gateway gatewayServices, authBaseURL string, opts ...func(*Config)) Module {
	config := newConfigFromGateway(gateway, authBaseURL)
	for _, opt := range opts {
		opt(&config)
	}
	return newModuleFromConfig(config)
}

func newModuleFromConfig(config Config) Module {
	return NewShell(config)
}

func newModuleFromGatewayWithFactory(
	gateway gatewayServices,
	authBaseURL string,
	newModule func(Config) Module,
	opts ...func(*Config),
) Module {
	config := newConfigFromGateway(gateway, authBaseURL)
	for _, opt := range opts {
		opt(&config)
	}
	return newModule(config)
}
