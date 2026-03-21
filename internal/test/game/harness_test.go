//go:build scenario

package game

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/app"
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

	setTempDBPath(t)
	seedScenarioContent(t)
	setTempAuthDBPath(t)
	setJoinGrantEnv(t)
	setAISessionGrantEnv(t)
	authAddr, stopAuth := startAuthServer(t)
	t.Setenv("FRACTURING_SPACE_AUTH_ADDR", authAddr)

	ctx, cancel := context.WithCancel(context.Background())
	grpcServer, err := app.NewWithAddr("127.0.0.1:0")
	if err != nil {
		cancel()
		stopAuth()
		t.Fatalf("new game server: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(ctx)
	}()

	addr := grpcServer.Addr()
	waitForGRPCHealth(t, addr)
	stop := func() {
		cancel()
		select {
		case err := <-serveErr:
			if err != nil {
				t.Fatalf("game server error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for game server to stop")
		}
		stopAuth()
	}

	return addr, authAddr, stop
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

func startAuthServer(t *testing.T) (string, func()) {
	t.Helper()
	return testkit.StartAuthServer(t)
}

func setTempDBPath(t *testing.T) {
	t.Helper()
	testkit.SetTempGameDBPaths(t)
}

func seedScenarioContent(t *testing.T) {
	t.Helper()
	testkit.SeedDaggerheartContent(t, testkit.ContentSeedProfileScenario)
}

func setTempAuthDBPath(t *testing.T) {
	t.Helper()
	testkit.SetTempAuthDBPath(t)
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
