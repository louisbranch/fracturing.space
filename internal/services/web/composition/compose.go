package composition

import (
	"net/http"

	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	webapp "github.com/louisbranch/fracturing.space/internal/services/web/app"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

// PrincipalResolvers carries request-scoped resolution callbacks built by the
// server from principal dependencies.
type PrincipalResolvers struct {
	AuthRequired    func(*http.Request) bool
	ResolveViewer   module.ResolveViewer
	ResolveSignedIn module.ResolveSignedIn
	ResolveUserID   module.ResolveUserID
	ResolveLanguage module.ResolveLanguage
}

// ModuleRegistry builds web module sets from composition input.
type ModuleRegistry interface {
	Build(modules.BuildInput) modules.BuildOutput
}

// ComposeInput describes the contracts needed to compose the application mux.
type ComposeInput struct {
	Principal PrincipalResolvers

	ModuleDependencies modules.Dependencies

	EnableExperimentalModules bool
	ChatHTTPAddr              string
	RequestSchemePolicy       requestmeta.SchemePolicy

	Registry ModuleRegistry
}

// ComposeAppHandler builds the web app handler with selected module sets.
func ComposeAppHandler(input ComposeInput) (http.Handler, error) {
	authRequired := input.Principal.AuthRequired
	if authRequired == nil {
		authRequired = func(*http.Request) bool { return false }
	}

	registry := input.Registry
	if registry == nil {
		defaultRegistry := modules.NewRegistry()
		registry = defaultRegistry
	}

	built := registry.Build(modules.BuildInput{
		Dependencies: input.ModuleDependencies,
		Resolvers: modules.ModuleResolvers{
			ResolveViewer:   input.Principal.ResolveViewer,
			ResolveSignedIn: input.Principal.ResolveSignedIn,
			ResolveUserID:   input.Principal.ResolveUserID,
			ResolveLanguage: input.Principal.ResolveLanguage,
		},
		PublicOptions: modules.PublicModuleOptions{
			RequestSchemePolicy: input.RequestSchemePolicy,
		},
		ProtectedOptions: modules.ProtectedModuleOptions{
			ChatFallbackPort:    websupport.ResolveChatFallbackPort(input.ChatHTTPAddr),
			RequestSchemePolicy: input.RequestSchemePolicy,
		},
		EnableExperimentalModules: input.EnableExperimentalModules,
	})

	return webapp.Compose(webapp.ComposeInput{
		AuthRequired:        authRequired,
		PublicModules:       built.Public,
		ProtectedModules:    built.Protected,
		RequestSchemePolicy: input.RequestSchemePolicy,
	})
}
