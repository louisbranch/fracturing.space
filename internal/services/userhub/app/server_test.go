package app

import (
	"context"
	"strings"
	"testing"
)

func TestRunRequiresGameAddress(t *testing.T) {
	t.Parallel()

	err := Run(context.Background(), RuntimeConfig{
		SocialAddr:        "social:8090",
		NotificationsAddr: "notifications:8088",
	})
	if err == nil || !strings.Contains(err.Error(), "game address is required") {
		t.Fatalf("Run error = %v, want game address validation", err)
	}
}

func TestRunRequiresSocialAddress(t *testing.T) {
	t.Parallel()

	err := Run(context.Background(), RuntimeConfig{
		GameAddr:          "game:8082",
		NotificationsAddr: "notifications:8088",
	})
	if err == nil || !strings.Contains(err.Error(), "social address is required") {
		t.Fatalf("Run error = %v, want social address validation", err)
	}
}

func TestRunRequiresNotificationsAddress(t *testing.T) {
	t.Parallel()

	err := Run(context.Background(), RuntimeConfig{
		GameAddr:   "game:8082",
		SocialAddr: "social:8090",
	})
	if err == nil || !strings.Contains(err.Error(), "notifications address is required") {
		t.Fatalf("Run error = %v, want notifications address validation", err)
	}
}
