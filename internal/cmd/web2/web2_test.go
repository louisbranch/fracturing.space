package web2

import (
	"flag"
	"testing"
)

func TestParseConfigDefaults(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web2", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if cfg.HTTPAddr != "localhost:8092" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, "localhost:8092")
	}
	if cfg.GameAddr != "game:8082" {
		t.Fatalf("GameAddr = %q, want %q", cfg.GameAddr, "game:8082")
	}
	if cfg.ChatHTTPAddr != "localhost:8086" {
		t.Fatalf("ChatHTTPAddr = %q, want %q", cfg.ChatHTTPAddr, "localhost:8086")
	}
}

func TestParseConfigOverrideHTTPAddr(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web2", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-http-addr", "127.0.0.1:9002"})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if cfg.HTTPAddr != "127.0.0.1:9002" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, "127.0.0.1:9002")
	}
}

func TestParseConfigOverrideGameAddr(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web2", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-game-addr", "127.0.0.1:19082"})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if cfg.GameAddr != "127.0.0.1:19082" {
		t.Fatalf("GameAddr = %q, want %q", cfg.GameAddr, "127.0.0.1:19082")
	}
}

func TestParseConfigOverrideChatHTTPAddr(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web2", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-chat-http-addr", "127.0.0.1:18086"})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if cfg.ChatHTTPAddr != "127.0.0.1:18086" {
		t.Fatalf("ChatHTTPAddr = %q, want %q", cfg.ChatHTTPAddr, "127.0.0.1:18086")
	}
}

func TestParseConfigDefaultsDisableExperimentalModules(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web2", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if cfg.EnableExperimentalModules {
		t.Fatalf("EnableExperimentalModules = %t, want false", cfg.EnableExperimentalModules)
	}
}

func TestParseConfigOverrideExperimentalModules(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web2", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-enable-experimental-modules"})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if !cfg.EnableExperimentalModules {
		t.Fatalf("EnableExperimentalModules = %t, want true", cfg.EnableExperimentalModules)
	}
}
