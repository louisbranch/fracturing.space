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
	CampaignID   string                         `json:"campaign_id,omitempty"`
	SceneID      string                         `json:"scene_id,omitempty"`
	Interaction  *interactionGMInteractionInput `json:"interaction"`
	CharacterIDs []string                       `json:"character_ids"`
}

type interactionGMInteractionIllustrationInput struct {
	ImageURL string `json:"image_url,omitempty"`
	Alt      string `json:"alt,omitempty"`
	Caption  string `json:"caption,omitempty"`
}

type interactionGMInteractionBeatInput struct {
	BeatID string `json:"beat_id,omitempty"`
	Type   string `json:"type"`
	Text   string `json:"text"`
}

type interactionGMInteractionInput struct {
	Title        string                                     `json:"title"`
	CharacterIDs []string                                   `json:"character_ids,omitempty"`
	Illustration *interactionGMInteractionIllustrationInput `json:"illustration,omitempty"`
	Beats        []interactionGMInteractionBeatInput        `json:"beats"`
}

type interactionScenePlayerRevisionInput struct {
	ParticipantID string   `json:"participant_id"`
	Reason        string   `json:"reason"`
	CharacterIDs  []string `json:"character_ids,omitempty"`
}

type interactionReviewAdvanceToPlayersInput struct {
	Interaction      *interactionGMInteractionInput `json:"interaction"`
	NextCharacterIDs []string                       `json:"next_character_ids"`
}

type interactionReviewRequestRevisionsInput struct {
	Interaction *interactionGMInteractionInput        `json:"interaction"`
	Revisions   []interactionScenePlayerRevisionInput `json:"revisions"`
}

type interactionReviewReturnToGMInput struct {
	Interaction *interactionGMInteractionInput `json:"interaction"`
}

type interactionResolveScenePlayerPhaseReviewInput struct {
	CampaignID       string                                  `json:"campaign_id,omitempty"`
	SceneID          string                                  `json:"scene_id,omitempty"`
	AdvanceToPlayers *interactionReviewAdvanceToPlayersInput `json:"advance_to_players,omitempty"`
	RequestRevisions *interactionReviewRequestRevisionsInput `json:"request_revisions,omitempty"`
	ReturnToGM       *interactionReviewReturnToGMInput       `json:"return_to_gm,omitempty"`
}

type interactionCommitSceneGMInteractionInput struct {
	CampaignID  string                         `json:"campaign_id,omitempty"`
	SceneID     string                         `json:"scene_id,omitempty"`
	Interaction *interactionGMInteractionInput `json:"interaction"`
}

type interactionReplaceInterruptedScenePhaseInput struct {
	SceneID      string                         `json:"scene_id,omitempty"`
	Interaction  *interactionGMInteractionInput `json:"interaction"`
	CharacterIDs []string                       `json:"character_ids"`
}

type interactionResolveInterruptedScenePhaseInput struct {
	CampaignID             string                                        `json:"campaign_id,omitempty"`
	ResumeOriginalPhase    bool                                          `json:"resume_original_phase,omitempty"`
	ReplaceWithPlayerPhase *interactionReplaceInterruptedScenePhaseInput `json:"replace_with_player_phase,omitempty"`
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
	SceneID            string                          `json:"scene_id,omitempty"`
	SessionID          string                          `json:"session_id,omitempty"`
	Name               string                          `json:"name,omitempty"`
	Description        string                          `json:"description,omitempty"`
	Characters         []interactionCharacterResult    `json:"characters,omitempty"`
	CurrentInteraction *interactionGMInteractionResult `json:"current_interaction,omitempty"`
}

type interactionGMInteractionBeatResult struct {
	BeatID string `json:"beat_id,omitempty"`
	Type   string `json:"type,omitempty"`
	Text   string `json:"text,omitempty"`
}

