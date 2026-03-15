package ai

import (
	"context"
	"strings"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type authReferenceUsageGuard struct {
	agentStore           storage.AgentStore
	gameCampaignAIClient gamev1.CampaignAIServiceClient
}

func newAuthReferenceUsageGuard(agentStore storage.AgentStore, gameCampaignAIClient gamev1.CampaignAIServiceClient) authReferenceUsageGuard {
	return authReferenceUsageGuard{
		agentStore:           agentStore,
		gameCampaignAIClient: gameCampaignAIClient,
	}
}

// ensureCredentialNotBoundToActiveCampaigns blocks revocation while any owned
// agent using the credential is still bound to a DRAFT or ACTIVE campaign.
func (s *Service) ensureCredentialNotBoundToActiveCampaigns(ctx context.Context, ownerUserID, credentialID string) error {
	return newAuthReferenceUsageGuard(s.agentStore, s.gameCampaignAIClient).ensureCredentialNotBoundToActiveCampaigns(ctx, ownerUserID, credentialID)
}

// ensureCredentialNotBoundToActiveCampaigns blocks revocation while any owned
// agent using the credential is still bound to a DRAFT or ACTIVE campaign.
func (g authReferenceUsageGuard) ensureCredentialNotBoundToActiveCampaigns(ctx context.Context, ownerUserID, credentialID string) error {
	credentialID = strings.TrimSpace(credentialID)
	if credentialID == "" {
		return status.Error(codes.InvalidArgument, "credential_id is required")
	}
	return g.ensureAuthReferenceNotBoundToActiveCampaigns(ctx, ownerUserID, func(record storage.AgentRecord) bool {
		return strings.TrimSpace(record.CredentialID) == credentialID
	}, "credential is in use by active campaigns")
}

// ensureProviderGrantNotBoundToActiveCampaigns blocks revocation while any owned
// agent using the provider grant is still bound to a DRAFT or ACTIVE campaign.
func (s *Service) ensureProviderGrantNotBoundToActiveCampaigns(ctx context.Context, ownerUserID, providerGrantID string) error {
	return newAuthReferenceUsageGuard(s.agentStore, s.gameCampaignAIClient).ensureProviderGrantNotBoundToActiveCampaigns(ctx, ownerUserID, providerGrantID)
}

// ensureProviderGrantNotBoundToActiveCampaigns blocks revocation while any owned
// agent using the provider grant is still bound to a DRAFT or ACTIVE campaign.
func (g authReferenceUsageGuard) ensureProviderGrantNotBoundToActiveCampaigns(ctx context.Context, ownerUserID, providerGrantID string) error {
	providerGrantID = strings.TrimSpace(providerGrantID)
	if providerGrantID == "" {
		return status.Error(codes.InvalidArgument, "provider_grant_id is required")
	}
	return g.ensureAuthReferenceNotBoundToActiveCampaigns(ctx, ownerUserID, func(record storage.AgentRecord) bool {
		return strings.TrimSpace(record.ProviderGrantID) == providerGrantID
	}, "provider grant is in use by active campaigns")
}

func (g authReferenceUsageGuard) ensureAuthReferenceNotBoundToActiveCampaigns(
	ctx context.Context,
	ownerUserID string,
	match func(storage.AgentRecord) bool,
	message string,
) error {
	if g.agentStore == nil {
		return status.Error(codes.Internal, "agent store is not configured")
	}
	if match == nil {
		return nil
	}

	pageToken := ""
	for {
		page, err := g.agentStore.ListAgentsByOwner(ctx, ownerUserID, maxPageSize, pageToken)
		if err != nil {
			return status.Errorf(codes.Internal, "list agents: %v", err)
		}
		for _, record := range page.Agents {
			if !match(record) {
				continue
			}
			if err := g.ensureAgentNotBoundToActiveCampaigns(ctx, record.ID); err != nil {
				return errWithMessage(err, message)
			}
		}

		nextPageToken := strings.TrimSpace(page.NextPageToken)
		if nextPageToken == "" || nextPageToken == pageToken {
			return nil
		}
		pageToken = nextPageToken
	}
}

func (g authReferenceUsageGuard) ensureAgentNotBoundToActiveCampaigns(ctx context.Context, agentID string) error {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return status.Error(codes.InvalidArgument, "agent_id is required")
	}

	activeCampaignCount, err := g.activeCampaignCount(ctx, agentID)
	if err != nil {
		return err
	}
	if activeCampaignCount > 0 {
		return status.Error(codes.FailedPrecondition, "agent is bound to active campaigns")
	}
	return nil
}

func (g authReferenceUsageGuard) activeCampaignCount(ctx context.Context, agentID string) (int32, error) {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return 0, status.Error(codes.InvalidArgument, "agent_id is required")
	}
	if g.gameCampaignAIClient == nil {
		return 0, nil
	}

	usage, err := g.gameCampaignAIClient.GetCampaignAIBindingUsage(ctx, &gamev1.GetCampaignAIBindingUsageRequest{
		AiAgentId: agentID,
	})
	if err != nil {
		return 0, status.Errorf(codes.Internal, "get campaign ai binding usage: %v", err)
	}
	return usage.GetActiveCampaignCount(), nil
}

func errWithMessage(err error, message string) error {
	if err == nil || strings.TrimSpace(message) == "" {
		return err
	}
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	return status.Error(st.Code(), message)
}
