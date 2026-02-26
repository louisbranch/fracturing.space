package server

import (
	"errors"
	"path/filepath"

	"github.com/louisbranch/fracturing.space/internal/platform/config"
	"github.com/louisbranch/fracturing.space/internal/platform/discovery"
)

type serverEnv struct {
	AuthAddr                                 string `env:"FRACTURING_SPACE_AUTH_ADDR"`
	SocialAddr                               string `env:"FRACTURING_SPACE_SOCIAL_ADDR"`
	EventsDBPath                             string `env:"FRACTURING_SPACE_GAME_EVENTS_DB_PATH"`
	ProjectionsDBPath                        string `env:"FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH"`
	ContentDBPath                            string `env:"FRACTURING_SPACE_GAME_CONTENT_DB_PATH"`
	DomainEnabled                            bool   `env:"FRACTURING_SPACE_GAME_DOMAIN_ENABLED"                       envDefault:"true"`
	CompatibilityAppendEnabled               bool   `env:"FRACTURING_SPACE_GAME_COMPATIBILITY_APPEND_ENABLED"         envDefault:"false"`
	ProjectionApplyOutboxEnabled             bool   `env:"FRACTURING_SPACE_GAME_PROJECTION_APPLY_OUTBOX_ENABLED"     envDefault:"false"`
	ProjectionApplyOutboxShadowWorkerEnabled bool   `env:"FRACTURING_SPACE_GAME_PROJECTION_APPLY_OUTBOX_SHADOW_WORKER_ENABLED" envDefault:"false"`
	ProjectionApplyOutboxWorkerEnabled       bool   `env:"FRACTURING_SPACE_GAME_PROJECTION_APPLY_OUTBOX_WORKER_ENABLED" envDefault:"false"`
}

const (
	projectionApplyModeInlineApplyOnly = "inline_apply_only"
	projectionApplyModeOutboxApplyOnly = "outbox_apply_only"
	projectionApplyModeShadowOnly      = "shadow_only"
)

func loadServerEnv() serverEnv {
	var cfg serverEnv
	_ = config.ParseEnv(&cfg)
	cfg.AuthAddr = discovery.OrDefaultGRPCAddr(cfg.AuthAddr, discovery.ServiceAuth)
	cfg.SocialAddr = discovery.OrDefaultGRPCAddr(cfg.SocialAddr, discovery.ServiceSocial)
	if cfg.EventsDBPath == "" {
		cfg.EventsDBPath = filepath.Join("data", "game-events.db")
	}
	if cfg.ProjectionsDBPath == "" {
		cfg.ProjectionsDBPath = filepath.Join("data", "game-projections.db")
	}
	if cfg.ContentDBPath == "" {
		cfg.ContentDBPath = filepath.Join("data", "game-content.db")
	}
	return cfg
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
	return false, false, projectionApplyModeInlineApplyOnly, nil
}
