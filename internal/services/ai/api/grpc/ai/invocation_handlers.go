package ai

import (
	"context"
	"errors"
	"strings"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
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

// SubmitCampaignTurn records one campaign chat turn and emits stream events.
func (s *Service) SubmitCampaignTurn(ctx context.Context, in *aiv1.SubmitCampaignTurnRequest) (*aiv1.SubmitCampaignTurnResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "submit campaign turn request is required")
	}
	if s.agentStore == nil {
		return nil, status.Error(codes.Internal, "agent store is not configured")
	}
	if s.campaignTurnStore == nil {
		return nil, status.Error(codes.Internal, "campaign turn store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign_id is required")
	}
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}
	agentID := strings.TrimSpace(in.GetAgentId())
	if agentID == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}
	sessionGrant := strings.TrimSpace(in.GetSessionGrant())
	if sessionGrant == "" {
		return nil, status.Error(codes.InvalidArgument, "session_grant is required")
	}
	body := strings.TrimSpace(in.GetBody())
	if body == "" {
		return nil, status.Error(codes.InvalidArgument, "body is required")
	}
	if _, err := s.validateCampaignSessionGrant(ctx, sessionGrant, campaignID, sessionID, agentID); err != nil {
		return nil, err
	}

	agentRecord, err := s.agentStore.GetAgent(ctx, agentID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "agent not found")
		}
		return nil, status.Errorf(codes.Internal, "get agent: %v", err)
	}

	turnID, err := s.idGenerator()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate turn id: %v", err)
	}
	createdAt := s.clock().UTC()
	turnRecord := storage.CampaignTurnRecord{
		ID:                 turnID,
		CampaignID:         campaignID,
		SessionID:          sessionID,
		AgentID:            agentID,
		RequesterUserID:    strings.TrimSpace(userIDFromContext(ctx)),
		ParticipantID:      strings.TrimSpace(in.GetParticipantId()),
		ParticipantName:    strings.TrimSpace(in.GetParticipantName()),
		CorrelationMessage: strings.TrimSpace(in.GetMessageId()),
		InputText:          body,
		Status:             "processing",
		CreatedAt:          createdAt,
		UpdatedAt:          createdAt,
	}
	if err := s.campaignTurnStore.PutCampaignTurn(ctx, turnRecord); err != nil {
		return nil, status.Errorf(codes.Internal, "put campaign turn: %v", err)
	}
	if _, err := s.campaignTurnStore.AppendCampaignTurnEvent(ctx, storage.CampaignTurnEventRecord{
		CampaignID:         campaignID,
		SessionID:          sessionID,
		TurnID:             turnID,
		Kind:               "turn_accepted",
		Content:            "",
		ParticipantVisible: false,
		CorrelationMessage: strings.TrimSpace(in.GetMessageId()),
		CreatedAt:          createdAt,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "append turn accepted event: %v", err)
	}

	provider := providergrant.Provider(strings.ToLower(strings.TrimSpace(agentRecord.Provider)))
	adapter, ok := s.providerInvocationAdapters[provider]
	if !ok || adapter == nil {
		_ = s.campaignTurnStore.UpdateCampaignTurnStatus(ctx, turnID, "failed", s.clock().UTC())
		_, _ = s.campaignTurnStore.AppendCampaignTurnEvent(ctx, storage.CampaignTurnEventRecord{
			CampaignID:         campaignID,
			SessionID:          sessionID,
			TurnID:             turnID,
			Kind:               "error",
			Content:            "provider invocation adapter is unavailable",
			ParticipantVisible: false,
			CorrelationMessage: strings.TrimSpace(in.GetMessageId()),
			CreatedAt:          s.clock().UTC(),
		})
		return &aiv1.SubmitCampaignTurnResponse{TurnId: turnID}, nil
	}

	invokeToken, err := s.resolveAgentInvokeToken(ctx, strings.TrimSpace(agentRecord.OwnerUserID), agentRecord)
	if err != nil {
		_ = s.campaignTurnStore.UpdateCampaignTurnStatus(ctx, turnID, "failed", s.clock().UTC())
		_, _ = s.campaignTurnStore.AppendCampaignTurnEvent(ctx, storage.CampaignTurnEventRecord{
			CampaignID:         campaignID,
			SessionID:          sessionID,
			TurnID:             turnID,
			Kind:               "error",
			Content:            err.Error(),
			ParticipantVisible: false,
			CorrelationMessage: strings.TrimSpace(in.GetMessageId()),
			CreatedAt:          s.clock().UTC(),
		})
		return &aiv1.SubmitCampaignTurnResponse{TurnId: turnID}, nil
	}

	result, err := adapter.Invoke(ctx, ProviderInvokeInput{
		Model:            agentRecord.Model,
		Input:            body,
		CredentialSecret: invokeToken,
	})
	if err != nil {
		_ = s.campaignTurnStore.UpdateCampaignTurnStatus(ctx, turnID, "failed", s.clock().UTC())
		_, _ = s.campaignTurnStore.AppendCampaignTurnEvent(ctx, storage.CampaignTurnEventRecord{
			CampaignID:         campaignID,
			SessionID:          sessionID,
			TurnID:             turnID,
			Kind:               "error",
			Content:            err.Error(),
			ParticipantVisible: false,
			CorrelationMessage: strings.TrimSpace(in.GetMessageId()),
			CreatedAt:          s.clock().UTC(),
		})
		return &aiv1.SubmitCampaignTurnResponse{TurnId: turnID}, nil
	}

	outputText := strings.TrimSpace(result.OutputText)
	if outputText == "" {
		outputText = "..."
	}
	if _, err := s.campaignTurnStore.AppendCampaignTurnEvent(ctx, storage.CampaignTurnEventRecord{
		CampaignID:         campaignID,
		SessionID:          sessionID,
		TurnID:             turnID,
		Kind:               "model_output",
		Content:            outputText,
		ParticipantVisible: true,
		CorrelationMessage: strings.TrimSpace(in.GetMessageId()),
		CreatedAt:          s.clock().UTC(),
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "append model output event: %v", err)
	}
	if err := s.campaignTurnStore.UpdateCampaignTurnStatus(ctx, turnID, "completed", s.clock().UTC()); err != nil {
		return nil, status.Errorf(codes.Internal, "update campaign turn status: %v", err)
	}

	return &aiv1.SubmitCampaignTurnResponse{TurnId: turnID}, nil
}

