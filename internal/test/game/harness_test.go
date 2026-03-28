//go:build scenario

package game

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/test/testkit"
)

var (
	joinGrantIssuer   = "scenario-issuer"
	joinGrantAudience = "game-service"
)

func scenarioTimeout() time.Duration {
	return 10 * time.Second
}

func startGRPCServer(t *testing.T) (string, string, func()) {
	t.Helper()

	runtime := testkit.StartGameRuntime(t, testkit.GameRuntimeConfig{
		ContentSeedProfile: testkit.ContentSeedProfileScenario,
		JoinGrantIssuer:    joinGrantIssuer,
		JoinGrantAudience:  joinGrantAudience,
	})
	return runtime.GameAddr, runtime.AuthAddr, func() {}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve runtime caller")
	}

	dir := filepath.Dir(filename)
	for {
		candidate := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(candidate); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	t.Fatalf("go.mod not found from %s", filename)
	return ""
}

func waitForGRPCHealth(t *testing.T, addr string) {
	t.Helper()
	testkit.WaitForGRPCHealth(t, addr)
}

func createAuthUser(t *testing.T, authAddr, displayName string) string {
	t.Helper()
	return testkit.CreateAuthUser(t, authAddr, displayName)
}
