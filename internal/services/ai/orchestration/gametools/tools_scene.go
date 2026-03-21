package gametools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
)

type sceneCreateInput struct {
	CampaignID   string   `json:"campaign_id,omitempty"`
	SessionID    string   `json:"session_id,omitempty"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	CharacterIDs []string `json:"character_ids,omitempty"`
}

type sceneCreateResult struct {
	SceneID      string   `json:"scene_id"`
	CampaignID   string   `json:"campaign_id"`
	SessionID    string   `json:"session_id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	CharacterIDs []string `json:"character_ids,omitempty"`
}

func (s *DirectSession) sceneCreate(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input sceneCreateInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID(input.CampaignID)
	sessionID := s.resolveSessionID(input.SessionID)
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	if sessionID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("session_id is required")
	}

	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	resp, err := s.clients.Scene.CreateScene(callCtx, &statev1.CreateSceneRequest{
		CampaignId:   campaignID,
		SessionId:    sessionID,
		Name:         strings.TrimSpace(input.Name),
		Description:  strings.TrimSpace(input.Description),
		CharacterIds: append([]string(nil), input.CharacterIDs...),
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("create scene failed: %w", err)
	}
	if resp == nil || strings.TrimSpace(resp.GetSceneId()) == "" {
		return orchestration.ToolResult{}, fmt.Errorf("create scene response is missing")
	}

	return toolResultJSON(sceneCreateResult{
		SceneID:      strings.TrimSpace(resp.GetSceneId()),
		CampaignID:   campaignID,
		SessionID:    sessionID,
		Name:         strings.TrimSpace(input.Name),
		Description:  strings.TrimSpace(input.Description),
		CharacterIDs: append([]string(nil), input.CharacterIDs...),
	})
}
