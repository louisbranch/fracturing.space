package cmd

import (
	"context"
	"errors"
	"flag"
	"log"
	"strings"
	"testing"
)

type testConfig struct {
	Address string `env:"CMD_TEST_ADDRESS" envDefault:"127.0.0.1:8080"`
	Mode    string `env:"CMD_TEST_MODE" envDefault:"server"`
}

func TestParseConfigReadsEnvAndFlags(t *testing.T) {
	t.Setenv("CMD_TEST_ADDRESS", "env:9000")
	t.Setenv("CMD_TEST_MODE", "env-mode")

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfgRef := testConfig{}
	if err := ParseConfig(&cfgRef); err != nil {
		t.Fatalf("load config defaults: %v", err)
	}
	fs.StringVar(&cfgRef.Address, "address", cfgRef.Address, "address")
	fs.StringVar(&cfgRef.Mode, "mode", cfgRef.Mode, "mode")

	if err := ParseArgs(fs, []string{"-address", "flag:9001"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	if cfgRef.Address != "flag:9001" {
		t.Fatalf("expected flag value for address, got %q", cfgRef.Address)
	}
	if cfgRef.Mode != "env-mode" {
		t.Fatalf("expected env default mode, got %q", cfgRef.Mode)
	}
}

func TestParseConfigFromArgsReadsEnvAndFlags(t *testing.T) {
	t.Setenv("CMD_TEST_ADDRESS", "configarg:9000")
	t.Setenv("CMD_TEST_MODE", "configarg-mode")

	cfgRef := testConfig{}
	fs := flag.NewFlagSet("configargs", flag.ContinueOnError)
	fs.StringVar(&cfgRef.Address, "address", "", "address")
	fs.StringVar(&cfgRef.Mode, "mode", "", "mode")
	if err := ParseConfigFromArgs(&cfgRef, fs, []string{"-address", "flag:9002"}); err != nil {
		t.Fatalf("parse config and args: %v", err)
	}
	if cfgRef.Address != "flag:9002" {
		t.Fatalf("expected parsed flag address, got %q", cfgRef.Address)
	}
	if cfgRef.Mode != "configarg-mode" {
		t.Fatalf("expected env default mode, got %q", cfgRef.Mode)
	}
}

func TestParseArgsRejectsNilParser(t *testing.T) {
	if err := ParseArgs(nil, []string{}); err == nil {
		t.Fatal("expected parse args to reject nil parser")
	}
}

func TestRunWithTelemetryRejectsMissingInputs(t *testing.T) {
	if err := RunWithTelemetry(nil, "", func(context.Context) error { return nil }); err == nil {
		t.Fatal("expected missing service error")
	}
	if err := RunWithTelemetry(nil, ServiceGame, nil); err == nil {
		t.Fatal("expected missing run function error")
	}
}

func TestRunServiceMainRejectsMissingInputs(t *testing.T) {
	if err := RunServiceMain(ServiceMainOptions[testConfig]{}); err == nil {
		t.Fatal("expected missing service error")
	}
	if err := RunServiceMain(ServiceMainOptions[testConfig]{
		Service: ServiceAI,
	}); err == nil {
		t.Fatal("expected missing parse config function error")
	}
	if err := RunServiceMain(ServiceMainOptions[testConfig]{
		Service:     ServiceAI,
		ParseConfig: func(*flag.FlagSet, []string) (testConfig, error) { return testConfig{}, nil },
	}); err == nil {
		t.Fatal("expected missing run function error")
	}
}

func TestRunServiceMainPassesArgsToParseAndRun(t *testing.T) {
	originalPrefix := log.Prefix()
	t.Cleanup(func() {
		log.SetPrefix(originalPrefix)
	})

	parseCalled := false
	runCalled := false
	err := RunServiceMain(ServiceMainOptions[testConfig]{
		Service: ServiceMCP,
		FlagSet: flag.NewFlagSet("test", flag.ContinueOnError),
		Args:    []string{"-addr", ":8081"},
		ParseConfig: func(fs *flag.FlagSet, args []string) (testConfig, error) {
			parseCalled = true
			if fs == nil {
				t.Fatal("expected flag set")
			}
			if len(args) != 2 || args[0] != "-addr" || args[1] != ":8081" {
				t.Fatalf("unexpected args: %v", args)
			}
			return testConfig{Address: args[1]}, nil
		},
		Run: func(ctx context.Context, cfg testConfig) error {
			runCalled = true
			if ctx == nil {
				t.Fatal("expected context")
			}
			if cfg.Address != ":8081" {
				t.Fatalf("unexpected config address: %q", cfg.Address)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("run service main: %v", err)
	}
	if !parseCalled {
		t.Fatal("expected parse to be called")
	}
	if !runCalled {
		t.Fatal("expected run to be called")
	}
	if got := log.Prefix(); got != "[MCP] " {
		t.Fatalf("expected log prefix [MCP], got %q", got)
	}
}

func TestRunServiceMainWrapsParseError(t *testing.T) {
	parseErr := errors.New("bad flags")
	err := RunServiceMain(ServiceMainOptions[testConfig]{
		Service: ServiceStatus,
		FlagSet: flag.NewFlagSet("test", flag.ContinueOnError),
		Args:    []string{},
		ParseConfig: func(*flag.FlagSet, []string) (testConfig, error) {
			return testConfig{}, parseErr
		},
		Run: func(context.Context, testConfig) error {
			t.Fatal("run should not be called")
			return nil
		},
	})
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "parse flags") {
		t.Fatalf("expected parse flags context, got: %v", err)
	}
	if !errors.Is(err, parseErr) {
		t.Fatalf("expected parse error wrapping, got: %v", err)
	}
}

func TestRunServiceMainWrapsRunError(t *testing.T) {
	runErr := errors.New("serve failed")
	err := RunServiceMain(ServiceMainOptions[testConfig]{
		Service: ServiceWorker,
		FlagSet: flag.NewFlagSet("test", flag.ContinueOnError),
		Args:    []string{},
		ParseConfig: func(*flag.FlagSet, []string) (testConfig, error) {
			return testConfig{}, nil
		},
		Run: func(context.Context, testConfig) error {
			return runErr
		},
	})
	if err == nil {
		t.Fatal("expected run error")
	}
	if !strings.Contains(err.Error(), "serve worker") {
		t.Fatalf("expected service context, got: %v", err)
	}
	if !errors.Is(err, runErr) {
		t.Fatalf("expected run error wrapping, got: %v", err)
	}
}
