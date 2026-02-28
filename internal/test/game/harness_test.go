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
	"strings"
	"sync"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	authserver "github.com/louisbranch/fracturing.space/internal/services/auth/app"
	server "github.com/louisbranch/fracturing.space/internal/services/game/app"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	storagesqlite "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

var (
	joinGrantIssuer     = "scenario-issuer"
	joinGrantAudience   = "game-service"
	joinGrantKeyOnce    sync.Once
	joinGrantPrivateKey ed25519.PrivateKey
	joinGrantPublicKey  ed25519.PublicKey
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
	authAddr, stopAuth := startAuthServer(t)
	t.Setenv("FRACTURING_SPACE_AUTH_ADDR", authAddr)

	ctx, cancel := context.WithCancel(context.Background())
	grpcServer, err := server.NewWithAddr("127.0.0.1:0")
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

func startAuthServer(t *testing.T) (string, func()) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	authServer, err := authserver.New(0, "")
	if err != nil {
		cancel()
		t.Fatalf("new auth server: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- authServer.Serve(ctx)
	}()

	authAddr := authServer.Addr()
	waitForGRPCHealth(t, authAddr)
	stop := func() {
		cancel()
		select {
		case err := <-serveErr:
			if err != nil {
				t.Fatalf("auth server error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for auth server to stop")
		}
	}

	return authAddr, stop
}

func setTempDBPath(t *testing.T) {
	t.Helper()
	base := t.TempDir()
	t.Setenv("FRACTURING_SPACE_GAME_EVENTS_DB_PATH", filepath.Join(base, "game-events.db"))
	t.Setenv("FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH", filepath.Join(base, "game-projections.db"))
	t.Setenv("FRACTURING_SPACE_GAME_CONTENT_DB_PATH", filepath.Join(base, "game-content.db"))
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "test-key")
}

func seedScenarioContent(t *testing.T) {
	t.Helper()

	contentPath := strings.TrimSpace(os.Getenv("FRACTURING_SPACE_GAME_CONTENT_DB_PATH"))
	if contentPath == "" {
		t.Fatal("content DB path env is required")
	}

	store, err := storagesqlite.OpenContent(contentPath)
	if err != nil {
		t.Fatalf("open content store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC()

	if err := store.PutDaggerheartClass(ctx, storage.DaggerheartClass{
		ID:              "class.guardian",
		Name:            "Guardian",
		StartingEvasion: 9,
		StartingHP:      7,
		DomainIDs:       []string{"domain.valor"},
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("seed class: %v", err)
	}

	if err := store.PutDaggerheartSubclass(ctx, storage.DaggerheartSubclass{
		ID:        "subclass.stalwart",
		Name:      "Stalwart",
		ClassID:   "class.guardian",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed subclass: %v", err)
	}

	if err := store.PutDaggerheartHeritage(ctx, storage.DaggerheartHeritage{
		ID:        "heritage.human",
		Name:      "Human",
		Kind:      "ancestry",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed ancestry heritage: %v", err)
	}

	if err := store.PutDaggerheartHeritage(ctx, storage.DaggerheartHeritage{
		ID:        "heritage.highborne",
		Name:      "Highborne",
		Kind:      "community",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed community heritage: %v", err)
	}

	if err := store.PutDaggerheartDomainCard(ctx, storage.DaggerheartDomainCard{
		ID:          "domain_card.valor-bare-bones",
		Name:        "Bare Bones",
		DomainID:    "domain.valor",
		Level:       1,
		Type:        "ability",
		UsageLimit:  "None",
		FeatureText: "Scenario seed card",
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("seed domain card: %v", err)
	}

	if err := store.PutDaggerheartWeapon(ctx, storage.DaggerheartWeapon{
		ID:         "weapon.longsword",
		Name:       "Longsword",
		Category:   "primary",
		Tier:       1,
		Trait:      "Agility",
		Range:      "melee",
		DamageDice: []storage.DaggerheartDamageDie{{Sides: 10, Count: 1}},
		DamageType: "physical",
		Burden:     2,
		Feature:    "Scenario seed weapon",
		CreatedAt:  now,
		UpdatedAt:  now,
	}); err != nil {
		t.Fatalf("seed weapon: %v", err)
	}

	if err := store.PutDaggerheartArmor(ctx, storage.DaggerheartArmor{
		ID:                  "armor.gambeson-armor",
		Name:                "Gambeson Armor",
		Tier:                1,
		BaseMajorThreshold:  6,
		BaseSevereThreshold: 12,
		ArmorScore:          1,
		Feature:             "Scenario seed armor",
		CreatedAt:           now,
		UpdatedAt:           now,
	}); err != nil {
		t.Fatalf("seed armor: %v", err)
	}

	if err := store.PutDaggerheartArmor(ctx, storage.DaggerheartArmor{
		ID:                  "armor.readiness-light",
		Name:                "Readiness Light Armor",
		Tier:                1,
		BaseMajorThreshold:  6,
		BaseSevereThreshold: 12,
		ArmorScore:          0,
		Feature:             "Scenario readiness armor",
		CreatedAt:           now,
		UpdatedAt:           now,
	}); err != nil {
		t.Fatalf("seed readiness armor: %v", err)
	}

	if err := store.PutDaggerheartItem(ctx, storage.DaggerheartItem{
		ID:        "item.minor-health-potion",
		Name:      "Minor Health Potion",
		Rarity:    "common",
		Kind:      "consumable",
		StackMax:  99,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed health potion: %v", err)
	}
}

func setTempAuthDBPath(t *testing.T) {
	t.Helper()
	base := t.TempDir()
	t.Setenv("FRACTURING_SPACE_AUTH_DB_PATH", filepath.Join(base, "auth.db"))
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial game server: %v", err)
	}
	defer conn.Close()

	healthClient := grpc_health_v1.NewHealthClient(conn)
	backoff := 100 * time.Millisecond
	for {
		callCtx, callCancel := context.WithTimeout(ctx, time.Second)
		response, err := healthClient.Check(callCtx, &grpc_health_v1.HealthCheckRequest{Service: ""})
		callCancel()
		if err == nil && response.GetStatus() == grpc_health_v1.HealthCheckResponse_SERVING {
			return
		}

		select {
		case <-ctx.Done():
			if err != nil {
				t.Fatalf("wait for gRPC health: %v", err)
			}
			t.Fatalf("wait for gRPC health: %v", ctx.Err())
		case <-time.After(backoff):
		}

		if backoff < time.Second {
			backoff *= 2
			if backoff > time.Second {
				backoff = time.Second
			}
		}
	}
}

func createAuthUser(t *testing.T, authAddr, displayName string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(
		authAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial auth server: %v", err)
	}
	defer conn.Close()

	client := authv1.NewAuthServiceClient(conn)
	email := displayName
	if !strings.Contains(email, "@") {
		email = email + "@example.com"
	}
	resp, err := client.CreateUser(ctx, &authv1.CreateUserRequest{Email: email})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	userID := resp.GetUser().GetId()
	if userID == "" {
		t.Fatal("create user: missing user id")
	}
	return userID
}
