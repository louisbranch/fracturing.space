package scene

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func TestDecidePlayerPhaseRequestRevisionsRequiresReason(t *testing.T) {
	t.Parallel()

	state := activeScene("scene-1", "char-1")
	state.PlayerPhaseID = "phase-1"
	state.PlayerPhaseStatus = PlayerPhaseStatusGMReview
	state.PlayerPhaseActingCharacters = []ids.CharacterID{"char-1"}
	state.PlayerPhaseActingParticipants = map[ids.ParticipantID]bool{"player-1": true}

	request := cmd(CommandTypePlayerPhaseRequestRevisions, `{
		"scene_id": "scene-1",
		"phase_id": "phase-1",
		"revisions": [{"participant_id":"player-1","reason":" ","character_ids":["char-1"]}]
	}`)
	request.SceneID = "scene-1"

	decision := Decide(scenesMap(state), request, nowFunc)

	requireRejected(t, decision, rejectionCodeScenePlayerPhaseRevisionRequired)
}
