package eventtransport

import (
	"os"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
)

// testRuntime is a shared write-path runtime configured once for all tests
// in this package. It replaces the previous package-level global writeRuntime.
var testRuntime *domainwrite.Runtime

func TestMain(m *testing.M) {
	testRuntime = gametest.SetupRuntime()
	os.Exit(m.Run())
}
