package service

import (
	"context"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

const maxPageSize = 50

// UsageGuard checks whether auth references (credentials, provider grants,
// agents) are still bound to active campaigns, blocking revocation or deletion
// when they are.
type UsageGuard struct {
	agentStore           storage.AgentStore
	gameCampaignAIClient gamev1.CampaignAIServiceClient
}

// NewUsageGuard creates a usage guard from an agent store and the game service
// campaign AI client. Both may be nil; a nil guard is safe but permits all ops.
func NewUsageGuard(agentStore storage.AgentStore, gameCampaignAIClient gamev1.CampaignAIServiceClient) *UsageGuard {
	return &UsageGuard{
		agentStore:           agentStore,
		gameCampaignAIClient: gameCampaignAIClient,
	}
}

// EnsureCredentialNotBoundToActiveCampaigns blocks revocation while any owned
// agent using the credential is still bound to a DRAFT or ACTIVE campaign.
func (g *UsageGuard) EnsureCredentialNotBoundToActiveCampaigns(ctx context.Context, ownerUserID, credentialID string) error {
	if credentialID == "" {
		return Errorf(ErrKindInvalidArgument, "credential_id is required")
	}
	return g.ensureAuthReferenceNotBound(ctx, ownerUserID, func(a agent.Agent) bool {
		return a.AuthReference.CredentialID() == credentialID
	}, "credential is in use by active campaigns")
}

// EnsureProviderGrantNotBoundToActiveCampaigns blocks revocation while any
// owned agent using the provider grant is bound to a DRAFT or ACTIVE campaign.
func (g *UsageGuard) EnsureProviderGrantNotBoundToActiveCampaigns(ctx context.Context, ownerUserID, providerGrantID string) error {
	if providerGrantID == "" {
		return Errorf(ErrKindInvalidArgument, "provider_grant_id is required")
	}
	return g.ensureAuthReferenceNotBound(ctx, ownerUserID, func(a agent.Agent) bool {
		return a.AuthReference.ProviderGrantID() == providerGrantID
	}, "provider grant is in use by active campaigns")
}

// EnsureAgentNotBoundToActiveCampaigns blocks deletion/update while the agent
// is bound to a DRAFT or ACTIVE campaign.
func (g *UsageGuard) EnsureAgentNotBoundToActiveCampaigns(ctx context.Context, agentID string) error {
	if agentID == "" {
		return Errorf(ErrKindInvalidArgument, "agent_id is required")
	}
	count, err := g.ActiveCampaignCount(ctx, agentID)
	if err != nil {
		return err
	}
	if count > 0 {
		return Errorf(ErrKindFailedPrecondition, "agent is bound to active campaigns")
	}
	return nil
}

// ActiveCampaignCount returns the number of DRAFT/ACTIVE campaigns bound to
// one AI agent. Returns zero when the game client is unavailable.
func (g *UsageGuard) ActiveCampaignCount(ctx context.Context, agentID string) (int32, error) {
	if agentID == "" {
		return 0, Errorf(ErrKindInvalidArgument, "agent_id is required")
	}
	if g == nil || g.gameCampaignAIClient == nil {
		return 0, nil
	}
	usage, err := g.gameCampaignAIClient.GetCampaignAIBindingUsage(ctx, &gamev1.GetCampaignAIBindingUsageRequest{
		AiAgentId: agentID,
	})
	if err != nil {
		return 0, Wrapf(ErrKindInternal, err, "get campaign ai binding usage")
	}
	return usage.GetActiveCampaignCount(), nil
}

func (g *UsageGuard) ensureAuthReferenceNotBound(
	ctx context.Context,
	ownerUserID string,
	match func(agent.Agent) bool,
	message string,
) error {
	if g == nil || g.agentStore == nil {
		return Errorf(ErrKindInternal, "agent store is not configured")
	}
	if match == nil {
		return nil
	}

	pageToken := ""
	for {
		page, err := g.agentStore.ListAgentsByOwner(ctx, ownerUserID, maxPageSize, pageToken)
		if err != nil {
			return Wrapf(ErrKindInternal, err, "list agents")
		}
		for _, record := range page.Agents {
			if !match(record) {
				continue
			}
			if err := g.EnsureAgentNotBoundToActiveCampaigns(ctx, record.ID); err != nil {
				// Replace generic message with the caller-supplied one.
				return Errorf(ErrorKindOf(err), "%s", message)
			}
		}

		if page.NextPageToken == "" || page.NextPageToken == pageToken {
			return nil
		}
		pageToken = page.NextPageToken
	}
}
