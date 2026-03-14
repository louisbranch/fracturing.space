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

// ComposeInput describes the contracts needed to compose the application mux.
type ComposeInput struct {
	Principal PrincipalResolvers

	ModuleDependencies modules.Dependencies

	ChatHTTPAddr        string
	RequestSchemePolicy requestmeta.SchemePolicy

	RegistryBuilder modules.RegistryBuilder
}

// ComposeAppHandler builds the web app handler with selected module sets.
func ComposeAppHandler(input ComposeInput) (http.Handler, error) {
	authRequired := input.Principal.AuthRequired
	if authRequired == nil {
		authRequired = func(*http.Request) bool { return false }
	}

	registryBuilder := input.RegistryBuilder
	if registryBuilder == nil {
		registryBuilder = modules.NewRegistryBuilder()
	}

	built := registryBuilder.Build(modules.RegistryInput{
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
	})

	return webapp.Compose(webapp.ComposeInput{
		AuthRequired:        authRequired,
		PublicModules:       built.Public,
		ProtectedModules:    built.Protected,
		RequestSchemePolicy: input.RequestSchemePolicy,
	})
}
