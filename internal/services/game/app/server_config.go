package server

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/config"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
)

type serverEnv struct {
	AuthAddr                                 string        `env:"FRACTURING_SPACE_AUTH_ADDR"`
	SocialAddr                               string        `env:"FRACTURING_SPACE_SOCIAL_ADDR"`
	AIAddr                                   string        `env:"FRACTURING_SPACE_AI_ADDR"`
	StatusAddr                               string        `env:"FRACTURING_SPACE_STATUS_ADDR"`
	EventsDBPath                             string        `env:"FRACTURING_SPACE_GAME_EVENTS_DB_PATH"`
	ProjectionsDBPath                        string        `env:"FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH"`
	ContentDBPath                            string        `env:"FRACTURING_SPACE_GAME_CONTENT_DB_PATH"`
	DomainEnabled                            bool          `env:"FRACTURING_SPACE_GAME_DOMAIN_ENABLED"                       envDefault:"true"`
	ProjectionApplyOutboxEnabled             bool          `env:"FRACTURING_SPACE_GAME_PROJECTION_APPLY_OUTBOX_ENABLED"     envDefault:"false"`
	ProjectionApplyOutboxShadowWorkerEnabled bool          `env:"FRACTURING_SPACE_GAME_PROJECTION_APPLY_OUTBOX_SHADOW_WORKER_ENABLED" envDefault:"false"`
	ProjectionApplyOutboxWorkerEnabled       bool          `env:"FRACTURING_SPACE_GAME_PROJECTION_APPLY_OUTBOX_WORKER_ENABLED" envDefault:"false"`
	InternalServiceAllowlist                 string        `env:"FRACTURING_SPACE_GAME_INTERNAL_SERVICE_ALLOWLIST" envDefault:"ai,worker"`
	StartupTimeout                           time.Duration `env:"FRACTURING_SPACE_GAME_STARTUP_TIMEOUT"            envDefault:"60s"`
}

const (
	projectionApplyModeInlineApplyOnly = "inline_apply_only"
	projectionApplyModeOutboxApplyOnly = "outbox_apply_only"
	projectionApplyModeShadowOnly      = "shadow_only"
)

func loadServerEnv() (serverEnv, error) {
	var cfg serverEnv
	if err := config.ParseEnv(&cfg); err != nil {
		return serverEnv{}, fmt.Errorf("parse game server env: %w", err)
	}
	cfg.AuthAddr = serviceaddr.OrDefaultGRPCAddr(cfg.AuthAddr, serviceaddr.ServiceAuth)
	cfg.SocialAddr = serviceaddr.OrDefaultGRPCAddr(cfg.SocialAddr, serviceaddr.ServiceSocial)
	cfg.AIAddr = serviceaddr.OrDefaultGRPCAddr(cfg.AIAddr, serviceaddr.ServiceAI)
	// Status address is not defaulted — the status service is optional/advisory.
	// When unset, the reporter starts with a nil client and accumulates locally.
	if cfg.StatusAddr != "" {
		cfg.StatusAddr = serviceaddr.OrDefaultGRPCAddr(cfg.StatusAddr, serviceaddr.ServiceStatus)
	}
	if cfg.EventsDBPath == "" {
		cfg.EventsDBPath = filepath.Join("data", "game-events.db")
	}
	if cfg.ProjectionsDBPath == "" {
		cfg.ProjectionsDBPath = filepath.Join("data", "game-projections.db")
	}
	if cfg.ContentDBPath == "" {
		cfg.ContentDBPath = filepath.Join("data", "game-content.db")
	}
	if _, _, _, err := resolveProjectionApplyOutboxModes(cfg); err != nil {
		return serverEnv{}, fmt.Errorf("validate projection apply config: %w", err)
	}
	return cfg, nil
}

func resolveProjectionApplyOutboxModes(srvEnv serverEnv) (bool, bool, string, error) {
	if !srvEnv.ProjectionApplyOutboxEnabled {
		if srvEnv.ProjectionApplyOutboxWorkerEnabled {
			return false, false, "", errors.New("projection apply outbox worker requested without outbox enabled")
		}
		if srvEnv.ProjectionApplyOutboxShadowWorkerEnabled {
			return false, false, "", errors.New("projection apply outbox shadow worker requested without outbox enabled")
		}
		return false, false, projectionApplyModeInlineApplyOnly, nil
	}

	if srvEnv.ProjectionApplyOutboxWorkerEnabled && srvEnv.ProjectionApplyOutboxShadowWorkerEnabled {
		return false, false, "", errors.New("projection apply outbox cannot enable both apply and shadow workers")
	}
	if srvEnv.ProjectionApplyOutboxWorkerEnabled {
		return true, false, projectionApplyModeOutboxApplyOnly, nil
	}
	if srvEnv.ProjectionApplyOutboxShadowWorkerEnabled {
		return false, true, projectionApplyModeShadowOnly, nil
	}
	return false, false, "", errors.New("projection apply outbox enabled but no worker configured; enable either worker or shadow worker")
}
