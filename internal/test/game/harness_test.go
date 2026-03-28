//go:build scenario

package game

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/test/testkit"
)

var (
	joinGrantIssuer     = "scenario-issuer"
	joinGrantAudience   = "game-service"
	joinGrantKeyOnce    sync.Once
	joinGrantPrivateKey ed25519.PrivateKey
	joinGrantPublicKey  ed25519.PublicKey
)

const (
	testAISessionGrantIssuer   = "fracturing-space-game"
	testAISessionGrantAudience = "fracturing-space-ai"
	testAISessionGrantTTL      = "10m"
	testAISessionGrantHMACKey  = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY"
)

func scenarioTimeout() time.Duration {
	return 10 * time.Second
}

func startGRPCServer(t *testing.T) (string, string, func()) {
	t.Helper()

	mesh := testkit.NewMesh(t, testkit.MeshConfig{
		ContentSeedProfile: testkit.ContentSeedProfileScenario,
	})
	setJoinGrantEnv(t)
	setAISessionGrantEnv(t)
	authAddr := mesh.StartAuthServer()
	addr := mesh.StartGameServer()
	return addr, authAddr, func() {}
}

func setJoinGrantEnv(t *testing.T) {
	t.Helper()

	joinGrantKeyOnce.Do(func() {
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate join grant key: %v", err)
		}
		joinGrantPublicKey = publicKey
		joinGrantPrivateKey = privateKey
	})

	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", joinGrantIssuer)
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", joinGrantAudience)
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY", base64.RawStdEncoding.EncodeToString(joinGrantPublicKey))
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY", base64.RawStdEncoding.EncodeToString(joinGrantPrivateKey))
}

func setAISessionGrantEnv(t *testing.T) {
	t.Helper()
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_ISSUER", testAISessionGrantIssuer)
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_AUDIENCE", testAISessionGrantAudience)
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_HMAC_KEY", testAISessionGrantHMACKey)
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_TTL", testAISessionGrantTTL)
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
