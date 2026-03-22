package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	gameintegration "github.com/louisbranch/fracturing.space/internal/services/game/integration"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxAITurnErrorLen = 512

type workerCampaignAIOrchestrationClient interface {
	QueueAIGMTurn(ctx context.Context, in *gamev1.QueueAIGMTurnRequest, opts ...grpc.CallOption) (*gamev1.QueueAIGMTurnResponse, error)
	StartAIGMTurn(ctx context.Context, in *gamev1.StartAIGMTurnRequest, opts ...grpc.CallOption) (*gamev1.StartAIGMTurnResponse, error)
	FailAIGMTurn(ctx context.Context, in *gamev1.FailAIGMTurnRequest, opts ...grpc.CallOption) (*gamev1.FailAIGMTurnResponse, error)
	CompleteAIGMTurn(ctx context.Context, in *gamev1.CompleteAIGMTurnRequest, opts ...grpc.CallOption) (*gamev1.CompleteAIGMTurnResponse, error)
}

type workerCampaignAIServiceClient interface {
	IssueCampaignAISessionGrant(ctx context.Context, in *gamev1.IssueCampaignAISessionGrantRequest, opts ...grpc.CallOption) (*gamev1.IssueCampaignAISessionGrantResponse, error)
}

type workerCampaignTurnClient interface {
	RunCampaignTurn(ctx context.Context, in *aiv1.RunCampaignTurnRequest, opts ...grpc.CallOption) (*aiv1.RunCampaignTurnResponse, error)
}

// AIGMTurnRequestedHandler runs the authoritative AI GM turn lifecycle.
type AIGMTurnRequestedHandler struct {
	orchestration workerCampaignAIOrchestrationClient
	game          workerCampaignAIServiceClient
	ai            workerCampaignTurnClient
}

func NewAIGMTurnRequestedHandler(orchestration workerCampaignAIOrchestrationClient, game workerCampaignAIServiceClient, ai workerCampaignTurnClient) *AIGMTurnRequestedHandler {
	return &AIGMTurnRequestedHandler{
		orchestration: orchestration,
		game:          game,
		ai:            ai,
	}
}

func (h *AIGMTurnRequestedHandler) Handle(ctx context.Context, event OutboxEvent) error {
	if h == nil || h.orchestration == nil || h.game == nil || h.ai == nil {
		return Permanent(fmt.Errorf("ai gm turn handler dependencies are not configured"))
	}
	payload, err := decodeAIGMTurnRequestedPayload(event)
	if err != nil {
		return Permanent(err)
	}
	callCtx := serviceContext(ctx)
	queued, err := h.orchestration.QueueAIGMTurn(callCtx, &gamev1.QueueAIGMTurnRequest{
		CampaignId:      payload.CampaignID,
		SessionId:       payload.SessionID,
		SourceEventType: payload.SourceEventType,
		SourceSceneId:   payload.SourceSceneID,
		SourcePhaseId:   payload.SourcePhaseID,
	})
	if err != nil {
		return classifyAIGMTurnError("queue ai gm turn", err)
	}
	queuedTurn := queued.GetAiTurn()
	if queuedTurn == nil {
		return fmt.Errorf("queue ai gm turn: ai turn state is missing")
	}
	switch queuedTurn.GetStatus() {
	case gamev1.AITurnStatus_AI_TURN_STATUS_IDLE,
		gamev1.AITurnStatus_AI_TURN_STATUS_RUNNING,
		gamev1.AITurnStatus_AI_TURN_STATUS_FAILED:
		return nil
	}
	turnToken := strings.TrimSpace(queuedTurn.GetTurnToken())
	if turnToken == "" {
		return fmt.Errorf("queue ai gm turn: turn token is missing")
	}
	if _, err := h.orchestration.StartAIGMTurn(callCtx, &gamev1.StartAIGMTurnRequest{
		CampaignId: payload.CampaignID,
		SessionId:  payload.SessionID,
		TurnToken:  turnToken,
	}); err != nil {
		return classifyAIGMTurnError("start ai gm turn", err)
	}

	fail := func(cause error) error {
		_, failErr := h.orchestration.FailAIGMTurn(callCtx, &gamev1.FailAIGMTurnRequest{
			CampaignId: payload.CampaignID,
			SessionId:  payload.SessionID,
			TurnToken:  turnToken,
			LastError:  truncateAITurnError(cause),
		})
		if failErr != nil {
			return fmt.Errorf("fail ai gm turn after error %q: %w", truncateAITurnError(cause), failErr)
		}
		return nil
	}

	grant, err := h.game.IssueCampaignAISessionGrant(callCtx, &gamev1.IssueCampaignAISessionGrantRequest{
		CampaignId: payload.CampaignID,
		SessionId:  payload.SessionID,
	})
	if err != nil {
		return fail(fmt.Errorf("issue campaign ai session grant: %w", err))
	}
	token := strings.TrimSpace(grant.GetGrant().GetToken())
	if token == "" {
		return fail(fmt.Errorf("issue campaign ai session grant: grant token is missing"))
	}
	if _, err := h.ai.RunCampaignTurn(callCtx, &aiv1.RunCampaignTurnRequest{
		SessionGrant: token,
		TurnToken:    turnToken,
	}); err != nil {
		return fail(fmt.Errorf("run campaign turn: %w", err))
	}
	if _, err := h.orchestration.CompleteAIGMTurn(callCtx, &gamev1.CompleteAIGMTurnRequest{
		CampaignId: payload.CampaignID,
		SessionId:  payload.SessionID,
		TurnToken:  turnToken,
	}); err != nil {
		cause := fmt.Errorf("complete ai gm turn: %w", err)
		failErr := fail(cause)
		if failErr == nil {
			return nil
		}
		if status.Code(failErr) == codes.FailedPrecondition {
			return nil
		}
		return failErr
	}
	return nil
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

func classifyAIGMTurnError(op string, err error) error {
	if isPermanentAIGMTurnError(err) {
		return Permanent(fmt.Errorf("%s: %w", op, err))
	}
	return fmt.Errorf("%s: %w", op, err)
}

func isPermanentAIGMTurnError(err error) bool {
	switch status.Code(err) {
	case codes.InvalidArgument, codes.PermissionDenied, codes.Unauthenticated, codes.FailedPrecondition:
		return true
	default:
		return false
	}
}

func truncateAITurnError(err error) string {
	if err == nil {
		return ""
	}
	value := strings.TrimSpace(err.Error())
	if len(value) <= maxAITurnErrorLen {
		return value
	}
	return strings.TrimSpace(value[:maxAITurnErrorLen])
}
