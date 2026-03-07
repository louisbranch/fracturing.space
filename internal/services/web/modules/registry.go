package modules

import (
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

// Registry builds public/protected module sets from composition inputs.
type Registry struct{}

// BuildInput carries the dependencies and options needed to compose module sets.
type BuildInput struct {
	Dependencies     Dependencies
	Resolvers        ModuleResolvers
	PublicOptions    PublicModuleOptions
	ProtectedOptions ProtectedModuleOptions
}

// BuildOutput contains the composed module sets.
type BuildOutput struct {
	Public    []Module
	Protected []Module
}

// NewRegistry returns the default web module registry.
func NewRegistry() Registry {
	return Registry{}
}

// Build composes module sets for the requested stability mode.
func (Registry) Build(input BuildInput) BuildOutput {
	publicModules := defaultPublicModules(input.Dependencies, input.Resolvers, input.PublicOptions)
	protectedModules := buildProtectedModules(
		input.Dependencies,
		input.Resolvers,
		input.ProtectedOptions,
	)

	return BuildOutput{
		Public:    publicModules,
		Protected: protectedModules,
	}
}

// PublicModuleOptions controls variant behavior for public module composition.
type PublicModuleOptions struct {
	RequestSchemePolicy requestmeta.SchemePolicy
}

// ProtectedModuleOptions controls variant behavior for protected module composition.
type ProtectedModuleOptions struct {
	// ChatFallbackPort is the derived chat service port passed to the campaigns module.
	ChatFallbackPort string

	// RequestSchemePolicy controls scheme resolution for scheme-sensitive behavior in protected modules.
	RequestSchemePolicy requestmeta.SchemePolicy
}
