package gateway

import (
	"context"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// CampaignGameSurface returns the game-owned interaction context mapped for the
// web game surface.
func (g gameReadGateway) CampaignGameSurface(ctx context.Context, campaignID string) (campaignapp.CampaignGameSurface, error) {
	if g.read.Interaction == nil {
		return campaignapp.CampaignGameSurface{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.campaign_service_client_is_not_configured", "interaction service client is not configured")
	}
	resp, err := g.read.Interaction.GetInteractionState(ctx, &statev1.GetInteractionStateRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		return campaignapp.CampaignGameSurface{}, err
	}
	state := resp.GetState()
	if state == nil {
		return campaignapp.CampaignGameSurface{}, apperrors.E(apperrors.KindNotFound, "interaction state not found")
	}

	surface := campaignapp.CampaignGameSurface{
		Participant: campaignapp.CampaignGameParticipant{
			ID:   strings.TrimSpace(state.GetViewer().GetParticipantId()),
			Name: strings.TrimSpace(state.GetViewer().GetName()),
			Role: participantRoleLabel(state.GetViewer().GetRole()),
		},
		GMAuthorityParticipantID: strings.TrimSpace(state.GetGmAuthorityParticipantId()),
		AITurn: campaignapp.CampaignGameAITurn{
			Status:             aiTurnStatusLabel(state.GetAiTurn().GetStatus()),
			TurnToken:          strings.TrimSpace(state.GetAiTurn().GetTurnToken()),
			OwnerParticipantID: strings.TrimSpace(state.GetAiTurn().GetOwnerParticipantId()),
			SourceEventType:    strings.TrimSpace(state.GetAiTurn().GetSourceEventType()),
			SourceSceneID:      strings.TrimSpace(state.GetAiTurn().GetSourceSceneId()),
			SourcePhaseID:      strings.TrimSpace(state.GetAiTurn().GetSourcePhaseId()),
			LastError:          strings.TrimSpace(state.GetAiTurn().GetLastError()),
		},
		OOC: campaignapp.CampaignGameOOCState{
			Open:                        state.GetOoc().GetOpen(),
			ReadyToResumeParticipantIDs: append([]string(nil), state.GetOoc().GetReadyToResumeParticipantIds()...),
			Posts:                       make([]campaignapp.CampaignGameOOCPost, 0, len(state.GetOoc().GetPosts())),
		},
	}
	if sessionState := state.GetActiveSession(); sessionState != nil {
		surface.SessionID = strings.TrimSpace(sessionState.GetSessionId())
		surface.SessionName = strings.TrimSpace(sessionState.GetName())
	}
	if sceneState := state.GetActiveScene(); sceneState != nil {
		scene := &campaignapp.CampaignGameScene{
			ID:          strings.TrimSpace(sceneState.GetSceneId()),
			SessionID:   strings.TrimSpace(sceneState.GetSessionId()),
			Name:        strings.TrimSpace(sceneState.GetName()),
			Description: strings.TrimSpace(sceneState.GetDescription()),
			Characters:  make([]campaignapp.CampaignGameCharacter, 0, len(sceneState.GetCharacters())),
		}
		for _, character := range sceneState.GetCharacters() {
			if character == nil {
				continue
			}
			scene.Characters = append(scene.Characters, campaignapp.CampaignGameCharacter{
				ID:                 strings.TrimSpace(character.GetCharacterId()),
				Name:               strings.TrimSpace(character.GetName()),
				OwnerParticipantID: strings.TrimSpace(character.GetOwnerParticipantId()),
			})
		}
		surface.ActiveScene = scene
	}
	if phaseState := state.GetPlayerPhase(); phaseState != nil && strings.TrimSpace(phaseState.GetPhaseId()) != "" {
		phase := &campaignapp.CampaignGamePlayerPhase{
			PhaseID:              strings.TrimSpace(phaseState.GetPhaseId()),
			Status:               scenePhaseStatusLabel(phaseState.GetStatus()),
			FrameText:            strings.TrimSpace(phaseState.GetFrameText()),
			ActingCharacterIDs:   append([]string(nil), phaseState.GetActingCharacterIds()...),
			ActingParticipantIDs: append([]string(nil), phaseState.GetActingParticipantIds()...),
			Slots:                make([]campaignapp.CampaignGamePlayerSlot, 0, len(phaseState.GetSlots())),
		}
		for _, slot := range phaseState.GetSlots() {
			if slot == nil {
				continue
			}
			updatedAtUnix := int64(0)
			if updatedAt := slot.GetUpdatedAt(); updatedAt != nil {
				updatedAtUnix = updatedAt.AsTime().Unix()
			}
			phase.Slots = append(phase.Slots, campaignapp.CampaignGamePlayerSlot{
				ParticipantID:      strings.TrimSpace(slot.GetParticipantId()),
				SummaryText:        strings.TrimSpace(slot.GetSummaryText()),
				CharacterIDs:       append([]string(nil), slot.GetCharacterIds()...),
				UpdatedAtUnix:      updatedAtUnix,
				Yielded:            slot.GetYielded(),
				ReviewStatus:       scenePlayerSlotReviewStatusLabel(slot.GetReviewStatus()),
				ReviewReason:       strings.TrimSpace(slot.GetReviewReason()),
				ReviewCharacterIDs: append([]string(nil), slot.GetReviewCharacterIds()...),
			})
		}
		surface.PlayerPhase = phase
	}
	for _, post := range state.GetOoc().GetPosts() {
		if post == nil {
			continue
		}
		createdAtUnix := int64(0)
		if createdAt := post.GetCreatedAt(); createdAt != nil {
			createdAtUnix = createdAt.AsTime().Unix()
		}
		surface.OOC.Posts = append(surface.OOC.Posts, campaignapp.CampaignGameOOCPost{
			PostID:        strings.TrimSpace(post.GetPostId()),
			ParticipantID: strings.TrimSpace(post.GetParticipantId()),
			Body:          strings.TrimSpace(post.GetBody()),
			CreatedAtUnix: createdAtUnix,
		})
	}
	return surface, nil
}

// scenePhaseStatusLabel keeps grpc enum drift out of the web-facing surface.
func scenePhaseStatusLabel(status statev1.ScenePhaseStatus) string {
	switch status {
	case statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_PLAYERS:
		return "players"
	case statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM_REVIEW:
		return "gm_review"
	case statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM:
		return "gm"
	default:
		return "unspecified"
	}
}

// scenePlayerSlotReviewStatusLabel keeps grpc enum drift out of the web-facing slot state.
func scenePlayerSlotReviewStatusLabel(status statev1.ScenePlayerSlotReviewStatus) string {
	switch status {
	case statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_UNDER_REVIEW:
		return "under_review"
	case statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_ACCEPTED:
		return "accepted"
	case statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_CHANGES_REQUESTED:
		return "changes_requested"
	default:
		return "open"
	}
}

// aiTurnStatusLabel keeps grpc enum drift out of the web-facing AI turn state.
func aiTurnStatusLabel(status statev1.AITurnStatus) string {
	switch status {
	case statev1.AITurnStatus_AI_TURN_STATUS_QUEUED:
		return "queued"
	case statev1.AITurnStatus_AI_TURN_STATUS_RUNNING:
		return "running"
	case statev1.AITurnStatus_AI_TURN_STATUS_FAILED:
		return "failed"
	case statev1.AITurnStatus_AI_TURN_STATUS_IDLE:
		return "idle"
	default:
		return "unspecified"
	}
}
