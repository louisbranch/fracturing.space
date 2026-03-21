package gametools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
)

// --- Input types ---

type interactionSetActiveSceneInput struct {
	CampaignID string `json:"campaign_id,omitempty"`
	SceneID    string `json:"scene_id"`
}

type interactionStartScenePlayerPhaseInput struct {
	CampaignID   string   `json:"campaign_id,omitempty"`
	SceneID      string   `json:"scene_id,omitempty"`
	FrameText    string   `json:"frame_text"`
	CharacterIDs []string `json:"character_ids"`
}

type interactionAcceptScenePlayerPhaseInput struct {
	CampaignID string `json:"campaign_id,omitempty"`
	SceneID    string `json:"scene_id,omitempty"`
}

type interactionScenePlayerRevisionInput struct {
	ParticipantID string   `json:"participant_id"`
	Reason        string   `json:"reason"`
	CharacterIDs  []string `json:"character_ids,omitempty"`
}

type interactionRequestScenePlayerRevisionsInput struct {
	CampaignID string                                `json:"campaign_id,omitempty"`
	SceneID    string                                `json:"scene_id,omitempty"`
	Revisions  []interactionScenePlayerRevisionInput `json:"revisions"`
}

type interactionEndScenePlayerPhaseInput struct {
	CampaignID string `json:"campaign_id,omitempty"`
	SceneID    string `json:"scene_id,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

type interactionCommitSceneGMOutputInput struct {
	CampaignID string `json:"campaign_id,omitempty"`
	SceneID    string `json:"scene_id,omitempty"`
	Text       string `json:"text"`
}

type interactionPauseOOCInput struct {
	CampaignID string `json:"campaign_id,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

type interactionPostOOCInput struct {
	CampaignID string `json:"campaign_id,omitempty"`
	Body       string `json:"body"`
}

// --- Result types ---

type interactionStateResult struct {
	CampaignID    string                       `json:"campaign_id,omitempty"`
	CampaignName  string                       `json:"campaign_name,omitempty"`
	Locale        string                       `json:"locale,omitempty"`
	Viewer        interactionViewerResult      `json:"viewer"`
	ActiveSession interactionSessionResult     `json:"active_session"`
	ActiveScene   interactionSceneResult       `json:"active_scene"`
	PlayerPhase   interactionPlayerPhaseResult `json:"player_phase"`
	OOC           interactionOOCStateResult    `json:"ooc"`
}

type interactionViewerResult struct {
	ParticipantID string `json:"participant_id,omitempty"`
	Name          string `json:"name,omitempty"`
	Role          string `json:"role,omitempty"`
}

type interactionSessionResult struct {
	SessionID string `json:"session_id,omitempty"`
	Name      string `json:"name,omitempty"`
}

type interactionCharacterResult struct {
	CharacterID        string `json:"character_id,omitempty"`
	Name               string `json:"name,omitempty"`
	OwnerParticipantID string `json:"owner_participant_id,omitempty"`
}

type interactionSceneResult struct {
	SceneID     string                       `json:"scene_id,omitempty"`
	SessionID   string                       `json:"session_id,omitempty"`
	Name        string                       `json:"name,omitempty"`
	Description string                       `json:"description,omitempty"`
	Characters  []interactionCharacterResult `json:"characters,omitempty"`
	GMOutput    *interactionGMOutputResult   `json:"gm_output,omitempty"`
}

type interactionGMOutputResult struct {
	Text          string `json:"text,omitempty"`
	ParticipantID string `json:"participant_id,omitempty"`
	UpdatedAt     string `json:"updated_at,omitempty"`
}

type interactionPlayerSlotResult struct {
	ParticipantID      string   `json:"participant_id,omitempty"`
	SummaryText        string   `json:"summary_text,omitempty"`
	CharacterIDs       []string `json:"character_ids,omitempty"`
	UpdatedAt          string   `json:"updated_at,omitempty"`
	Yielded            bool     `json:"yielded,omitempty"`
	ReviewStatus       string   `json:"review_status,omitempty"`
	ReviewReason       string   `json:"review_reason,omitempty"`
	ReviewCharacterIDs []string `json:"review_character_ids,omitempty"`
}

type interactionPlayerPhaseResult struct {
	PhaseID              string                        `json:"phase_id,omitempty"`
	Status               string                        `json:"status,omitempty"`
	FrameText            string                        `json:"frame_text,omitempty"`
	ActingCharacterIDs   []string                      `json:"acting_character_ids,omitempty"`
	ActingParticipantIDs []string                      `json:"acting_participant_ids,omitempty"`
	Slots                []interactionPlayerSlotResult `json:"slots,omitempty"`
}

type interactionOOCPostResult struct {
	PostID        string `json:"post_id,omitempty"`
	ParticipantID string `json:"participant_id,omitempty"`
	Body          string `json:"body,omitempty"`
	CreatedAt     string `json:"created_at,omitempty"`
}

type interactionOOCStateResult struct {
	Open                        bool                       `json:"open"`
	Posts                       []interactionOOCPostResult `json:"posts,omitempty"`
	ReadyToResumeParticipantIDs []string                   `json:"ready_to_resume_participant_ids,omitempty"`
}

// --- Handlers ---

func (s *DirectSession) interactionSetActiveScene(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input interactionSetActiveSceneInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID(input.CampaignID)
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()
	resp, err := s.clients.Interaction.SetActiveScene(callCtx, &statev1.SetActiveSceneRequest{
		CampaignId: campaignID,
		SceneId:    strings.TrimSpace(input.SceneID),
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("set active scene failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("set active scene response is missing")
	}
	return toolResultJSON(interactionStateFromProto(resp.GetState()))
}

func (s *DirectSession) interactionStartScenePlayerPhase(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input interactionStartScenePlayerPhaseInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID(input.CampaignID)
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()
	sceneID, err := s.resolveSceneID(callCtx, campaignID, input.SceneID)
	if err != nil {
		return orchestration.ToolResult{}, err
	}
	resp, err := s.clients.Interaction.StartScenePlayerPhase(callCtx, &statev1.StartScenePlayerPhaseRequest{
		CampaignId:   campaignID,
		SceneId:      sceneID,
		FrameText:    strings.TrimSpace(input.FrameText),
		CharacterIds: input.CharacterIDs,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("start scene player phase failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("start scene player phase response is missing")
	}
	return toolResultJSON(interactionStateFromProto(resp.GetState()))
}

func (s *DirectSession) interactionAcceptScenePlayerPhase(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input interactionAcceptScenePlayerPhaseInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID(input.CampaignID)
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()
	sceneID, err := s.resolveSceneID(callCtx, campaignID, input.SceneID)
	if err != nil {
		return orchestration.ToolResult{}, err
	}
	resp, err := s.clients.Interaction.AcceptScenePlayerPhase(callCtx, &statev1.AcceptScenePlayerPhaseRequest{
		CampaignId: campaignID,
		SceneId:    sceneID,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("accept scene player phase failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("accept scene player phase response is missing")
	}
	return toolResultJSON(interactionStateFromProto(resp.GetState()))
}

func (s *DirectSession) interactionRequestScenePlayerRevisions(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input interactionRequestScenePlayerRevisionsInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID(input.CampaignID)
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()
	sceneID, err := s.resolveSceneID(callCtx, campaignID, input.SceneID)
	if err != nil {
		return orchestration.ToolResult{}, err
	}
	revisions := make([]*statev1.ScenePlayerRevisionRequest, 0, len(input.Revisions))
	for _, rev := range input.Revisions {
		revisions = append(revisions, &statev1.ScenePlayerRevisionRequest{
			ParticipantId: strings.TrimSpace(rev.ParticipantID),
			Reason:        strings.TrimSpace(rev.Reason),
			CharacterIds:  append([]string(nil), rev.CharacterIDs...),
		})
	}
	resp, err := s.clients.Interaction.RequestScenePlayerRevisions(callCtx, &statev1.RequestScenePlayerRevisionsRequest{
		CampaignId: campaignID,
		SceneId:    sceneID,
		Revisions:  revisions,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("request scene player revisions failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("request scene player revisions response is missing")
	}
	return toolResultJSON(interactionStateFromProto(resp.GetState()))
}

func (s *DirectSession) interactionEndScenePlayerPhase(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input interactionEndScenePlayerPhaseInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID(input.CampaignID)
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()
	sceneID, err := s.resolveSceneID(callCtx, campaignID, input.SceneID)
	if err != nil {
		return orchestration.ToolResult{}, err
	}
	resp, err := s.clients.Interaction.EndScenePlayerPhase(callCtx, &statev1.EndScenePlayerPhaseRequest{
		CampaignId: campaignID,
		SceneId:    sceneID,
		Reason:     strings.TrimSpace(input.Reason),
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("end scene player phase failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("end scene player phase response is missing")
	}
	return toolResultJSON(interactionStateFromProto(resp.GetState()))
}

func (s *DirectSession) interactionCommitSceneGMOutput(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input interactionCommitSceneGMOutputInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID(input.CampaignID)
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()
	sceneID, err := s.resolveSceneID(callCtx, campaignID, input.SceneID)
	if err != nil {
		return orchestration.ToolResult{}, err
	}
	resp, err := s.clients.Interaction.CommitSceneGMOutput(callCtx, &statev1.CommitSceneGMOutputRequest{
		CampaignId: campaignID,
		SceneId:    sceneID,
		Text:       strings.TrimSpace(input.Text),
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("commit scene gm output failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("commit scene gm output response is missing")
	}
	return toolResultJSON(interactionStateFromProto(resp.GetState()))
}

func (s *DirectSession) interactionPauseOOC(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input interactionPauseOOCInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID(input.CampaignID)
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()
	resp, err := s.clients.Interaction.PauseSessionForOOC(callCtx, &statev1.PauseSessionForOOCRequest{
		CampaignId: campaignID,
		Reason:     strings.TrimSpace(input.Reason),
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("pause session for ooc failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("pause session for ooc response is missing")
	}
	return toolResultJSON(interactionStateFromProto(resp.GetState()))
}

func (s *DirectSession) interactionPostOOC(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input interactionPostOOCInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID(input.CampaignID)
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()
	resp, err := s.clients.Interaction.PostSessionOOC(callCtx, &statev1.PostSessionOOCRequest{
		CampaignId: campaignID,
		Body:       strings.TrimSpace(input.Body),
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("post session ooc failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("post session ooc response is missing")
	}
	return toolResultJSON(interactionStateFromProto(resp.GetState()))
}

func (s *DirectSession) interactionMarkOOCReady(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input interactionPauseOOCInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID(input.CampaignID)
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()
	resp, err := s.clients.Interaction.MarkOOCReadyToResume(callCtx, &statev1.MarkOOCReadyToResumeRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("mark ooc ready failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("mark ooc ready response is missing")
	}
	return toolResultJSON(interactionStateFromProto(resp.GetState()))
}

func (s *DirectSession) interactionClearOOCReady(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input interactionPauseOOCInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID(input.CampaignID)
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()
	resp, err := s.clients.Interaction.ClearOOCReadyToResume(callCtx, &statev1.ClearOOCReadyToResumeRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("clear ooc ready failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("clear ooc ready response is missing")
	}
	return toolResultJSON(interactionStateFromProto(resp.GetState()))
}

func (s *DirectSession) interactionResumeOOC(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input interactionPauseOOCInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID(input.CampaignID)
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()
	resp, err := s.clients.Interaction.ResumeFromOOC(callCtx, &statev1.ResumeFromOOCRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("resume from ooc failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("resume from ooc response is missing")
	}
	return toolResultJSON(interactionStateFromProto(resp.GetState()))
}

// --- Helpers ---

func (s *DirectSession) resolveCampaignID(explicit string) string {
	if id := strings.TrimSpace(explicit); id != "" {
		return id
	}
	return strings.TrimSpace(s.sc.CampaignID)
}

func (s *DirectSession) resolveSessionID(explicit string) string {
	if id := strings.TrimSpace(explicit); id != "" {
		return id
	}
	return strings.TrimSpace(s.sc.SessionID)
}

func (s *DirectSession) resolveSceneID(ctx context.Context, campaignID, explicit string) (string, error) {
	sceneID := strings.TrimSpace(explicit)
	if sceneID != "" {
		return sceneID, nil
	}
	resp, err := s.clients.Interaction.GetInteractionState(ctx, &statev1.GetInteractionStateRequest{CampaignId: campaignID})
	if err != nil {
		return "", fmt.Errorf("get interaction state failed: %w", err)
	}
	if resp == nil || resp.State == nil {
		return "", fmt.Errorf("get interaction state response is missing")
	}
	sceneID = strings.TrimSpace(resp.GetState().GetActiveScene().GetSceneId())
	if sceneID == "" {
		return "", fmt.Errorf("scene_id is required when no active scene is set")
	}
	return sceneID, nil
}

func interactionStateFromProto(state *statev1.InteractionState) interactionStateResult {
	if state == nil {
		return interactionStateResult{}
	}
	result := interactionStateResult{
		CampaignID:   state.GetCampaignId(),
		CampaignName: state.GetCampaignName(),
		Locale:       state.GetLocale().String(),
		Viewer: interactionViewerResult{
			ParticipantID: state.GetViewer().GetParticipantId(),
			Name:          state.GetViewer().GetName(),
			Role:          participantRoleToString(state.GetViewer().GetRole()),
		},
		ActiveSession: interactionSessionResult{
			SessionID: state.GetActiveSession().GetSessionId(),
			Name:      state.GetActiveSession().GetName(),
		},
		ActiveScene: interactionSceneResult{
			SceneID:     state.GetActiveScene().GetSceneId(),
			SessionID:   state.GetActiveScene().GetSessionId(),
			Name:        state.GetActiveScene().GetName(),
			Description: state.GetActiveScene().GetDescription(),
			Characters:  make([]interactionCharacterResult, 0, len(state.GetActiveScene().GetCharacters())),
			GMOutput:    interactionGMOutputFromProto(state.GetActiveScene().GetGmOutput()),
		},
		PlayerPhase: interactionPlayerPhaseResult{
			PhaseID:              state.GetPlayerPhase().GetPhaseId(),
			Status:               scenePhaseStatusToString(state.GetPlayerPhase().GetStatus()),
			FrameText:            state.GetPlayerPhase().GetFrameText(),
			ActingCharacterIDs:   append([]string(nil), state.GetPlayerPhase().GetActingCharacterIds()...),
			ActingParticipantIDs: append([]string(nil), state.GetPlayerPhase().GetActingParticipantIds()...),
			Slots:                make([]interactionPlayerSlotResult, 0, len(state.GetPlayerPhase().GetSlots())),
		},
		OOC: interactionOOCStateResult{
			Open:                        state.GetOoc().GetOpen(),
			Posts:                       make([]interactionOOCPostResult, 0, len(state.GetOoc().GetPosts())),
			ReadyToResumeParticipantIDs: append([]string(nil), state.GetOoc().GetReadyToResumeParticipantIds()...),
		},
	}
	for _, ch := range state.GetActiveScene().GetCharacters() {
		result.ActiveScene.Characters = append(result.ActiveScene.Characters, interactionCharacterResult{
			CharacterID:        ch.GetCharacterId(),
			Name:               ch.GetName(),
			OwnerParticipantID: ch.GetOwnerParticipantId(),
		})
	}
	for _, slot := range state.GetPlayerPhase().GetSlots() {
		result.PlayerPhase.Slots = append(result.PlayerPhase.Slots, interactionPlayerSlotResult{
			ParticipantID:      slot.GetParticipantId(),
			SummaryText:        slot.GetSummaryText(),
			CharacterIDs:       append([]string(nil), slot.GetCharacterIds()...),
			UpdatedAt:          formatTimestamp(slot.GetUpdatedAt()),
			Yielded:            slot.GetYielded(),
			ReviewStatus:       scenePlayerSlotReviewStatusToString(slot.GetReviewStatus()),
			ReviewReason:       slot.GetReviewReason(),
			ReviewCharacterIDs: append([]string(nil), slot.GetReviewCharacterIds()...),
		})
	}
	for _, post := range state.GetOoc().GetPosts() {
		result.OOC.Posts = append(result.OOC.Posts, interactionOOCPostResult{
			PostID:        post.GetPostId(),
			ParticipantID: post.GetParticipantId(),
			Body:          post.GetBody(),
			CreatedAt:     formatTimestamp(post.GetCreatedAt()),
		})
	}
	return result
}

func interactionGMOutputFromProto(output *statev1.InteractionGMOutput) *interactionGMOutputResult {
	if output == nil {
		return nil
	}
	result := &interactionGMOutputResult{
		Text:          output.GetText(),
		ParticipantID: output.GetParticipantId(),
	}
	if output.GetUpdatedAt() != nil {
		result.UpdatedAt = output.GetUpdatedAt().AsTime().UTC().Format(time.RFC3339)
	}
	return result
}
