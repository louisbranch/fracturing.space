package scene

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func decidePlayerPhaseStart(scenes map[ids.SceneID]State, cmd command.Command, now func() time.Time) command.Decision {
	var payload PlayerPhaseStartedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	sceneID := strings.TrimSpace(payload.SceneID.String())
	if sceneID == "" {
		sceneID = strings.TrimSpace(cmd.SceneID.String())
	}
	if sceneID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSceneIDRequired,
			Message: "scene id is required",
		})
	}
	if rejection := requireActiveScene(scenes, sceneID); rejection != nil {
		return command.Reject(*rejection)
	}
	current := scenes[ids.SceneID(sceneID)]
	if strings.TrimSpace(current.PlayerPhaseID) != "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeScenePlayerPhaseAlreadyOpen,
			Message: "scene player phase is already open",
		})
	}
	if strings.TrimSpace(payload.PhaseID) == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeScenePlayerPhaseIDRequired,
			Message: "player phase id is required",
		})
	}
	actingCharacterIDs := normalizeCharacterIDs(payload.ActingCharacterIDs)
	if len(actingCharacterIDs) == 0 {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSceneCharactersRequired,
			Message: "at least one acting character is required",
		})
	}
	for _, characterID := range actingCharacterIDs {
		if !current.Characters[characterID] {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterNotInScene,
				Message: "acting character is not in the scene",
			})
		}
	}
	actingParticipantIDs := normalizeParticipantIDs(payload.ActingParticipantIDs)
	if len(actingParticipantIDs) == 0 {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeScenePlayerPhaseParticipantRequired,
			Message: "at least one acting participant is required",
		})
	}

	payload.SceneID = ids.SceneID(sceneID)
	payload.FrameText = strings.TrimSpace(payload.FrameText)
	payload.ActingCharacterIDs = actingCharacterIDs
	payload.ActingParticipantIDs = actingParticipantIDs
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, EventTypePlayerPhaseStarted, "scene", sceneID, payloadJSON, now().UTC())
	evt.SceneID = ids.SceneID(sceneID)
	return command.Accept(evt)
}

func decidePlayerPhasePost(scenes map[ids.SceneID]State, cmd command.Command, now func() time.Time) command.Decision {
	current, sceneID, rejection := requireOpenPlayerPhase(scenes, cmd)
	if rejection != nil {
		return command.Reject(*rejection)
	}
	if current.PlayerPhaseStatus != PlayerPhaseStatusPlayers {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeScenePlayerPhaseNotInPlayersState,
			Message: "scene player phase is waiting for gm review",
		})
	}
	var payload PlayerPhasePostedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	participantID := ids.ParticipantID(strings.TrimSpace(payload.ParticipantID.String()))
	if participantID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeScenePlayerPhaseParticipantRequired,
			Message: "participant id is required",
		})
	}
	if !current.PlayerPhaseActingParticipants[participantID] {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeScenePlayerPhaseParticipantNotActing,
			Message: "participant is not acting in the current player phase",
		})
	}
	characterIDs := normalizeCharacterIDs(payload.CharacterIDs)
	if len(characterIDs) == 0 {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeSceneCharactersRequired,
			Message: "at least one acting character is required",
		})
	}
	for _, characterID := range characterIDs {
		if !slices.Contains(current.PlayerPhaseActingCharacters, characterID) {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterNotInScene,
				Message: "acting character is not part of the current player phase",
			})
		}
	}
	summaryText := strings.TrimSpace(payload.SummaryText)
	if summaryText == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeScenePlayerPhaseSummaryRequired,
			Message: "player phase summary is required",
		})
	}
	payload.SceneID = sceneID
	payload.PhaseID = current.PlayerPhaseID
	payload.ParticipantID = participantID
	payload.CharacterIDs = characterIDs
	payload.SummaryText = summaryText
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, EventTypePlayerPhasePosted, "scene", sceneID.String(), payloadJSON, now().UTC())
	evt.SceneID = sceneID
	return command.Accept(evt)
}

