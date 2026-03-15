package publicauth

import (
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver"
)

func newConfigFromGateway(gateway gatewayServices, authBaseURL string) Config {
	return Config{
		Services: newHandlerServicesFromGateway(gateway, authBaseURL),
	}
}

func withRequestMeta(policy requestmeta.SchemePolicy) func(*Config) {
	return func(config *Config) {
		config.RequestMeta = policy
	}
}

func withPrincipal(principal requestresolver.PrincipalResolver) func(*Config) {
	return func(config *Config) {
		config.Principal = principal
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
