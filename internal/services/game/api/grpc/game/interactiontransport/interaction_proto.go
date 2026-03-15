package interactiontransport

import (
	"sort"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func sessionInteractionToProto(interaction storage.SessionInteraction) *campaignv1.OOCState {
	posts := make([]*campaignv1.OOCPost, 0, len(interaction.OOCPosts))
	for _, post := range interaction.OOCPosts {
		posts = append(posts, &campaignv1.OOCPost{
			PostId:        post.PostID,
			ParticipantId: post.ParticipantID,
			Body:          post.Body,
			CreatedAt:     timestamppb.New(post.CreatedAt),
		})
	}
	sort.SliceStable(posts, func(i, j int) bool {
		return posts[i].CreatedAt.AsTime().Before(posts[j].CreatedAt.AsTime())
	})
	ready := append([]string(nil), interaction.ReadyToResumeParticipantIDs...)
	sort.Strings(ready)
	return &campaignv1.OOCState{
		Open:                        interaction.OOCPaused,
		Posts:                       posts,
		ReadyToResumeParticipantIds: ready,
	}
}

func aiTurnToProto(turn storage.SessionAITurn) *campaignv1.AITurnState {
	return &campaignv1.AITurnState{
		Status:             aiTurnStatusToProto(turn.Status),
		TurnToken:          turn.TurnToken,
		OwnerParticipantId: turn.OwnerParticipantID,
		SourceEventType:    turn.SourceEventType,
		SourceSceneId:      turn.SourceSceneID,
		SourcePhaseId:      turn.SourcePhaseID,
		LastError:          turn.LastError,
	}
}

func aiTurnStatusToProto(status session.AITurnStatus) campaignv1.AITurnStatus {
	switch status {
	case session.AITurnStatusQueued:
		return campaignv1.AITurnStatus_AI_TURN_STATUS_QUEUED
	case session.AITurnStatusRunning:
		return campaignv1.AITurnStatus_AI_TURN_STATUS_RUNNING
	case session.AITurnStatusFailed:
		return campaignv1.AITurnStatus_AI_TURN_STATUS_FAILED
	default:
		return campaignv1.AITurnStatus_AI_TURN_STATUS_IDLE
	}
}

func sceneInteractionToProto(interaction storage.SceneInteraction) *campaignv1.ScenePlayerPhase {
	if !interaction.PhaseOpen || strings.TrimSpace(interaction.PhaseID) == "" {
		return &campaignv1.ScenePlayerPhase{
			Status:               campaignv1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM,
			ActingCharacterIds:   []string{},
			ActingParticipantIds: []string{},
			Slots:                []*campaignv1.ScenePlayerSlot{},
		}
	}
	slots := make([]*campaignv1.ScenePlayerSlot, 0, len(interaction.Slots))
	for _, slot := range interaction.Slots {
		slots = append(slots, &campaignv1.ScenePlayerSlot{
			ParticipantId:      slot.ParticipantID,
			SummaryText:        slot.SummaryText,
			CharacterIds:       append([]string(nil), slot.CharacterIDs...),
			UpdatedAt:          timestamppb.New(slot.UpdatedAt),
			Yielded:            slot.Yielded,
			ReviewStatus:       scenePlayerSlotReviewStatusToProto(slot.ReviewStatus),
			ReviewReason:       slot.ReviewReason,
			ReviewCharacterIds: append([]string(nil), slot.ReviewCharacterIDs...),
		})
	}
	sort.SliceStable(slots, func(i, j int) bool {
		if slots[i].ParticipantId == slots[j].ParticipantId {
			return slots[i].UpdatedAt.AsTime().Before(slots[j].UpdatedAt.AsTime())
		}
		return slots[i].ParticipantId < slots[j].ParticipantId
	})
	actingCharacters := append([]string(nil), interaction.ActingCharacterIDs...)
	actingParticipants := append([]string(nil), interaction.ActingParticipantIDs...)
	sort.Strings(actingCharacters)
	sort.Strings(actingParticipants)
	return &campaignv1.ScenePlayerPhase{
		PhaseId:              interaction.PhaseID,
		Status:               scenePhaseStatusToProto(interaction.PhaseStatus),
		FrameText:            interaction.FrameText,
		ActingCharacterIds:   actingCharacters,
		ActingParticipantIds: actingParticipants,
		Slots:                slots,
	}
}

func sceneGMOutputToProto(interaction storage.SceneInteraction) *campaignv1.InteractionGMOutput {
	if strings.TrimSpace(interaction.GMOutputText) == "" {
		return nil
	}
	output := &campaignv1.InteractionGMOutput{
		Text:          interaction.GMOutputText,
		ParticipantId: interaction.GMOutputParticipantID,
	}
	if interaction.GMOutputUpdatedAt != nil {
		output.UpdatedAt = timestamppb.New(*interaction.GMOutputUpdatedAt)
	}
	return output
}

func scenePhaseStatusToProto(status scene.PlayerPhaseStatus) campaignv1.ScenePhaseStatus {
	switch status {
	case scene.PlayerPhaseStatusGMReview:
		return campaignv1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM_REVIEW
	case scene.PlayerPhaseStatusPlayers:
		return campaignv1.ScenePhaseStatus_SCENE_PHASE_STATUS_PLAYERS
	default:
		return campaignv1.ScenePhaseStatus_SCENE_PHASE_STATUS_PLAYERS
	}
}

func scenePlayerSlotReviewStatusToProto(status scene.PlayerPhaseSlotReviewStatus) campaignv1.ScenePlayerSlotReviewStatus {
	switch status {
	case scene.PlayerPhaseSlotReviewStatusUnderReview:
		return campaignv1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_UNDER_REVIEW
	case scene.PlayerPhaseSlotReviewStatusAccepted:
		return campaignv1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_ACCEPTED
	case scene.PlayerPhaseSlotReviewStatusChangesRequested:
		return campaignv1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_CHANGES_REQUESTED
	default:
		return campaignv1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_OPEN
	}
}
