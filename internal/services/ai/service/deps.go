package service

import (
	"context"
	"fmt"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providercatalog"
)

// Clock returns the current time. Used as a dependency for deterministic tests.
type Clock = func() time.Time

// IDGenerator returns a new unique identifier. Used as a dependency for
// deterministic tests.
type IDGenerator = func() (string, error)

// CampaignUsageReader reads game-owned usage counts for one AI agent.
type CampaignUsageReader interface {
	ActiveCampaignCount(context.Context, string) (int32, error)
}

// CampaignAuthStateReader reads the game-owned AI runtime state for one
// campaign. The response stays in its game-owned shape because AI treats it as
// an integration contract, not as AI domain state.
type CampaignAuthStateReader interface {
	CampaignAuthState(context.Context, string) (*gamev1.GetCampaignAIAuthStateResponse, error)
}

func withDefaultClock(clock Clock) Clock {
	if clock != nil {
		return clock
	}
	return time.Now
}

func withDefaultIDGenerator(gen IDGenerator) IDGenerator {
	if gen != nil {
		return gen
	}
	return id.NewID
}

func RequireProviderRegistry(registry *providercatalog.Registry, caller string) error {
	if registry == nil {
		return fmt.Errorf("ai: %s: provider registry is required", caller)
	}
	return nil
}
