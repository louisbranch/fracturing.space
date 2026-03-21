package gametest

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
)

// SetupRuntime builds a write-path runtime configured for tests, using the
// full system manifest. Panics on error so it is safe for TestMain.
func SetupRuntime() *domainwrite.Runtime {
	registries, err := engine.BuildRegistries(systemmanifest.Modules()...)
	if err != nil {
		panic("build registries for test: " + err.Error())
	}
	rt := domainwrite.NewRuntime()
	rt.SetIntentFilter(registries.Events)
	return rt
}
