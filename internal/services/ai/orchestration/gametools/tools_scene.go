package gametools

import (
	"context"
	"encoding/json"
	"fmt"

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

type sceneUpdateInput struct {
	CampaignID  string `json:"campaign_id,omitempty"`
	SceneID     string `json:"scene_id"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type sceneUpdateResult struct {
	SceneID string `json:"scene_id"`
	Updated bool   `json:"updated"`
}

type sceneEndInput struct {
	CampaignID string `json:"campaign_id,omitempty"`
	SceneID    string `json:"scene_id"`
	Reason     string `json:"reason,omitempty"`
}

type sceneEndResult struct {
	SceneID string `json:"scene_id"`
	Ended   bool   `json:"ended"`
}

type sceneTransitionInput struct {
	CampaignID    string `json:"campaign_id,omitempty"`
	SourceSceneID string `json:"source_scene_id,omitempty"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
}

type sceneTransitionResult struct {
	NewSceneID    string `json:"new_scene_id"`
	SourceSceneID string `json:"source_scene_id"`
}

type sceneAddCharacterInput struct {
	CampaignID  string `json:"campaign_id,omitempty"`
	SceneID     string `json:"scene_id"`
	CharacterID string `json:"character_id"`
}

type sceneAddCharacterResult struct {
	SceneID     string `json:"scene_id"`
	CharacterID string `json:"character_id"`
	Added       bool   `json:"added"`
}

type sceneRemoveCharacterInput struct {
	CampaignID  string `json:"campaign_id,omitempty"`
	SceneID     string `json:"scene_id"`
	CharacterID string `json:"character_id"`
}

type sceneRemoveCharacterResult struct {
	SceneID     string `json:"scene_id"`
	CharacterID string `json:"character_id"`
	Removed     bool   `json:"removed"`
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
		Name:         input.Name,
		Description:  input.Description,
		CharacterIds: append([]string(nil), input.CharacterIDs...),
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("create scene failed: %w", err)
	}
	if resp == nil || resp.GetSceneId() == "" {
		return orchestration.ToolResult{}, fmt.Errorf("create scene response is missing")
	}

	return toolResultJSON(sceneCreateResult{
		SceneID:      resp.GetSceneId(),
		CampaignID:   campaignID,
		SessionID:    sessionID,
		Name:         input.Name,
		Description:  input.Description,
		CharacterIDs: append([]string(nil), input.CharacterIDs...),
	})
}

func (s *DirectSession) sceneUpdate(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input sceneUpdateInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID(input.CampaignID)
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	if input.SceneID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("scene_id is required")
	}

	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	if _, err := s.clients.Scene.UpdateScene(callCtx, &statev1.UpdateSceneRequest{
		CampaignId:  campaignID,
		SceneId:     input.SceneID,
		Name:        input.Name,
		Description: input.Description,
	}); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("update scene failed: %w", err)
	}

	return toolResultJSON(sceneUpdateResult{SceneID: input.SceneID, Updated: true})
}

func (s *DirectSession) sceneEnd(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input sceneEndInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID(input.CampaignID)
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	if input.SceneID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("scene_id is required")
	}

	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	if _, err := s.clients.Scene.EndScene(callCtx, &statev1.EndSceneRequest{
		CampaignId: campaignID,
		SceneId:    input.SceneID,
		Reason:     input.Reason,
	}); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("end scene failed: %w", err)
	}

	return toolResultJSON(sceneEndResult{SceneID: input.SceneID, Ended: true})
}

func (s *DirectSession) sceneTransition(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input sceneTransitionInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID(input.CampaignID)
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}

	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	sourceSceneID, err := s.resolveSceneID(callCtx, campaignID, input.SourceSceneID)
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("resolve source scene: %w", err)
	}

	resp, err := s.clients.Scene.TransitionScene(callCtx, &statev1.TransitionSceneRequest{
		CampaignId:    campaignID,
		SourceSceneId: sourceSceneID,
		Name:          input.Name,
		Description:   input.Description,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("transition scene failed: %w", err)
	}

	newSceneID := resp.GetNewSceneId()
	if newSceneID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("transition scene response is missing new_scene_id")
	}

	return toolResultJSON(sceneTransitionResult{
		NewSceneID:    newSceneID,
		SourceSceneID: sourceSceneID,
	})
}

func (s *DirectSession) sceneAddCharacter(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input sceneAddCharacterInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID(input.CampaignID)
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	if input.SceneID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("scene_id is required")
	}
	if input.CharacterID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("character_id is required")
	}

	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	if _, err := s.clients.Scene.AddCharacterToScene(callCtx, &statev1.AddCharacterToSceneRequest{
		CampaignId:  campaignID,
		SceneId:     input.SceneID,
		CharacterId: input.CharacterID,
	}); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("add character to scene failed: %w", err)
	}

	return toolResultJSON(sceneAddCharacterResult{
		SceneID:     input.SceneID,
		CharacterID: input.CharacterID,
		Added:       true,
	})
}

func (s *DirectSession) sceneRemoveCharacter(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input sceneRemoveCharacterInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID(input.CampaignID)
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	if input.SceneID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("scene_id is required")
	}
	if input.CharacterID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("character_id is required")
	}

	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	if _, err := s.clients.Scene.RemoveCharacterFromScene(callCtx, &statev1.RemoveCharacterFromSceneRequest{
		CampaignId:  campaignID,
		SceneId:     input.SceneID,
		CharacterId: input.CharacterID,
	}); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("remove character from scene failed: %w", err)
	}

	return toolResultJSON(sceneRemoveCharacterResult{
		SceneID:     input.SceneID,
		CharacterID: input.CharacterID,
		Removed:     true,
	})
}
