//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/app/server"
	"github.com/louisbranch/fracturing.space/internal/mcp/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

// integrationSuite shares resources across integration subtests.
type integrationSuite struct {
	client *mcp.ClientSession
}

// integrationTimeout returns the default timeout for integration calls.
func integrationTimeout() time.Duration {
	return 10 * time.Second
}

// startGRPCServer boots the gRPC server and returns its address and shutdown function.
func startGRPCServer(t *testing.T) (string, func()) {
	t.Helper()

	setTempDBPath(t)

	ctx, cancel := context.WithCancel(context.Background())
	grpcServer, err := server.New(0)
	if err != nil {
		cancel()
		t.Fatalf("new gRPC server: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(ctx)
	}()

	addr := normalizeAddress(t, grpcServer.Addr())
	waitForGRPCHealth(t, addr)
	stop := func() {
		cancel()
		select {
		case err := <-serveErr:
			if err != nil {
				t.Fatalf("gRPC server error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for gRPC server to stop")
		}
	}

	return addr, stop
}

// startMCPClient boots the MCP stdio process and returns a client session and shutdown function.
func startMCPClient(t *testing.T, grpcAddr string) (*mcp.ClientSession, func()) {
	t.Helper()

	cmd := exec.Command("go", "run", "./cmd/mcp")
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), fmt.Sprintf("DUALITY_GRPC_ADDR=%s", grpcAddr))
	cmd.Stderr = os.Stderr

	transport := &mcp.CommandTransport{Command: cmd}
	client := mcp.NewClient(&mcp.Implementation{Name: "integration-client", Version: "dev"}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	clientSession, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("connect MCP client: %v", err)
	}

	closeClient := func() {
		closeErr := clientSession.Close()
		if closeErr != nil {
			t.Fatalf("close MCP client: %v", closeErr)
		}
	}

	return clientSession, closeClient
}

// decodeStructuredContent decodes structured MCP content into the target type.
func decodeStructuredContent[T any](t *testing.T, value any) T {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	var output T
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("unmarshal structured content: %v", err)
	}
	return output
}

// parseCampaignListPayload decodes a campaign list JSON payload.
func parseCampaignListPayload(t *testing.T, raw string) domain.CampaignListPayload {
	t.Helper()

	var payload domain.CampaignListPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal campaign list payload: %v", err)
	}
	return payload
}

// findCampaignByID searches for a campaign entry by ID.
func findCampaignByID(payload domain.CampaignListPayload, id string) (domain.CampaignListEntry, bool) {
	for _, campaign := range payload.Campaigns {
		if campaign.ID == id {
			return campaign, true
		}
	}
	return domain.CampaignListEntry{}, false
}

// parseRFC3339 parses an RFC3339 timestamp string.
func parseRFC3339(t *testing.T, value string) time.Time {
	t.Helper()

	if value == "" {
		t.Fatal("expected non-empty timestamp")
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse timestamp %q: %v", value, err)
	}
	return parsed
}

// setTempDBPath configures a temporary database for integration tests.
func setTempDBPath(t *testing.T) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "duality.db")
	t.Setenv("DUALITY_DB_PATH", path)
}

// repoRoot returns the repository root by walking up to go.mod.
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

// normalizeAddress maps wildcard listener hosts to localhost.
func normalizeAddress(t *testing.T, addr string) string {
	t.Helper()

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split address %q: %v", addr, err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port)
}

// waitForGRPCHealth waits for the gRPC health check to report SERVING.
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
		t.Fatalf("dial gRPC server: %v", err)
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

// intPointer returns a pointer to the provided int value.
func intPointer(value int) *int {
	return &value
}
