package otel_test

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/platform/otel"
)

func TestSetup_NoopWhenEndpointEmpty(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_OTEL_ENDPOINT", "")
	t.Setenv("FRACTURING_SPACE_OTEL_ENABLED", "")

	shutdown, err := otel.Setup(context.Background(), "test-service")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}
}

func TestSetup_NoopWhenExplicitlyDisabled(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_OTEL_ENDPOINT", "http://localhost:4318")
	t.Setenv("FRACTURING_SPACE_OTEL_ENABLED", "false")

	shutdown, err := otel.Setup(context.Background(), "test-service")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}
}

func TestSetup_CreatesProviderWhenEndpointSet(t *testing.T) {
	// Use a non-routable address so no actual export happens.
	t.Setenv("FRACTURING_SPACE_OTEL_ENDPOINT", "http://192.0.2.1:4318")
	t.Setenv("FRACTURING_SPACE_OTEL_ENABLED", "")

	shutdown, err := otel.Setup(context.Background(), "test-service")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Shutdown should flush cleanly even though the endpoint is unreachable.
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}
}

func TestSetup_ShutdownFlushesCleanly(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_OTEL_ENDPOINT", "http://192.0.2.1:4318")
	t.Setenv("FRACTURING_SPACE_OTEL_ENABLED", "")

	shutdown, err := otel.Setup(context.Background(), "flush-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}
}

func TestSetup_NoopShutdownIgnoresCancelledContext(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_OTEL_ENDPOINT", "")
	t.Setenv("FRACTURING_SPACE_OTEL_ENABLED", "")

	shutdown, err := otel.Setup(context.Background(), "noop-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := shutdown(ctx); err != nil {
		t.Fatalf("noop shutdown should not error: %v", err)
	}
}