type interactionGMInteractionResult struct {
	InteractionID string                               `json:"interaction_id,omitempty"`
	SceneID       string                               `json:"scene_id,omitempty"`
	PhaseID       string                               `json:"phase_id,omitempty"`
	ParticipantID string                               `json:"participant_id,omitempty"`
	Title         string                               `json:"title,omitempty"`
	CharacterIDs  []string                             `json:"character_ids,omitempty"`
	Beats         []interactionGMInteractionBeatResult `json:"beats,omitempty"`
	CreatedAt     string                               `json:"created_at,omitempty"`
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
	RequestedByParticipantID    string                     `json:"requested_by_participant_id,omitempty"`
	Reason                      string                     `json:"reason,omitempty"`
	InterruptedSceneID          string                     `json:"interrupted_scene_id,omitempty"`
	InterruptedPhaseID          string                     `json:"interrupted_phase_id,omitempty"`
	InterruptedPhaseStatus      string                     `json:"interrupted_phase_status,omitempty"`
	ResolutionPending           bool                       `json:"resolution_pending"`
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
		SceneId:    input.SceneID,
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
	interaction, err := gmInteractionInputFromTool(input.Interaction, input.CharacterIDs)
	if err != nil {
		return orchestration.ToolResult{}, err
	}
	resp, err := s.clients.Interaction.StartScenePlayerPhase(callCtx, &statev1.StartScenePlayerPhaseRequest{
		CampaignId:   campaignID,
		SceneId:      sceneID,
		CharacterIds: input.CharacterIDs,
		Interaction:  interaction,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("start scene player phase failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("start scene player phase response is missing")
	}
	return toolResultJSON(interactionStateFromProto(resp.GetState()))
}

func (s *DirectSession) interactionResolveScenePlayerPhaseReview(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input interactionResolveScenePlayerPhaseReviewInput
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
	req := &statev1.ResolveScenePlayerPhaseReviewRequest{
		CampaignId: campaignID,
		SceneId:    sceneID,
	}
	switch {
	case input.AdvanceToPlayers != nil:
		advanceInteraction, err := gmInteractionInputFromTool(input.AdvanceToPlayers.Interaction, input.AdvanceToPlayers.NextCharacterIDs)
		if err != nil {
			return orchestration.ToolResult{}, err
		}
		req.Resolution = &statev1.ResolveScenePlayerPhaseReviewRequest_AdvanceToPlayers{
			AdvanceToPlayers: &statev1.ResolveScenePlayerPhaseReviewAdvanceToPlayers{
				NextCharacterIds: append([]string(nil), input.AdvanceToPlayers.NextCharacterIDs...),
				Interaction:      advanceInteraction,
			},
		}
	case input.RequestRevisions != nil:
		revisionInteraction, err := gmInteractionInputFromTool(input.RequestRevisions.Interaction, nil)
		if err != nil {
			return orchestration.ToolResult{}, err
		}
		revisions := make([]*statev1.ScenePlayerRevisionRequest, 0, len(input.RequestRevisions.Revisions))
		for _, rev := range input.RequestRevisions.Revisions {
			revisions = append(revisions, &statev1.ScenePlayerRevisionRequest{
				ParticipantId: strings.TrimSpace(rev.ParticipantID),
				Reason:        strings.TrimSpace(rev.Reason),
				CharacterIds:  append([]string(nil), rev.CharacterIDs...),
			})
		}
		req.Resolution = &statev1.ResolveScenePlayerPhaseReviewRequest_RequestRevisions{
			RequestRevisions: &statev1.ResolveScenePlayerPhaseReviewRequestRevisions{
				Interaction: revisionInteraction,
				Revisions:   revisions,
			},
		}
	case input.ReturnToGM != nil:
		returnInteraction, err := gmInteractionInputFromTool(input.ReturnToGM.Interaction, nil)
		if err != nil {
			return orchestration.ToolResult{}, err
		}
		req.Resolution = &statev1.ResolveScenePlayerPhaseReviewRequest_ReturnToGm{
			ReturnToGm: &statev1.ResolveScenePlayerPhaseReviewReturnToGM{
				Interaction: returnInteraction,
			},
		}
	default:
		return orchestration.ToolResult{}, fmt.Errorf("advance_to_players, request_revisions, or return_to_gm is required")
	}
	resp, err := s.clients.Interaction.ResolveScenePlayerPhaseReview(callCtx, req)
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("resolve scene player phase review failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("resolve scene player phase review response is missing")
	}
	return toolResultJSON(interactionStateFromProto(resp.GetState()))
}

func (s *DirectSession) interactionCommitSceneGMInteraction(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input interactionCommitSceneGMInteractionInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID(input.CampaignID)
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()
	state, err := s.getInteractionState(callCtx, campaignID)
	if err != nil {
		return orchestration.ToolResult{}, err
	}
	if state.GetPlayerPhase().GetStatus() == statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM_REVIEW {
		return orchestration.ToolResult{}, fmt.Errorf("scene is waiting on gm review; use interaction_scene_review_resolve instead of interaction_scene_gm_interaction_commit")
	}
	sceneID, err := s.resolveSceneIDFromState(state, input.SceneID)
	if err != nil {
		return orchestration.ToolResult{}, err
	}
	interaction, err := gmInteractionInputFromTool(input.Interaction, nil)
	if err != nil {
		return orchestration.ToolResult{}, err
	}
	resp, err := s.clients.Interaction.CommitSceneGMInteraction(callCtx, &statev1.CommitSceneGMInteractionRequest{
		CampaignId:  campaignID,
		SceneId:     sceneID,
		Interaction: interaction,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("commit scene gm interaction failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("commit scene gm interaction response is missing")
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
		Reason:     input.Reason,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("pause session for ooc failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("pause session for ooc response is missing")
	}
	return toolResultJSON(interactionStateFromProto(resp.GetState()))
}

func (s *DirectSession) interactionResolveInterruptedScenePhase(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input interactionResolveInterruptedScenePhaseInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	campaignID := s.resolveCampaignID(input.CampaignID)
	if campaignID == "" {
		return orchestration.ToolResult{}, fmt.Errorf("campaign_id is required")
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()
	req := &statev1.ResolveInterruptedScenePhaseRequest{CampaignId: campaignID}
	switch {
	case input.ResumeOriginalPhase:
		req.Resolution = &statev1.ResolveInterruptedScenePhaseRequest_ResumeOriginalPhase{
			ResumeOriginalPhase: &statev1.ResolveInterruptedScenePhaseResumeOriginal{},
		}
	case input.ReplaceWithPlayerPhase != nil:
		interaction, err := gmInteractionInputFromTool(input.ReplaceWithPlayerPhase.Interaction, input.ReplaceWithPlayerPhase.CharacterIDs)
		if err != nil {
			return orchestration.ToolResult{}, err
		}
		req.Resolution = &statev1.ResolveInterruptedScenePhaseRequest_ReplaceWithPlayerPhase{
			ReplaceWithPlayerPhase: &statev1.ResolveInterruptedScenePhaseReplaceWithPlayerPhase{
				SceneId:          strings.TrimSpace(input.ReplaceWithPlayerPhase.SceneID),
				NextCharacterIds: append([]string(nil), input.ReplaceWithPlayerPhase.CharacterIDs...),
				Interaction:      interaction,
			},
		}
	default:
		return orchestration.ToolResult{}, fmt.Errorf("resume_original_phase or replace_with_player_phase is required")
	}
	resp, err := s.clients.Interaction.ResolveInterruptedScenePhase(callCtx, req)
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("resolve interrupted scene phase failed: %w", err)
	}
	if resp == nil {
		return orchestration.ToolResult{}, fmt.Errorf("resolve interrupted scene phase response is missing")
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
		Body:       input.Body,
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
	state, err := s.getInteractionState(ctx, campaignID)
	if err != nil {
		return "", err
	}
	return s.resolveSceneIDFromState(state, explicit)
}

func (s *DirectSession) getInteractionState(ctx context.Context, campaignID string) (*statev1.InteractionState, error) {
	resp, err := s.clients.Interaction.GetInteractionState(ctx, &statev1.GetInteractionStateRequest{CampaignId: campaignID})
	if err != nil {
		return nil, fmt.Errorf("get interaction state failed: %w", err)
	}
	if resp == nil || resp.State == nil {
		return nil, fmt.Errorf("get interaction state response is missing")
	}
	return resp.GetState(), nil
}

func (s *DirectSession) resolveSceneIDFromState(state *statev1.InteractionState, explicit string) (string, error) {
	sceneID := strings.TrimSpace(explicit)
	if sceneID != "" {
		return sceneID, nil
	}
	if state == nil {
		return "", fmt.Errorf("interaction state is required")
	}
	sceneID = strings.TrimSpace(state.GetActiveScene().GetSceneId())
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
			SceneID:            state.GetActiveScene().GetSceneId(),
			SessionID:          state.GetActiveScene().GetSessionId(),
			Name:               state.GetActiveScene().GetName(),
			Description:        state.GetActiveScene().GetDescription(),
			Characters:         make([]interactionCharacterResult, 0, len(state.GetActiveScene().GetCharacters())),
			CurrentInteraction: interactionGMInteractionFromProto(state.GetActiveScene().GetCurrentInteraction()),
		},
		PlayerPhase: interactionPlayerPhaseResult{
			PhaseID:              state.GetPlayerPhase().GetPhaseId(),
			Status:               scenePhaseStatusToString(state.GetPlayerPhase().GetStatus()),
			ActingCharacterIDs:   append([]string(nil), state.GetPlayerPhase().GetActingCharacterIds()...),
			ActingParticipantIDs: append([]string(nil), state.GetPlayerPhase().GetActingParticipantIds()...),
			Slots:                make([]interactionPlayerSlotResult, 0, len(state.GetPlayerPhase().GetSlots())),
		},
		OOC: interactionOOCStateResult{
			Open:                        state.GetOoc().GetOpen(),
			Posts:                       make([]interactionOOCPostResult, 0, len(state.GetOoc().GetPosts())),
			ReadyToResumeParticipantIDs: append([]string(nil), state.GetOoc().GetReadyToResumeParticipantIds()...),
			RequestedByParticipantID:    state.GetOoc().GetRequestedByParticipantId(),
			Reason:                      state.GetOoc().GetReason(),
			InterruptedSceneID:          state.GetOoc().GetInterruptedSceneId(),
			InterruptedPhaseID:          state.GetOoc().GetInterruptedPhaseId(),
			InterruptedPhaseStatus:      scenePhaseStatusToString(state.GetOoc().GetInterruptedPhaseStatus()),
			ResolutionPending:           state.GetOoc().GetResolutionPending(),
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

func interactionGMInteractionFromProto(interaction *statev1.GMInteraction) *interactionGMInteractionResult {
	if interaction == nil {
		return nil
	}
	result := &interactionGMInteractionResult{
		InteractionID: interaction.GetInteractionId(),
		SceneID:       interaction.GetSceneId(),
		PhaseID:       interaction.GetPhaseId(),
		ParticipantID: interaction.GetParticipantId(),
		Title:         interaction.GetTitle(),
		CharacterIDs:  append([]string(nil), interaction.GetCharacterIds()...),
		Beats:         make([]interactionGMInteractionBeatResult, 0, len(interaction.GetBeats())),
	}
	if interaction.GetCreatedAt() != nil {
		result.CreatedAt = interaction.GetCreatedAt().AsTime().UTC().Format(time.RFC3339)
	}
	for _, beat := range interaction.GetBeats() {
		result.Beats = append(result.Beats, interactionGMInteractionBeatResult{
			BeatID: beat.GetBeatId(),
			Type:   gmInteractionBeatTypeToString(beat.GetType()),
			Text:   beat.GetText(),
		})
	}
	return result
}

func singleBeatGMInteractionInput(title string, beatType statev1.GMInteractionBeatType, text string, characterIDs ...string) *statev1.GMInteractionInput {
	return &statev1.GMInteractionInput{
		Title:        title,
		CharacterIds: append([]string(nil), characterIDs...),
		Beats: []*statev1.GMInteractionInputBeat{{
			BeatId: "beat-1",
			Type:   beatType,
			Text:   strings.TrimSpace(text),
		}},
	}
}

func gmInteractionInputFromTool(input *interactionGMInteractionInput, fallbackCharacterIDs []string) (*statev1.GMInteractionInput, error) {
	if input == nil {
		return nil, fmt.Errorf("interaction is required")
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, fmt.Errorf("interaction title is required")
	}
	beats := make([]*statev1.GMInteractionInputBeat, 0, len(input.Beats))
	for idx, beat := range input.Beats {
		text := strings.TrimSpace(beat.Text)
		if text == "" {
			return nil, fmt.Errorf("interaction beats[%d].text is required", idx)
		}
		beatType, err := parseGMInteractionBeatType(beat.Type)
		if err != nil {
			return nil, fmt.Errorf("interaction beats[%d].type: %w", idx, err)
		}
		beatID := strings.TrimSpace(beat.BeatID)
		if beatID == "" {
			beatID = fmt.Sprintf("beat-%d", idx+1)
		}
		beats = append(beats, &statev1.GMInteractionInputBeat{
			BeatId: beatID,
			Type:   beatType,
			Text:   text,
		})
	}
	if len(beats) == 0 {
		return nil, fmt.Errorf("interaction beats are required")
	}
	characterIDs := append([]string(nil), input.CharacterIDs...)
	if len(characterIDs) == 0 {
		characterIDs = append([]string(nil), fallbackCharacterIDs...)
	}
	result := &statev1.GMInteractionInput{
		Title:        title,
		CharacterIds: characterIDs,
		Beats:        beats,
	}
	if input.Illustration != nil {
		imageURL := strings.TrimSpace(input.Illustration.ImageURL)
		alt := strings.TrimSpace(input.Illustration.Alt)
		caption := strings.TrimSpace(input.Illustration.Caption)
		if imageURL == "" {
			return nil, fmt.Errorf("interaction illustration image_url is required when illustration is provided")
		}
		if alt == "" {
			return nil, fmt.Errorf("interaction illustration alt is required when illustration is provided")
		}
		result.Illustration = &statev1.GMInteractionInputIllustration{
			ImageUrl: imageURL,
			Alt:      alt,
			Caption:  caption,
		}
	}
	return result, nil
}

func parseGMInteractionBeatType(raw string) (statev1.GMInteractionBeatType, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "fiction":
		return statev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_FICTION, nil
	case "prompt":
		return statev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_PROMPT, nil
	case "resolution":
		return statev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_RESOLUTION, nil
	case "consequence":
		return statev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_CONSEQUENCE, nil
	case "guidance":
		return statev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_GUIDANCE, nil
	default:
		return statev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_UNSPECIFIED, fmt.Errorf("unsupported beat type %q", raw)
	}
}

func gmInteractionBeatTypeToString(value statev1.GMInteractionBeatType) string {
	name := strings.TrimSpace(value.String())
	if name == "" || name == statev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_UNSPECIFIED.String() {
		return ""
	}
	name = strings.TrimPrefix(name, "GM_INTERACTION_BEAT_TYPE_")
	return strings.ToLower(name)
}
