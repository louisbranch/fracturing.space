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
