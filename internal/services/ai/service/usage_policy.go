package service

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

const maxPageSize = 50

// AgentBindingUsageReader reads the number of active campaign bindings for one
// AI agent. A missing game-side collaborator degrades to zero so read paths can
// remain available when campaign usage is optional.
type AgentBindingUsageReader struct {
	campaignUsageReader CampaignUsageReader
}

// NewAgentBindingUsageReader builds a read-only active-campaign usage reader.
func NewAgentBindingUsageReader(campaignUsageReader CampaignUsageReader) *AgentBindingUsageReader {
	return &AgentBindingUsageReader{campaignUsageReader: campaignUsageReader}
}

// ActiveCampaignCount returns the number of DRAFT/ACTIVE campaigns bound to
// one AI agent. Returns zero when the game-owned usage reader is unavailable.
func (r *AgentBindingUsageReader) ActiveCampaignCount(ctx context.Context, agentID string) (int32, error) {
	if agentID == "" {
		return 0, Errorf(ErrKindInvalidArgument, "agent_id is required")
	}
	if r == nil || r.campaignUsageReader == nil {
		return 0, nil
	}
	count, err := r.campaignUsageReader.ActiveCampaignCount(ctx, agentID)
	if err != nil {
		return 0, Wrapf(ErrKindInternal, err, "get campaign ai binding usage")
	}
	return count, nil
}

// AuthReferenceUsageReader maps credential/provider-grant usage through owned
// agents plus active campaign bindings.
type AuthReferenceUsageReader struct {
	agentStore              storage.AgentStore
	agentBindingUsageReader *AgentBindingUsageReader
}

// NewAuthReferenceUsageReader builds an auth-reference usage reader.
func NewAuthReferenceUsageReader(agentStore storage.AgentStore, agentBindingUsageReader *AgentBindingUsageReader) *AuthReferenceUsageReader {
	return &AuthReferenceUsageReader{
		agentStore:              agentStore,
		agentBindingUsageReader: agentBindingUsageReader,
	}
}

// CredentialHasActiveCampaignUsage reports whether any owned agent using the
// credential is still bound to an active campaign.
func (r *AuthReferenceUsageReader) CredentialHasActiveCampaignUsage(ctx context.Context, ownerUserID, credentialID string) (bool, error) {
	if credentialID == "" {
		return false, Errorf(ErrKindInvalidArgument, "credential_id is required")
	}
	return r.anyMatchingAgentHasActiveCampaignUsage(ctx, ownerUserID, func(a agent.Agent) bool {
		return a.AuthReference.CredentialID() == credentialID
	})
}

// ProviderGrantHasActiveCampaignUsage reports whether any owned agent using the
// provider grant is still bound to an active campaign.
func (r *AuthReferenceUsageReader) ProviderGrantHasActiveCampaignUsage(ctx context.Context, ownerUserID, providerGrantID string) (bool, error) {
	if providerGrantID == "" {
		return false, Errorf(ErrKindInvalidArgument, "provider_grant_id is required")
	}
	return r.anyMatchingAgentHasActiveCampaignUsage(ctx, ownerUserID, func(a agent.Agent) bool {
		return a.AuthReference.ProviderGrantID() == providerGrantID
	})
}

func (r *AuthReferenceUsageReader) anyMatchingAgentHasActiveCampaignUsage(
	ctx context.Context,
	ownerUserID string,
	match func(agent.Agent) bool,
) (bool, error) {
	if r == nil || r.agentStore == nil {
		return false, Errorf(ErrKindInternal, "agent store is not configured")
	}
	if r.agentBindingUsageReader == nil {
		return false, Errorf(ErrKindInternal, "agent binding usage reader is not configured")
	}
	if match == nil {
		return false, nil
	}

	pageToken := ""
	for {
		page, err := r.agentStore.ListAgentsByOwner(ctx, ownerUserID, maxPageSize, pageToken)
		if err != nil {
			return false, Wrapf(ErrKindInternal, err, "list agents")
		}
		for _, record := range page.Agents {
			if !match(record) {
				continue
			}
			count, err := r.agentBindingUsageReader.ActiveCampaignCount(ctx, record.ID)
			if err != nil {
				return false, err
			}
			if count > 0 {
				return true, nil
			}
		}

		if page.NextPageToken == "" || page.NextPageToken == pageToken {
			return false, nil
		}
		pageToken = page.NextPageToken
	}
}

// UsagePolicy converts usage reads into mutation-blocking precondition errors.
type UsagePolicy struct {
	agentBindingUsageReader  *AgentBindingUsageReader
	authReferenceUsageReader *AuthReferenceUsageReader
}

// UsagePolicyConfig declares dependencies for mutation usage policy.
type UsagePolicyConfig struct {
	AgentBindingUsageReader  *AgentBindingUsageReader
	AuthReferenceUsageReader *AuthReferenceUsageReader
}

// NewUsagePolicy builds a thin policy object from explicit read seams.
func NewUsagePolicy(cfg UsagePolicyConfig) *UsagePolicy {
	return &UsagePolicy{
		agentBindingUsageReader:  cfg.AgentBindingUsageReader,
		authReferenceUsageReader: cfg.AuthReferenceUsageReader,
	}
}

// EnsureCredentialNotBoundToActiveCampaigns blocks revocation while any owned
// agent using the credential is still bound to a DRAFT or ACTIVE campaign.
func (p *UsagePolicy) EnsureCredentialNotBoundToActiveCampaigns(ctx context.Context, ownerUserID, credentialID string) error {
	if p == nil {
		return nil
	}
	if p.authReferenceUsageReader == nil {
		return Errorf(ErrKindInternal, "auth reference usage reader is not configured")
	}
	inUse, err := p.authReferenceUsageReader.CredentialHasActiveCampaignUsage(ctx, ownerUserID, credentialID)
	if err != nil {
		return err
	}
	if inUse {
		return Errorf(ErrKindFailedPrecondition, "credential is in use by active campaigns")
	}
	return nil
}

// EnsureProviderGrantNotBoundToActiveCampaigns blocks revocation while any
// owned agent using the provider grant is bound to a DRAFT or ACTIVE campaign.
func (p *UsagePolicy) EnsureProviderGrantNotBoundToActiveCampaigns(ctx context.Context, ownerUserID, providerGrantID string) error {
	if p == nil {
		return nil
	}
	if p.authReferenceUsageReader == nil {
		return Errorf(ErrKindInternal, "auth reference usage reader is not configured")
	}
	inUse, err := p.authReferenceUsageReader.ProviderGrantHasActiveCampaignUsage(ctx, ownerUserID, providerGrantID)
	if err != nil {
		return err
	}
	if inUse {
		return Errorf(ErrKindFailedPrecondition, "provider grant is in use by active campaigns")
	}
	return nil
}

// EnsureAgentNotBoundToActiveCampaigns blocks deletion/update while the agent
// is bound to a DRAFT or ACTIVE campaign.
func (p *UsagePolicy) EnsureAgentNotBoundToActiveCampaigns(ctx context.Context, agentID string) error {
	if p == nil {
		return nil
	}
	if p.agentBindingUsageReader == nil {
		return Errorf(ErrKindInternal, "agent binding usage reader is not configured")
	}
	count, err := p.agentBindingUsageReader.ActiveCampaignCount(ctx, agentID)
	if err != nil {
		return err
	}
	if count > 0 {
		return Errorf(ErrKindFailedPrecondition, "agent is bound to active campaigns")
	}
	return nil
}