// SubscribeCampaignTurnEvents streams campaign turn events to callers.
func (s *Service) SubscribeCampaignTurnEvents(in *aiv1.SubscribeCampaignTurnEventsRequest, stream aiv1.InvocationService_SubscribeCampaignTurnEventsServer) error {
	if in == nil {
		return status.Error(codes.InvalidArgument, "subscribe campaign turn events request is required")
	}
	if s.campaignTurnStore == nil {
		return status.Error(codes.Internal, "campaign turn store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return status.Error(codes.InvalidArgument, "campaign_id is required")
	}
	sessionGrant := strings.TrimSpace(in.GetSessionGrant())
	if sessionGrant == "" {
		return status.Error(codes.InvalidArgument, "session_grant is required")
	}
	grantClaims, err := s.validateCampaignSessionGrant(stream.Context(), sessionGrant, campaignID, "", "")
	if err != nil {
		return err
	}

	afterSequenceID := in.GetAfterSequenceId()
	for {
		if err := stream.Context().Err(); err != nil {
			return err
		}
		now := time.Now
		if s != nil && s.clock != nil {
			now = s.clock
		}
		if now().UTC().After(grantClaims.ExpiresAt) {
			return status.Error(codes.PermissionDenied, "session grant is expired")
		}

		records, err := s.campaignTurnStore.ListCampaignTurnEvents(stream.Context(), campaignID, afterSequenceID, 100)
		if err != nil {
			return status.Errorf(codes.Internal, "list campaign turn events: %v", err)
		}
		if len(records) == 0 {
			timer := time.NewTimer(500 * time.Millisecond)
			select {
			case <-stream.Context().Done():
				timer.Stop()
				return stream.Context().Err()
			case <-timer.C:
			}
			continue
		}

		for _, record := range records {
			if err := stream.Send(&aiv1.CampaignTurnEvent{
				CampaignId:           record.CampaignID,
				SessionId:            record.SessionID,
				TurnId:               record.TurnID,
				SequenceId:           record.SequenceID,
				Kind:                 campaignTurnEventKindToProto(record.Kind),
				Content:              record.Content,
				ParticipantVisible:   record.ParticipantVisible,
				CorrelationMessageId: record.CorrelationMessage,
				CreatedAt:            timestamppb.New(record.CreatedAt),
			}); err != nil {
				return err
			}
			afterSequenceID = record.SequenceID
		}
	}
}

func campaignTurnEventKindToProto(value string) aiv1.CampaignTurnEventKind {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "turn_accepted":
		return aiv1.CampaignTurnEventKind_CAMPAIGN_TURN_EVENT_KIND_TURN_ACCEPTED
	case "model_output":
		return aiv1.CampaignTurnEventKind_CAMPAIGN_TURN_EVENT_KIND_MODEL_OUTPUT
	case "error":
		return aiv1.CampaignTurnEventKind_CAMPAIGN_TURN_EVENT_KIND_ERROR
	default:
		return aiv1.CampaignTurnEventKind_CAMPAIGN_TURN_EVENT_KIND_UNSPECIFIED
	}
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
