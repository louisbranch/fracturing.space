package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
)

func issueAISessionGrantForRoom(ctx context.Context, campaignAIClient statev1.CampaignAIServiceClient, room *campaignRoom, userID string) error {
	if campaignAIClient == nil || room == nil {
		return nil
	}
	if !room.aiRelayEnabled() {
		room.clearAISessionGrant()
		return nil
	}
	sessionID := strings.TrimSpace(room.currentSessionID())
	if sessionID == "" {
		room.clearAISessionGrant()
		return fmt.Errorf("session id is required")
	}
	aiAgentID := strings.TrimSpace(room.aiAgentIDValue())
	if aiAgentID == "" {
		room.clearAISessionGrant()
		return fmt.Errorf("ai agent id is required")
	}
	callCtx := ctx
	if callCtx == nil {
		callCtx = context.Background()
	}
	if strings.TrimSpace(userID) != "" {
		callCtx = grpcauthctx.WithUserID(callCtx, userID)
	}
	resp, err := campaignAIClient.IssueCampaignAISessionGrant(callCtx, &statev1.IssueCampaignAISessionGrantRequest{
		CampaignId: room.campaignID,
		SessionId:  sessionID,
		AiAgentId:  aiAgentID,
	})
	if err != nil {
		room.clearAISessionGrant()
		return err
	}
	grant := resp.GetGrant()
	token := strings.TrimSpace(grant.GetToken())
	if token == "" {
		room.clearAISessionGrant()
		return fmt.Errorf("issued ai session grant token is empty")
	}
	expiresAt := time.Time{}
	if grant.GetExpiresAt() != nil {
		expiresAt = grant.GetExpiresAt().AsTime().UTC()
	}
	room.setAISessionGrant(token, grant.GetAuthEpoch(), expiresAt)
	return nil
}

func syncRoomAIContextFromGame(ctx context.Context, campaignAIClient statev1.CampaignAIServiceClient, room *campaignRoom) error {
	if campaignAIClient == nil || room == nil {
		return nil
	}
	callCtx := ctx
	if callCtx == nil {
		callCtx = context.Background()
	}
	resp, err := campaignAIClient.GetCampaignAIAuthState(callCtx, &statev1.GetCampaignAIAuthStateRequest{
		CampaignId: room.campaignID,
	})
	if err != nil {
		return err
	}
	room.setSessionID(strings.TrimSpace(resp.GetActiveSessionId()))
	room.setAIBinding(room.gmModeValue(), strings.TrimSpace(resp.GetAiAgentId()))
	return nil
}

func isAICampaignContextEvent(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case "campaign.ai_bound", "campaign.ai_unbound", "campaign.ai_auth_rotated", "session.started", "session.ended":
		return true
	default:
		return false
	}
}
