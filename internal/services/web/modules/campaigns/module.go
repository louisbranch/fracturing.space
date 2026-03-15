package campaigns

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	"github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides authenticated campaign workspace routes.
type Module struct {
	handlers handlers
	mountErr error
}

// Config defines constructor dependencies for a campaigns module.
type Config struct {
	Services         handlerServices
	Base             modulehandler.Base
	PlayFallbackPort string
	PlayLaunchGrant  playlaunchgrant.Config
	RequestMeta      requestmeta.SchemePolicy
	Systems          campaignSystemRegistry
	DashboardSync    DashboardSync
}

// New returns a campaigns module with explicit dependencies.
func New(config Config) Module {
	handlerSet, err := newHandlers(handlersConfig{
		Services:         config.Services,
		Base:             config.Base,
		PlayFallbackPort: config.PlayFallbackPort,
		PlayLaunchGrant:  config.PlayLaunchGrant,
		RequestMeta:      config.RequestMeta,
		Sync:             config.DashboardSync,
		Systems:          config.Systems,
	})
	return Module{
		handlers: handlerSet,
		mountErr: err,
	}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "campaigns" }

// Mount wires campaign route handlers.
func (m Module) Mount() (module.Mount, error) {
	if m.mountErr != nil {
		return module.Mount{}, m.mountErr
	}
	mux := http.NewServeMux()
	registerStableRoutes(mux, m.handlers)
	return module.Mount{Prefix: routepath.CampaignsPrefix, CanonicalRoot: true, Handler: mux}, nil
}
