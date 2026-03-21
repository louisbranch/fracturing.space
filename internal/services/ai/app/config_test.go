package server

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestLoadRuntimeConfigFromEnvRequiresEncryptionKey(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", "")

	_, err := loadRuntimeConfigFromEnv()
	if err == nil || !strings.Contains(err.Error(), "FRACTURING_SPACE_AI_ENCRYPTION_KEY is required") {
		t.Fatalf("loadRuntimeConfigFromEnv() error = %v", err)
	}
}

func TestLoadRuntimeConfigFromEnvRejectsInvalidEncryptionKey(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", "not-base64")

	_, err := loadRuntimeConfigFromEnv()
	if err == nil || !strings.Contains(err.Error(), "decode encryption key") {
		t.Fatalf("loadRuntimeConfigFromEnv() error = %v", err)
	}
}

func TestLoadRuntimeConfigFromEnvRejectsInvalidOrchestrationDuration(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	t.Setenv("FRACTURING_SPACE_AI_ORCHESTRATION_TURN_TIMEOUT", "not-a-duration")

	_, err := loadRuntimeConfigFromEnv()
	if err == nil || !strings.Contains(err.Error(), "load AI runtime env") {
		t.Fatalf("loadRuntimeConfigFromEnv() error = %v", err)
	}
}

func TestLoadRuntimeConfigFromEnvRejectsNonPositiveOrchestrationBudget(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	t.Setenv("FRACTURING_SPACE_AI_ORCHESTRATION_MAX_STEPS", "0")

	_, err := loadRuntimeConfigFromEnv()
	if err == nil || !strings.Contains(err.Error(), "FRACTURING_SPACE_AI_ORCHESTRATION_MAX_STEPS must be positive") {
		t.Fatalf("loadRuntimeConfigFromEnv() error = %v", err)
	}
}

func TestLoadRuntimeConfigFromEnvLoadsOrchestrationDefaults(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))

	cfg, err := loadRuntimeConfigFromEnv()
	if err != nil {
		t.Fatalf("loadRuntimeConfigFromEnv() error = %v", err)
	}
	if cfg.OrchestrationTurnTimeout != 2*time.Minute {
		t.Fatalf("OrchestrationTurnTimeout = %v", cfg.OrchestrationTurnTimeout)
	}
	if cfg.OrchestrationMaxSteps != 8 {
		t.Fatalf("OrchestrationMaxSteps = %d", cfg.OrchestrationMaxSteps)
	}
	if cfg.ToolResultMaxBytes != 32768 {
		t.Fatalf("ToolResultMaxBytes = %d", cfg.ToolResultMaxBytes)
	}
}
