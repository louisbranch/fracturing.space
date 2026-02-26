package web

import (
	appfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/app"
)

type appRouteDependencies struct {
	appHomeDependencies appfeature.AppHomeDependencies
}

func (h *handler) appRouteDependencies() appRouteDependencies {
	return appRouteDependencies{
		appHomeDependencies: h.appHomeDependenciesImpl(),
	}
}
