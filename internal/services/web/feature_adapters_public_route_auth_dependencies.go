package web

import (
	appfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/app"
	authfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/auth"
)

type publicAuthRouteDependencies struct {
	appRootDependencies  appfeature.AppRootDependencies
	authFlowDependencies authfeature.AuthFlowDependencies
}

func (h *handler) publicAuthRouteDependencies() publicAuthRouteDependencies {
	return publicAuthRouteDependencies{
		appRootDependencies:  h.appRootDependenciesImpl(),
		authFlowDependencies: h.authFlowDependenciesImpl(),
	}
}
