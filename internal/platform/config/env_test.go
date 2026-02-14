package config

import (
	"strings"
	"testing"
)

type envTestConfig struct {
	Port int `env:"FRACTURING_SPACE_TEST_PORT" envDefault:"123"`
}

func TestParseEnvDefaults(t *testing.T) {
	var cfg envTestConfig

	if err := ParseEnv(&cfg); err != nil {
		t.Fatalf("parse env: %v", err)
	}
	if cfg.Port != 123 {
		t.Fatalf("expected default port 123, got %d", cfg.Port)
	}
}

func TestParseEnvError(t *testing.T) {
	var cfg envTestConfig
	t.Setenv("FRACTURING_SPACE_TEST_PORT", "not-an-int")

	err := ParseEnv(&cfg)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parse env:") {
		t.Fatalf("expected parse env prefix, got %v", err)
	}
}
