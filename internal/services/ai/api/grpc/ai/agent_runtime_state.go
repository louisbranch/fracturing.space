package ai

import (
	"context"
	"errors"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// agentProtoWithUsage enriches one agent read with auth health and usage metadata.
func (s *Service) agentProtoWithUsage(ctx context.Context, record storage.AgentRecord) (*aiv1.Agent, error) {
	proto := agentToProto(record)
	proto.AuthState = s.agentAuthState(ctx, record)

	activeCampaignCount, err := s.activeCampaignCount(ctx, record.ID)
	if err != nil {
		return nil, err
	}
	proto.ActiveCampaignCount = activeCampaignCount
	return proto, nil
}

// agentProtoWithAuthState enriches one agent read with auth health only.
func (s *Service) agentProtoWithAuthState(ctx context.Context, record storage.AgentRecord) *aiv1.Agent {
	proto := agentToProto(record)
	proto.AuthState = s.agentAuthState(ctx, record)
	return proto
}

// agentAuthState derives a non-mutating runtime auth-health view for an agent.
func (s *Service) agentAuthState(ctx context.Context, record storage.AgentRecord) aiv1.AgentAuthState {
	requestedProvider := providerFromString(record.Provider)
	credentialID := strings.TrimSpace(record.CredentialID)
	if credentialID != "" {
		if s == nil || s.credentialStore == nil {
			return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE
		}
		credentialRecord, err := s.credentialStore.GetCredential(ctx, credentialID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE
			}
			return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE
		}
		switch {
		case (credential.Credential{
			OwnerUserID: credentialRecord.OwnerUserID,
			Provider:    providerFromString(credentialRecord.Provider),
			Status:      credential.ParseStatus(credentialRecord.Status),
		}).IsUsableBy(record.OwnerUserID, requestedProvider):
			return aiv1.AgentAuthState_AGENT_AUTH_STATE_READY
		case credential.ParseStatus(credentialRecord.Status).IsRevoked():
			return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_REVOKED
		default:
			return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE
		}
	}

	providerGrantID := strings.TrimSpace(record.ProviderGrantID)
	if providerGrantID == "" {
		return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE
	}
	if s == nil || s.providerGrantStore == nil {
		return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE
	}
	grantRecord, err := s.providerGrantStore.GetProviderGrant(ctx, providerGrantID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE
		}
		return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE
	}
	switch {
	case (providergrant.ProviderGrant{
		OwnerUserID: grantRecord.OwnerUserID,
		Provider:    providerFromString(grantRecord.Provider),
		Status:      providergrant.ParseStatus(grantRecord.Status),
	}).IsUsableBy(record.OwnerUserID, requestedProvider):
		return aiv1.AgentAuthState_AGENT_AUTH_STATE_READY
	case providergrant.ParseStatus(grantRecord.Status).IsRevoked():
		return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_REVOKED
	default:
		return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE
	}
}

// activeCampaignCount returns the number of DRAFT/ACTIVE campaigns currently bound
// to one AI agent. When the game usage client is unavailable, the count degrades
// to zero to preserve existing standalone AI-service behavior.
func (s *Service) activeCampaignCount(ctx context.Context, agentID string) (int32, error) {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return 0, status.Error(codes.InvalidArgument, "agent_id is required")
	}
	if s == nil || s.gameCampaignAIClient == nil {
		return 0, nil
	}

	usage, err := s.gameCampaignAIClient.GetCampaignAIBindingUsage(ctx, &gamev1.GetCampaignAIBindingUsageRequest{
		AiAgentId: agentID,
	})
	if err != nil {
		return 0, status.Errorf(codes.Internal, "get campaign ai binding usage: %v", err)
	}
	return usage.GetActiveCampaignCount(), nil
}
