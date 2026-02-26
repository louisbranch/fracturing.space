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
	t.Setenv("FRACTURING_SPACE_USERHUB_GAME_ADDR", "custom-game:19000")

	fs := flag.NewFlagSet("userhub", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{
		"-port", "9001",
		"-social-addr", "custom-social:19001",
		"-notifications-addr", "custom-notifications:19002",
		"-cache-fresh-ttl", "20s",
		"-cache-stale-ttl", "4m",
		"-dial-timeout", "3s",
	})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Port != 9001 {
		t.Fatalf("port = %d, want 9001", cfg.Port)
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
	if cfg.GRPCDialTimeout != 3*time.Second {
		t.Fatalf("dial_timeout = %s, want %s", cfg.GRPCDialTimeout, 3*time.Second)
	}
}
