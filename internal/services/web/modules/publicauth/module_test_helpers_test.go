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
		PasskeyService: publicauthapp.NewPasskeyService(gateway, authBaseURL),
		Recovery:       publicauthapp.NewRecoveryService(gateway, authBaseURL),
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

func withSurface(surface Surface) func(*Config) {
	return func(config *Config) {
		config.Surface = surface
	}
}

func newModuleFromGateway(gateway gatewayServices, authBaseURL string, opts ...func(*Config)) Module {
	config := newConfigFromGateway(gateway, authBaseURL)
	for _, opt := range opts {
		opt(&config)
	}
	return New(config)
}
