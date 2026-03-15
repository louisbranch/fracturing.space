package chat

import (
	"flag"
	"testing"
)

func TestParseConfigDefaults(t *testing.T) {
	fs := flag.NewFlagSet("chat", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.HTTPAddr != ":8086" {
		t.Fatalf("expected default http addr, got %q", cfg.HTTPAddr)
	}
	if cfg.GameAddr != "game:8082" {
		t.Fatalf("expected default game addr, got %q", cfg.GameAddr)
	}
	if cfg.AuthAddr != "auth:8083" {
		t.Fatalf("expected default auth addr, got %q", cfg.AuthAddr)
	}
}

func TestParseConfigOverrides(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_CHAT_HTTP_ADDR", "env-chat")
	t.Setenv("FRACTURING_SPACE_GAME_ADDR", "env-game")
	t.Setenv("FRACTURING_SPACE_AUTH_ADDR", "env-auth")

	fs := flag.NewFlagSet("chat", flag.ContinueOnError)
	args := []string{
		"-http-addr", "flag-chat",
		"-game-addr", "flag-game",
		"-auth-addr", "flag-auth",
	}
	cfg, err := ParseConfig(fs, args)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.HTTPAddr != "flag-chat" {
		t.Fatalf("expected flag http addr, got %q", cfg.HTTPAddr)
	}
	if cfg.GameAddr != "flag-game" {
		t.Fatalf("expected flag game addr, got %q", cfg.GameAddr)
	}
	if cfg.AuthAddr != "flag-auth" {
		t.Fatalf("expected flag auth addr, got %q", cfg.AuthAddr)
	}
}