func decidePlayerPhaseYield(scenes map[ids.SceneID]State, cmd command.Command, now func() time.Time) command.Decision {
	current, sceneID, rejection := requireOpenPlayerPhase(scenes, cmd)
	if rejection != nil {
		return command.Reject(*rejection)
	}
	if current.PlayerPhaseStatus != PlayerPhaseStatusPlayers {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeScenePlayerPhaseNotInPlayersState,
			Message: "scene player phase is waiting for gm review",
		})
	}
	var payload PlayerPhaseYieldedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	participantID := ids.ParticipantID(strings.TrimSpace(payload.ParticipantID.String()))
	if participantID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeScenePlayerPhaseParticipantRequired,
			Message: "participant id is required",
		})
	}
	if !current.PlayerPhaseActingParticipants[participantID] {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeScenePlayerPhaseParticipantNotActing,
			Message: "participant is not acting in the current player phase",
		})
	}
	payload.SceneID = sceneID
	payload.PhaseID = current.PlayerPhaseID
	payload.ParticipantID = participantID
	payloadJSON, _ := json.Marshal(payload)
	yieldEvt := command.NewEvent(cmd, EventTypePlayerPhaseYielded, "scene", sceneID.String(), payloadJSON, now().UTC())
	yieldEvt.SceneID = sceneID
	if !allActingParticipantsYieldedAfter(current, participantID) {
		return command.Accept(yieldEvt)
	}
	reviewPayload := PlayerPhaseReviewStartedPayload{
		SceneID: sceneID,
		PhaseID: current.PlayerPhaseID,
	}
	reviewJSON, _ := json.Marshal(reviewPayload)
	reviewEvt := command.NewEvent(cmd, EventTypePlayerPhaseReviewStarted, "scene", sceneID.String(), reviewJSON, now().UTC())
	reviewEvt.SceneID = sceneID
	return command.Accept(yieldEvt, reviewEvt)
}

func decidePlayerPhaseUnyield(scenes map[ids.SceneID]State, cmd command.Command, now func() time.Time) command.Decision {
	current, sceneID, rejection := requireOpenPlayerPhase(scenes, cmd)
	if rejection != nil {
		return command.Reject(*rejection)
	}
	if current.PlayerPhaseStatus != PlayerPhaseStatusPlayers {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeScenePlayerPhaseNotInPlayersState,
			Message: "scene player phase is waiting for gm review",
		})
	}
	var payload PlayerPhaseUnyieldedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	participantID := ids.ParticipantID(strings.TrimSpace(payload.ParticipantID.String()))
	if participantID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeScenePlayerPhaseParticipantRequired,
			Message: "participant id is required",
		})
	}
	if !current.PlayerPhaseActingParticipants[participantID] {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeScenePlayerPhaseParticipantNotActing,
			Message: "participant is not acting in the current player phase",
		})
	}
	payload.SceneID = sceneID
	payload.PhaseID = current.PlayerPhaseID
	payload.ParticipantID = participantID
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, EventTypePlayerPhaseUnyielded, "scene", sceneID.String(), payloadJSON, now().UTC())
	evt.SceneID = sceneID
	return command.Accept(evt)
}

func decidePlayerPhaseAccept(scenes map[ids.SceneID]State, cmd command.Command, now func() time.Time) command.Decision {
	current, sceneID, rejection := requireOpenPlayerPhase(scenes, cmd)
	if rejection != nil {
		return command.Reject(*rejection)
	}
	if current.PlayerPhaseStatus != PlayerPhaseStatusGMReview {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeScenePlayerPhaseNotInReviewState,
			Message: "scene player phase is not waiting for gm review",
		})
	}
	payload := PlayerPhaseAcceptedPayload{
		SceneID: sceneID,
		PhaseID: current.PlayerPhaseID,
	}
	payloadJSON, _ := json.Marshal(payload)
	acceptedEvt := command.NewEvent(cmd, EventTypePlayerPhaseAccepted, "scene", sceneID.String(), payloadJSON, now().UTC())
	acceptedEvt.SceneID = sceneID
	endPayload := PlayerPhaseEndedPayload{
		SceneID: sceneID,
		PhaseID: current.PlayerPhaseID,
		Reason:  "accepted",
	}
	endJSON, _ := json.Marshal(endPayload)
	endEvt := command.NewEvent(cmd, EventTypePlayerPhaseEnded, "scene", sceneID.String(), endJSON, now().UTC())
	endEvt.SceneID = sceneID
	return command.Accept(acceptedEvt, endEvt)
}

