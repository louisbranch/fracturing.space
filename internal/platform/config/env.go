package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// ParseEnv loads configuration from environment variables.
func ParseEnv(target any) error {
	if err := env.Parse(target); err != nil {
		return fmt.Errorf("parse env: %w", err)
	}
	return nil
}
