package daggerheart

import (
	"os"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
)

func TestMain(m *testing.M) {
	registries, err := engine.BuildRegistries(systemmanifest.Modules()...)
	if err != nil {
		panic("build registries for test: " + err.Error())
	}
	SetIntentFilter(registries.Events)
	os.Exit(m.Run())
}
