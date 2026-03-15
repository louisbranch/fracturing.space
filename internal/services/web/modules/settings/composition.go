package settings

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
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

// Compose builds the production settings module from area-owned startup
// dependencies.
func Compose(config CompositionConfig) module.Module {
	gateway := settingsgateway.NewGRPCGateway(
		config.SocialClient,
		config.AccountClient,
		config.PasskeyClient,
		config.CredentialClient,
		config.AgentClient,
	)
	return New(Config{
		AccountGateway: gateway,
		AIGateway:      gateway,
		Base:           config.Base,
		FlashMeta:      config.FlashMeta,
		DashboardSync:  config.DashboardSync,
	})
}
