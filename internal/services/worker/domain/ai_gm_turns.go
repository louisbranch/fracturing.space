package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	gameintegration "github.com/louisbranch/fracturing.space/internal/services/game/integration"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type workerCampaignAIOrchestrationClient interface {
	QueueAIGMTurn(ctx context.Context, in *gamev1.QueueAIGMTurnRequest, opts ...grpc.CallOption) (*gamev1.QueueAIGMTurnResponse, error)
}

// AIGMTurnRequestedHandler re-checks authoritative interaction state and queues
// an AI-owned GM turn when the current session is eligible.
type AIGMTurnRequestedHandler struct {
	orchestration workerCampaignAIOrchestrationClient
}

func NewAIGMTurnRequestedHandler(orchestration workerCampaignAIOrchestrationClient) *AIGMTurnRequestedHandler {
	return &AIGMTurnRequestedHandler{orchestration: orchestration}
}

func (h *AIGMTurnRequestedHandler) Handle(ctx context.Context, event OutboxEvent) error {
	if h == nil || h.orchestration == nil {
		return Permanent(fmt.Errorf("ai gm turn handler dependencies are not configured"))
	}
	payload, err := decodeAIGMTurnRequestedPayload(event)
	if err != nil {
		return Permanent(err)
	}
	_, err = h.orchestration.QueueAIGMTurn(serviceContext(ctx), &gamev1.QueueAIGMTurnRequest{
		CampaignId:      payload.CampaignID,
		SessionId:       payload.SessionID,
		SourceEventType: payload.SourceEventType,
		SourceSceneId:   payload.SourceSceneID,
		SourcePhaseId:   payload.SourcePhaseID,
	})
	if err == nil {
		return nil
	}
	switch status.Code(err) {
	case codes.InvalidArgument, codes.PermissionDenied, codes.Unauthenticated:
		return Permanent(fmt.Errorf("queue ai gm turn: %w", err))
	default:
		return fmt.Errorf("queue ai gm turn: %w", err)
	}
}

func decodeAIGMTurnRequestedPayload(event OutboxEvent) (gameintegration.AIGMTurnRequestedOutboxPayload, error) {
	if event == nil {
		return gameintegration.AIGMTurnRequestedOutboxPayload{}, fmt.Errorf("outbox event is required")
	}
	var payload gameintegration.AIGMTurnRequestedOutboxPayload
	if err := json.Unmarshal([]byte(event.GetPayloadJson()), &payload); err != nil {
		return payload, fmt.Errorf("decode ai gm turn requested payload: %w", err)
	}
	payload.CampaignID = strings.TrimSpace(payload.CampaignID)
	payload.SessionID = strings.TrimSpace(payload.SessionID)
	payload.SourceEventType = strings.TrimSpace(payload.SourceEventType)
	payload.SourceSceneID = strings.TrimSpace(payload.SourceSceneID)
	payload.SourcePhaseID = strings.TrimSpace(payload.SourcePhaseID)
	if payload.CampaignID == "" {
		return payload, fmt.Errorf("campaign_id is required in ai gm turn requested payload")
	}
	if payload.SessionID == "" {
		return payload, fmt.Errorf("session_id is required in ai gm turn requested payload")
	}
	return payload, nil
}
