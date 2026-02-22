package ai

import (
	"context"
	"errors"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// InvokeAgent executes one provider call using an owned active agent auth reference.
func (s *Service) InvokeAgent(ctx context.Context, in *aiv1.InvokeAgentRequest) (*aiv1.InvokeAgentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "invoke agent request is required")
	}
	if s.agentStore == nil {
		return nil, status.Error(codes.Internal, "agent store is not configured")
	}
	if s.sealer == nil {
		return nil, status.Error(codes.Internal, "secret sealer is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}
	agentID := strings.TrimSpace(in.GetAgentId())
	if agentID == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}
	input := strings.TrimSpace(in.GetInput())
	if input == "" {
		return nil, status.Error(codes.InvalidArgument, "input is required")
	}

	agentRecord, err := s.agentStore.GetAgent(ctx, agentID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "agent not found")
		}
		return nil, status.Errorf(codes.Internal, "get agent: %v", err)
	}

	authorized, sharedAccess, accessRequestID, err := s.isAuthorizedToInvokeAgent(ctx, userID, agentRecord)
	if err != nil {
		return nil, err
	}
	if !authorized {
		return nil, status.Error(codes.NotFound, "agent not found")
	}

	provider := providergrant.Provider(strings.ToLower(strings.TrimSpace(agentRecord.Provider)))
	adapter, ok := s.providerInvocationAdapters[provider]
	if !ok || adapter == nil {
		return nil, status.Error(codes.FailedPrecondition, "provider invocation adapter is unavailable")
	}

	invokeToken, err := s.resolveAgentInvokeToken(ctx, strings.TrimSpace(agentRecord.OwnerUserID), agentRecord)
	if err != nil {
		return nil, err
	}
	result, err := adapter.Invoke(ctx, ProviderInvokeInput{
		Model:            agentRecord.Model,
		Input:            input,
		CredentialSecret: invokeToken,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "invoke provider: %v", err)
	}
	outputText := strings.TrimSpace(result.OutputText)
	if outputText == "" {
		return nil, status.Error(codes.Internal, "provider returned empty output")
	}
	if sharedAccess {
		if err := s.putAuditEvent(ctx, storage.AuditEventRecord{
			EventName:       "agent.invoke.shared",
			ActorUserID:     userID,
			OwnerUserID:     strings.TrimSpace(agentRecord.OwnerUserID),
			RequesterUserID: userID,
			AgentID:         strings.TrimSpace(agentRecord.ID),
			AccessRequestID: accessRequestID,
			Outcome:         "success",
			CreatedAt:       s.clock().UTC(),
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "put audit event: %v", err)
		}
	}
	return &aiv1.InvokeAgentResponse{
		OutputText: outputText,
		Provider:   providerToProto(agentRecord.Provider),
		Model:      agentRecord.Model,
	}, nil
}
func (s *Service) resolveAgentInvokeToken(ctx context.Context, ownerUserID string, agentRecord storage.AgentRecord) (string, error) {
	credentialID := strings.TrimSpace(agentRecord.CredentialID)
	providerGrantID := strings.TrimSpace(agentRecord.ProviderGrantID)
	hasCredential := credentialID != ""
	hasProviderGrant := providerGrantID != ""
	if hasCredential == hasProviderGrant {
		return "", status.Error(codes.FailedPrecondition, "agent auth reference is invalid")
	}

	if hasCredential {
		if s.credentialStore == nil {
			return "", status.Error(codes.Internal, "credential store is not configured")
		}
		credentialRecord, err := s.credentialStore.GetCredential(ctx, credentialID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return "", status.Error(codes.FailedPrecondition, "credential is unavailable")
			}
			return "", status.Errorf(codes.Internal, "get credential: %v", err)
		}
		if !isCredentialActiveForUser(credentialRecord, ownerUserID, agentRecord.Provider) {
			return "", status.Error(codes.FailedPrecondition, "credential must be active and owned by caller")
		}
		// Secrets are decrypted only for in-memory request dispatch and never returned.
		credentialSecret, err := s.sealer.Open(credentialRecord.SecretCiphertext)
		if err != nil {
			return "", status.Errorf(codes.Internal, "open credential secret: %v", err)
		}
		return credentialSecret, nil
	}

	grantRecord, err := s.resolveProviderGrantForInvocation(ctx, ownerUserID, providerGrantID, agentRecord.Provider)
	if err != nil {
		return "", err
	}
	tokenPlaintext, err := s.sealer.Open(grantRecord.TokenCiphertext)
	if err != nil {
		return "", status.Errorf(codes.Internal, "open provider token: %v", err)
	}
	accessToken, err := accessTokenFromTokenPayload(tokenPlaintext)
	if err != nil {
		return "", status.Errorf(codes.FailedPrecondition, "provider token payload is invalid: %v", err)
	}
	return accessToken, nil
}
