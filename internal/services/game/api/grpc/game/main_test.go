package game

import (
	"os"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
)

// testRuntime is a shared write-path runtime configured once for all tests
// in this package. It replaces the previous package-level global writeRuntime.
var testRuntime *domainwrite.Runtime

func TestMain(m *testing.M) {
	registries, err := engine.BuildRegistries(systemmanifest.Modules()...)
	if err != nil {
		panic("build registries for test: " + err.Error())
	}
	testRuntime = domainwrite.NewRuntime()
	testRuntime.SetIntentFilter(registries.Events)
	os.Exit(m.Run())
}
