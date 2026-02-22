package cmd

import (
	"context"
	"flag"
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
