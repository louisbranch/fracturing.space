package ai

import (
	"context"
	"errors"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// agentProtoWithUsage enriches one agent read with auth health and usage metadata.
func (h *AgentHandlers) agentProtoWithUsage(ctx context.Context, record storage.AgentRecord) (*aiv1.Agent, error) {
	proto := agentToProto(record)
	proto.AuthState = h.agentAuthState(ctx, record)

	activeCampaignCount, err := h.activeCampaignCount(ctx, record.ID)
	if err != nil {
		return nil, err
	}
	proto.ActiveCampaignCount = activeCampaignCount
	return proto, nil
}

// agentProtoWithAuthState enriches one agent read with auth health only.
func (h *AgentHandlers) agentProtoWithAuthState(ctx context.Context, record storage.AgentRecord) *aiv1.Agent {
	proto := agentToProto(record)
	proto.AuthState = h.agentAuthState(ctx, record)
	return proto
}

// agentAuthState derives a non-mutating runtime auth-health view for an agent.
func (h *AgentHandlers) agentAuthState(ctx context.Context, record storage.AgentRecord) aiv1.AgentAuthState {
	requestedProvider := providerFromString(record.Provider)
	authReference, err := agentAuthReferenceFromRecord(record)
	if err != nil {
		return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE
	}

	switch authReference.Kind {
	case agent.AuthReferenceKindCredential:
		if h == nil || h.credentialStore == nil {
			return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE
		}
		credentialRecord, err := h.credentialStore.GetCredential(ctx, authReference.CredentialID())
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE
			}
			return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE
		}
		switch {
		case credentialFromRecord(credentialRecord).IsUsableBy(record.OwnerUserID, requestedProvider):
			return aiv1.AgentAuthState_AGENT_AUTH_STATE_READY
		case credential.ParseStatus(credentialRecord.Status).IsRevoked():
			return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_REVOKED
		default:
			return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE
		}
	case agent.AuthReferenceKindProviderGrant:
		if h == nil || h.providerGrantStore == nil {
			return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE
		}
		grantRecord, err := h.providerGrantStore.GetProviderGrant(ctx, authReference.ProviderGrantID())
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE
			}
			return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE
		}
		switch {
		case providerGrantFromRecord(grantRecord).IsUsableBy(record.OwnerUserID, requestedProvider):
			return aiv1.AgentAuthState_AGENT_AUTH_STATE_READY
		case providergrant.ParseStatus(grantRecord.Status).IsRevoked():
			return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_REVOKED
		default:
			return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE
		}
	default:
		return aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE
	}
}

// activeCampaignCount returns the number of DRAFT/ACTIVE campaigns currently bound
// to one AI agent. When the game usage client is unavailable, the count degrades
// to zero to preserve existing standalone AI-service behavior.
func (h *AgentHandlers) activeCampaignCount(ctx context.Context, agentID string) (int32, error) {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return 0, status.Error(codes.InvalidArgument, "agent_id is required")
	}
	if h == nil || h.gameCampaignAIClient == nil {
		return 0, nil
	}

	usage, err := h.gameCampaignAIClient.GetCampaignAIBindingUsage(ctx, &gamev1.GetCampaignAIBindingUsageRequest{
		AiAgentId: agentID,
	})
	if err != nil {
		return 0, status.Errorf(codes.Internal, "get campaign ai binding usage: %v", err)
	}
	return usage.GetActiveCampaignCount(), nil
}
