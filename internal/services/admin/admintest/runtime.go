// Package admintest provides runtime-backed test helpers for the admin service.
package admintest

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/admin"
	"github.com/louisbranch/fracturing.space/internal/test/testkit"
)

// Runtime exposes the live HTTP surface for admin tests.
type Runtime struct {
	BaseURL  string
	HTTPAddr string
}

// StartRuntime boots an admin server against the supplied game gRPC runtime.
func StartRuntime(ctx context.Context, t *testing.T, grpcAddr string) Runtime {
	t.Helper()

	httpAddr := testkit.PickUnusedAddress(t)
	server, err := admin.NewServer(ctx, admin.Config{
		HTTPAddr: httpAddr,
		GRPCAddr: grpcAddr,
	})
	if err != nil {
		t.Fatalf("create admin server: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.ListenAndServe(ctx)
	}()

	baseURL := "http://" + httpAddr
	WaitForHealth(t, baseURL)

	t.Cleanup(func() {
		server.Close()
		select {
		case err := <-serveErr:
			if err != nil {
				t.Fatalf("admin server error: %v", err)
			}
		case <-time.After(5 * time.Second):
		}
	})

	return Runtime{
		BaseURL:  baseURL,
		HTTPAddr: httpAddr,
	}
}

// WaitForHealth waits until the admin dashboard responds with HTTP 200.
func WaitForHealth(t *testing.T, baseURL string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := &http.Client{Timeout: time.Second}
	backoff := 100 * time.Millisecond

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/app/dashboard/", nil)
		if err != nil {
			t.Fatalf("create health request: %v", err)
		}
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}

		select {
		case <-ctx.Done():
			t.Fatalf("wait for admin health: %v", ctx.Err())
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
