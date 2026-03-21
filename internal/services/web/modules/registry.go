package modules

import (
	"log/slog"

	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/dashboardsync"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
)

// RegistryBuilder builds public/protected module sets from registry inputs.
type RegistryBuilder interface {
	Build(RegistryInput) RegistryOutput
}

// Registry builds public/protected module sets from composition inputs.
type Registry struct{}

// RegistryInput carries the dependencies and options needed to compose module sets.
type RegistryInput struct {
	Dependencies     Dependencies
	Principal        principal.PrincipalResolver
	PublicOptions    PublicModuleOptions
	ProtectedOptions ProtectedModuleOptions
}

// RegistryOutput contains the composed module sets.
type RegistryOutput struct {
	Public    []module.Module
	Protected []module.Module
}

// NewRegistryBuilder returns the default web module registry builder.
func NewRegistryBuilder() RegistryBuilder {
	return Registry{}
}

// Build composes module sets for the requested stability mode.
func (Registry) Build(input RegistryInput) RegistryOutput {
	publicOptions := input.PublicOptions
	protectedOptions := input.ProtectedOptions
	shared := newSharedServices(input.Dependencies, registryLogger(publicOptions.Logger, protectedOptions.Logger))
	if publicOptions.DashboardSync == nil {
		publicOptions.DashboardSync = shared.dashboardSync
	}
	if protectedOptions.DashboardSync == nil {
		protectedOptions.DashboardSync = shared.dashboardSync
	}

	publicModules := defaultPublicModules(input.Dependencies, input.Principal, publicOptions)
	protectedModules := buildProtectedModules(
		input.Dependencies,
		input.Principal,
		protectedOptions,
	)

	return RegistryOutput{
		Public:    publicModules,
		Protected: protectedModules,
	}
}

// PublicModuleOptions controls variant behavior for public module composition.
type PublicModuleOptions struct {
	RequestSchemePolicy requestmeta.SchemePolicy
	DashboardSync       dashboardsync.Service
	Logger              *slog.Logger
}

// ProtectedModuleOptions controls variant behavior for protected module composition.
type ProtectedModuleOptions struct {
	// PlayFallbackPort is the derived play service port used when no subdomain router is present.
	PlayFallbackPort string

	// PlayLaunchGrant signs redirects from the web campaign game route to play.
	PlayLaunchGrant playlaunchgrant.Config

	// RequestSchemePolicy controls scheme resolution for scheme-sensitive behavior in protected modules.
	RequestSchemePolicy requestmeta.SchemePolicy

	// DashboardSync coordinates shared dashboard freshness after successful mutations.
	DashboardSync dashboardsync.Service

	// Logger carries the runtime-owned logger for module composition and shared helpers.
	Logger *slog.Logger
}

// registryLogger chooses the root-owned runtime logger for registry-built helpers.
func registryLogger(publicLogger, protectedLogger *slog.Logger) *slog.Logger {
	if protectedLogger != nil {
		return protectedLogger
	}
	return publicLogger
}
