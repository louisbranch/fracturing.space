package scenario

import (
	"flag"
	"testing"
)

func TestParseConfigDefaults(t *testing.T) {
	fs := flag.NewFlagSet("scenario", flag.ContinueOnError)

	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.GRPCAddr != "game:8082" {
		t.Fatalf("expected default grpc addr, got %q", cfg.GRPCAddr)
	}
	if !cfg.Assertions {
		t.Fatal("expected assertions to default to true")
	}
	if !cfg.ValidateComments {
		t.Fatal("expected comment validation to default to true")
	}
}
