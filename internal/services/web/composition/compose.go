package composition

import (
	"log/slog"
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	webapp "github.com/louisbranch/fracturing.space/internal/services/web/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
)

// ComposeInput describes the contracts needed to compose the application mux.
type ComposeInput struct {
	Principal principal.PrincipalResolver
	Logger    *slog.Logger

	ModuleDependencies modules.Dependencies

	PlayHTTPAddr        string
	PlayLaunchGrant     playlaunchgrant.Config
	RequestSchemePolicy requestmeta.SchemePolicy

	RegistryBuilder modules.RegistryBuilder
}

// ComposeAppHandler builds the web app handler with selected module sets.
func ComposeAppHandler(input ComposeInput) (http.Handler, error) {
	registryBuilder := input.RegistryBuilder
	if registryBuilder == nil {
		registryBuilder = modules.NewRegistryBuilder()
	}
	var authRequired func(*http.Request) bool
	if input.Principal != nil {
		authRequired = input.Principal.AuthRequired
	}

	built := registryBuilder.Build(modules.RegistryInput{
		Dependencies: input.ModuleDependencies,
		Principal:    input.Principal,
		PublicOptions: modules.PublicModuleOptions{
			RequestSchemePolicy: input.RequestSchemePolicy,
			Logger:              input.Logger,
		},
		ProtectedOptions: modules.ProtectedModuleOptions{
			PlayFallbackPort:    websupport.ResolveHTTPFallbackPort(input.PlayHTTPAddr),
			PlayLaunchGrant:     input.PlayLaunchGrant,
			RequestSchemePolicy: input.RequestSchemePolicy,
			Logger:              input.Logger,
		},
	})

	return webapp.Compose(webapp.ComposeInput{
		AuthRequired:        authRequired,
		PublicModules:       built.Public,
		ProtectedModules:    built.Protected,
		RequestSchemePolicy: input.RequestSchemePolicy,
	})
}
