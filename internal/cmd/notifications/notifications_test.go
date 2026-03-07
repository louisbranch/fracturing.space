package notifications

import (
	"context"
	"errors"
	"flag"
	"testing"

	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
)

func TestParseConfigDefaults(t *testing.T) {
	fs := flag.NewFlagSet("notifications", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Port != 8088 {
		t.Fatalf("expected default port 8088, got %d", cfg.Port)
	}
	if cfg.Mode != string(runtimeModeAPI) {
		t.Fatalf("expected default mode %q, got %q", runtimeModeAPI, cfg.Mode)
	}
}

func TestParseConfigOverrides(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_NOTIFICATIONS_PORT", "9090")
	t.Setenv("FRACTURING_SPACE_NOTIFICATIONS_MODE", "api")

	fs := flag.NewFlagSet("notifications", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-port", "9091", "-mode", "worker"})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Port != 9091 {
		t.Fatalf("expected port override 9091, got %d", cfg.Port)
	}
	if cfg.Mode != string(runtimeModeWorker) {
		t.Fatalf("expected mode %q, got %q", runtimeModeWorker, cfg.Mode)
	}
}

func TestParseConfigRejectsInvalidMode(t *testing.T) {
	fs := flag.NewFlagSet("notifications", flag.ContinueOnError)
	if _, err := ParseConfig(fs, []string{"-mode", "invalid"}); err == nil {
		t.Fatal("expected invalid mode error")
	}
}

func TestRunModeAPIDispatch(t *testing.T) {
	restore := stubRuntimeHooks(t)

	var gotCaps []entrypoint.CapabilityRegistration
	var apiCalled bool
	runWithTelemetry = func(ctx context.Context, service string, run func(context.Context) error) error {
		if service != entrypoint.ServiceNotifications {
			t.Fatalf("service = %q, want %q", service, entrypoint.ServiceNotifications)
		}
		return run(ctx)
	}
	startStatusReporter = func(_ context.Context, _ string, _ string, caps ...entrypoint.CapabilityRegistration) func() {
		gotCaps = append(gotCaps, caps...)
		return func() {}
	}
	runNotificationsAPI = func(context.Context, int) error {
		apiCalled = true
		return nil
	}
	runNotificationsWorker = func(context.Context) error {
		t.Fatal("worker should not run in api mode")
		return nil
	}

	err := Run(context.Background(), Config{
		Port:       8088,
		StatusAddr: "127.0.0.1:8087",
		Mode:       "api",
	})
	restore()
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !apiCalled {
		t.Fatal("expected api runtime call")
	}
	if len(gotCaps) != 1 || gotCaps[0].Name != "notifications.inbox" {
		t.Fatalf("unexpected capabilities: %+v", gotCaps)
	}
}

func TestRunModeWorkerDispatch(t *testing.T) {
	restore := stubRuntimeHooks(t)

	var gotCaps []entrypoint.CapabilityRegistration
	var workerCalled bool
	runWithTelemetry = func(ctx context.Context, service string, run func(context.Context) error) error {
		return run(ctx)
	}
	startStatusReporter = func(_ context.Context, _ string, _ string, caps ...entrypoint.CapabilityRegistration) func() {
		gotCaps = append(gotCaps, caps...)
		return func() {}
	}
	runNotificationsAPI = func(context.Context, int) error {
		t.Fatal("api should not run in worker mode")
		return nil
	}
	runNotificationsWorker = func(context.Context) error {
		workerCalled = true
		return nil
	}

	err := Run(context.Background(), Config{
		Port:       8088,
		StatusAddr: "127.0.0.1:8087",
		Mode:       "worker",
	})
	restore()
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !workerCalled {
		t.Fatal("expected worker runtime call")
	}
	if len(gotCaps) != 1 || gotCaps[0].Name != "notifications.email.delivery-worker" {
		t.Fatalf("unexpected capabilities: %+v", gotCaps)
	}
}

func TestRunModeAllDispatch(t *testing.T) {
	restore := stubRuntimeHooks(t)

	apiCalled := 0
	workerCalled := 0
	runWithTelemetry = func(ctx context.Context, service string, run func(context.Context) error) error {
		return run(ctx)
	}
	startStatusReporter = func(context.Context, string, string, ...entrypoint.CapabilityRegistration) func() {
		return func() {}
	}
	runNotificationsAPI = func(context.Context, int) error {
		apiCalled++
		return nil
	}
	runNotificationsWorker = func(context.Context) error {
		workerCalled++
		return nil
	}

	err := Run(context.Background(), Config{
		Port:       8088,
		StatusAddr: "127.0.0.1:8087",
		Mode:       "all",
	})
	restore()
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if apiCalled != 1 {
		t.Fatalf("apiCalled = %d, want 1", apiCalled)
	}
	if workerCalled != 1 {
		t.Fatalf("workerCalled = %d, want 1", workerCalled)
	}
}

func TestRunPropagatesRuntimeError(t *testing.T) {
	restore := stubRuntimeHooks(t)

	boom := errors.New("boom")
	runWithTelemetry = func(ctx context.Context, service string, run func(context.Context) error) error {
		return run(ctx)
	}
	startStatusReporter = func(context.Context, string, string, ...entrypoint.CapabilityRegistration) func() {
		return func() {}
	}
	runNotificationsAPI = func(context.Context, int) error {
		return boom
	}

	err := Run(context.Background(), Config{
		Port:       8088,
		StatusAddr: "127.0.0.1:8087",
		Mode:       "api",
	})
	restore()
	if !errors.Is(err, boom) {
		t.Fatalf("expected run error propagation, got %v", err)
	}
}

func stubRuntimeHooks(t *testing.T) func() {
	t.Helper()
	prevRunWithTelemetry := runWithTelemetry
	prevStartStatusReporter := startStatusReporter
	prevRunNotificationsAPI := runNotificationsAPI
	prevRunNotificationsWorker := runNotificationsWorker
	return func() {
		runWithTelemetry = prevRunWithTelemetry
		startStatusReporter = prevStartStatusReporter
		runNotificationsAPI = prevRunNotificationsAPI
		runNotificationsWorker = prevRunNotificationsWorker
	}
}
