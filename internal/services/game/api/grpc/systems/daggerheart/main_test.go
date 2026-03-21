package daggerheart

import (
	"os"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/gameplaystores"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
)

// testRuntime is a shared write-path runtime configured once for all tests
// in this package. It replaces the previous package-level global writeRuntime.
var testRuntime *domainwrite.Runtime

type Stores = gameplaystores.Stores
type StoresFromProjectionConfig = gameplaystores.FromProjectionConfig

var NewStoresFromProjection = gameplaystores.NewFromProjection

func TestMain(m *testing.M) {
	registries, err := engine.BuildRegistries(systemmanifest.Modules()...)
	if err != nil {
		panic("build registries for test: " + err.Error())
	}
	testRuntime = domainwrite.NewRuntime()
	testRuntime.SetIntentFilter(registries.Events)
	os.Exit(m.Run())
}