func decidePlayerPhaseRequestRevisions(scenes map[ids.SceneID]State, cmd command.Command, now func() time.Time) command.Decision {
	current, sceneID, rejection := requireOpenPlayerPhase(scenes, cmd)
	if rejection != nil {
		return command.Reject(*rejection)
	}
	if current.PlayerPhaseStatus != PlayerPhaseStatusGMReview {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeScenePlayerPhaseNotInReviewState,
			Message: "scene player phase is not waiting for gm review",
		})
	}
	var payload PlayerPhaseRevisionsRequestedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	revisions := make([]PlayerPhaseRevisionRequest, 0, len(payload.Revisions))
	seenParticipants := make(map[ids.ParticipantID]struct{}, len(payload.Revisions))
	for _, revision := range payload.Revisions {
		participantID := ids.ParticipantID(strings.TrimSpace(revision.ParticipantID.String()))
		if participantID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeScenePlayerPhaseRevisionRequired,
				Message: "revision participant id is required",
			})
		}
		if !current.PlayerPhaseActingParticipants[participantID] {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeScenePlayerPhaseParticipantNotActing,
				Message: "revision participant is not acting in the current player phase",
			})
		}
		if _, exists := seenParticipants[participantID]; exists {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeScenePlayerPhaseRevisionRequired,
				Message: "revision participants must be unique",
			})
		}
		seenParticipants[participantID] = struct{}{}
		reason := strings.TrimSpace(revision.Reason)
		if reason == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeScenePlayerPhaseRevisionRequired,
				Message: "revision reason is required",
			})
		}
		characterIDs := normalizeCharacterIDs(revision.CharacterIDs)
		for _, characterID := range characterIDs {
			if !slices.Contains(current.PlayerPhaseActingCharacters, characterID) {
				return command.Reject(command.Rejection{
					Code:    rejectionCodeCharacterNotInScene,
					Message: "revision character is not part of the current player phase",
				})
			}
		}
		revisions = append(revisions, PlayerPhaseRevisionRequest{
			ParticipantID: participantID,
			Reason:        reason,
			CharacterIDs:  characterIDs,
		})
	}
	if len(revisions) == 0 {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeScenePlayerPhaseRevisionRequired,
			Message: "at least one revision request is required",
		})
	}
	payload.SceneID = sceneID
	payload.PhaseID = current.PlayerPhaseID
	payload.Revisions = revisions
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, EventTypePlayerPhaseRevisionsRequested, "scene", sceneID.String(), payloadJSON, now().UTC())
	evt.SceneID = sceneID
	return command.Accept(evt)
}

func decidePlayerPhaseEnd(scenes map[ids.SceneID]State, cmd command.Command, now func() time.Time) command.Decision {
	current, sceneID, rejection := requireOpenPlayerPhase(scenes, cmd)
	if rejection != nil {
		return command.Reject(*rejection)
	}
	var payload PlayerPhaseEndedPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{Code: command.RejectionCodePayloadDecodeFailed, Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err)})
	}
	payload.SceneID = sceneID
	payload.PhaseID = current.PlayerPhaseID
	payload.Reason = strings.TrimSpace(payload.Reason)
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, EventTypePlayerPhaseEnded, "scene", sceneID.String(), payloadJSON, now().UTC())
	evt.SceneID = sceneID
	return command.Accept(evt)
}

func requireOpenPlayerPhase(scenes map[ids.SceneID]State, cmd command.Command) (State, ids.SceneID, *command.Rejection) {
	sceneID := ids.SceneID(strings.TrimSpace(cmd.SceneID.String()))
	if sceneID == "" {
		return State{}, "", &command.Rejection{
			Code:    rejectionCodeSceneIDRequired,
			Message: "scene id is required",
		}
	}
	current, ok := scenes[sceneID]
	if !ok {
		return State{}, "", &command.Rejection{
			Code:    rejectionCodeSceneNotFound,
			Message: "scene not found",
		}
	}
	if !current.Active {
		return State{}, "", &command.Rejection{
			Code:    rejectionCodeSceneNotActive,
			Message: "scene is not active",
		}
	}
	if strings.TrimSpace(current.PlayerPhaseID) == "" {
		return State{}, "", &command.Rejection{
			Code:    rejectionCodeScenePlayerPhaseNotOpen,
			Message: "scene player phase is not open",
		}
	}
	return current, sceneID, nil
}

func allActingParticipantsYieldedAfter(current State, next ids.ParticipantID) bool {
	if len(current.PlayerPhaseActingParticipants) == 0 {
		return false
	}
	for participantID := range current.PlayerPhaseActingParticipants {
		slot, ok := current.PlayerPhaseSlots[participantID]
		if participantID == next {
			if !ok || slot.Yielded {
				continue
			}
			continue
		}
		if !ok || !slot.Yielded {
			return false
		}
	}
	return true
}
