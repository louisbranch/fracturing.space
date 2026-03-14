package userhub

import (
	"flag"
	"testing"
	"time"
)

func TestParseConfigDefaults(t *testing.T) {
	fs := flag.NewFlagSet("userhub", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Port != 8092 {
		t.Fatalf("port = %d, want 8092", cfg.Port)
	}
	if cfg.AuthAddr != "auth:8083" {
		t.Fatalf("auth_addr = %q, want %q", cfg.AuthAddr, "auth:8083")
	}
	if cfg.GameAddr != "game:8082" {
		t.Fatalf("game_addr = %q, want %q", cfg.GameAddr, "game:8082")
	}
	if cfg.SocialAddr != "social:8090" {
		t.Fatalf("social_addr = %q, want %q", cfg.SocialAddr, "social:8090")
	}
	if cfg.NotificationsAddr != "notifications:8088" {
		t.Fatalf("notifications_addr = %q, want %q", cfg.NotificationsAddr, "notifications:8088")
	}
	if cfg.CacheFreshTTL != 15*time.Second {
		t.Fatalf("cache_fresh_ttl = %s, want %s", cfg.CacheFreshTTL, 15*time.Second)
	}
	if cfg.CacheStaleTTL != 2*time.Minute {
		t.Fatalf("cache_stale_ttl = %s, want %s", cfg.CacheStaleTTL, 2*time.Minute)
	}
}

func TestParseConfigOverrides(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_USERHUB_PORT", "9000")
	t.Setenv("FRACTURING_SPACE_USERHUB_AUTH_ADDR", "custom-auth:19003")
	t.Setenv("FRACTURING_SPACE_USERHUB_GAME_ADDR", "custom-game:19000")

	fs := flag.NewFlagSet("userhub", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{
		"-port", "9001",
		"-auth-addr", "custom-auth-flag:19004",
		"-social-addr", "custom-social:19001",
		"-notifications-addr", "custom-notifications:19002",
		"-cache-fresh-ttl", "20s",
		"-cache-stale-ttl", "4m",
	})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Port != 9001 {
		t.Fatalf("port = %d, want 9001", cfg.Port)
	}
	if cfg.AuthAddr != "custom-auth-flag:19004" {
		t.Fatalf("auth_addr = %q, want %q", cfg.AuthAddr, "custom-auth-flag:19004")
	}
	if cfg.GameAddr != "custom-game:19000" {
		t.Fatalf("game_addr = %q, want %q", cfg.GameAddr, "custom-game:19000")
	}
	if cfg.SocialAddr != "custom-social:19001" {
		t.Fatalf("social_addr = %q, want %q", cfg.SocialAddr, "custom-social:19001")
	}
	if cfg.NotificationsAddr != "custom-notifications:19002" {
		t.Fatalf("notifications_addr = %q, want %q", cfg.NotificationsAddr, "custom-notifications:19002")
	}
	if cfg.CacheFreshTTL != 20*time.Second {
		t.Fatalf("cache_fresh_ttl = %s, want %s", cfg.CacheFreshTTL, 20*time.Second)
	}
	if cfg.CacheStaleTTL != 4*time.Minute {
		t.Fatalf("cache_stale_ttl = %s, want %s", cfg.CacheStaleTTL, 4*time.Minute)
	}
}

func TestParseConfigFallsBackToGlobalAuthAddr(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AUTH_ADDR", "localhost:18083")

	fs := flag.NewFlagSet("userhub", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.AuthAddr != "localhost:18083" {
		t.Fatalf("auth_addr = %q, want %q", cfg.AuthAddr, "localhost:18083")
	}
}
