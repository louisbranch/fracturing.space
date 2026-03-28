package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/test/mock/aifakes"
)

func TestAgentBindingUsageReaderReturnsZeroWhenUnavailable(t *testing.T) {
	reader := NewAgentBindingUsageReader(nil)

	count, err := reader.ActiveCampaignCount(context.Background(), "agent-1")
	if err != nil {
		t.Fatalf("ActiveCampaignCount: %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
}

func TestAuthReferenceUsageReaderDetectsCredentialUsage(t *testing.T) {
	agentStore := aifakes.NewAgentStore()
	now := time.Date(2026, 3, 23, 21, 0, 0, 0, time.UTC)
	agentStore.Agents["agent-1"] = agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "user-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-5-mini",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	reader := NewAuthReferenceUsageReader(agentStore, NewAgentBindingUsageReader(&fakeCampaignUsageReader{
		usageByAgent: map[string]int32{"agent-1": 2},
	}))

	inUse, err := reader.CredentialHasActiveCampaignUsage(context.Background(), "user-1", "cred-1")
	if err != nil {
		t.Fatalf("CredentialHasActiveCampaignUsage: %v", err)
	}
	if !inUse {
		t.Fatal("expected credential to be reported as in use")
	}
}

func TestUsagePolicyEnsuresAgentNotBound(t *testing.T) {
	policy := NewUsagePolicy(UsagePolicyConfig{
		AgentBindingUsageReader: NewAgentBindingUsageReader(&fakeCampaignUsageReader{
			usageByAgent: map[string]int32{"agent-1": 1},
		}),
	})

	err := policy.EnsureAgentNotBoundToActiveCampaigns(context.Background(), "agent-1")
	if got := ErrorKindOf(err); got != ErrKindFailedPrecondition {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindFailedPrecondition)
	}
}

func TestUsagePolicyWrapsUsageReadFailures(t *testing.T) {
	policy := NewUsagePolicy(UsagePolicyConfig{
		AgentBindingUsageReader: NewAgentBindingUsageReader(&fakeCampaignUsageReader{
			err: errors.New("usage unavailable"),
		}),
	})

	err := policy.EnsureAgentNotBoundToActiveCampaigns(context.Background(), "agent-1")
	if got := ErrorKindOf(err); got != ErrKindInternal {
		t.Fatalf("ErrorKindOf(err) = %v, want %v", got, ErrKindInternal)
	}
}
