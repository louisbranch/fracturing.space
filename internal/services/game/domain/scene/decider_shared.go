package scene

import (
	"fmt"
	"slices"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

// requireActiveScene validates that the scene exists and is active.
func requireActiveScene(scenes map[ids.SceneID]State, sceneID string) *command.Rejection {
	scene, ok := scenes[ids.SceneID(sceneID)]
	if !ok {
		return &command.Rejection{Code: rejectionCodeSceneNotFound, Message: "scene not found"}
	}
	if !scene.Active {
		return &command.Rejection{Code: rejectionCodeSceneNotActive, Message: "scene is not active"}
	}
	return nil
}

func rejectPayloadDecode(cmd command.Command, err error) command.Decision {
	return command.Reject(command.Rejection{
		Code:    command.RejectionCodePayloadDecodeFailed,
		Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
	})
}

// normalizeCharacterIDs trims, deduplicates, and filters empty character IDs.
func normalizeCharacterIDs(charIDs []ids.CharacterID) []ids.CharacterID {
	seen := make(map[ids.CharacterID]bool, len(charIDs))
	result := make([]ids.CharacterID, 0, len(charIDs))
	for _, id := range charIDs {
		trimmed := ids.CharacterID(strings.TrimSpace(id.String()))
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		result = append(result, trimmed)
	}
	return result
}

// sortedCharacterIDs returns a stable-sorted slice of character IDs from a map.
func sortedCharacterIDs(chars map[ids.CharacterID]bool) []ids.CharacterID {
	strs := make([]string, 0, len(chars))
	for id := range chars {
		strs = append(strs, string(id))
	}
	// Sort for deterministic event order in replay.
	slices.Sort(strs)
	result := make([]ids.CharacterID, 0, len(strs))
	for _, s := range strs {
		result = append(result, ids.CharacterID(s))
	}
	return result
}

func normalizeParticipantIDs(participantIDs []ids.ParticipantID) []ids.ParticipantID {
	seen := make(map[ids.ParticipantID]bool, len(participantIDs))
	result := make([]ids.ParticipantID, 0, len(participantIDs))
	for _, id := range participantIDs {
		trimmed := ids.ParticipantID(strings.TrimSpace(id.String()))
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		result = append(result, trimmed)
	}
	return result
}

func normalizeGMInteractionBeats(beats []GMInteractionBeat) ([]GMInteractionBeat, *command.Rejection) {
	result := make([]GMInteractionBeat, 0, len(beats))
	seen := make(map[string]bool, len(beats))
	for _, beat := range beats {
		beatID := strings.TrimSpace(beat.BeatID)
		if beatID == "" || seen[beatID] {
			return nil, &command.Rejection{
				Code:    rejectionCodeSceneGMInteractionBeatRequired,
				Message: "gm interaction beat id is required and must be unique",
			}
		}
		seen[beatID] = true
		text := strings.TrimSpace(beat.Text)
		if text == "" {
			return nil, &command.Rejection{
				Code:    rejectionCodeSceneGMInteractionBeatTextRequired,
				Message: "gm interaction beat text is required",
			}
		}
		switch GMInteractionBeatType(strings.TrimSpace(string(beat.Type))) {
		case GMInteractionBeatTypeFiction,
			GMInteractionBeatTypePrompt,
			GMInteractionBeatTypeResolution,
			GMInteractionBeatTypeConsequence,
			GMInteractionBeatTypeGuidance:
		default:
			return nil, &command.Rejection{
				Code:    rejectionCodeSceneGMInteractionBeatTypeRequired,
				Message: "gm interaction beat type is required",
			}
		}
		result = append(result, GMInteractionBeat{
			BeatID: beatID,
			Type:   GMInteractionBeatType(strings.TrimSpace(string(beat.Type))),
			Text:   text,
		})
	}
	if len(result) == 0 {
		return nil, &command.Rejection{
			Code:    rejectionCodeSceneGMInteractionBeatRequired,
			Message: "at least one gm interaction beat is required",
		}
	}
	return result, nil
}
