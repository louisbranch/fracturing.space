package participant

import (
	"encoding/json"
	"fmt"
	"testing"

	assetcatalog "github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
)

func TestDecideParticipantJoin_DefaultAvatarUsesUserIDWhenPresent(t *testing.T) {
	manifest := assetcatalog.AvatarManifest()
	expectedSetID, expectedAssetID, err := manifest.ResolveSelection(assetcatalog.SelectionInput{
		EntityType: "user",
		EntityID:   "user-42",
	})
	if err != nil {
		t.Fatalf("resolve expected user avatar: %v", err)
	}

	for _, participantID := range []string{"p-1", "p-2", "p-3", "p-4", "p-5"} {
		cmd := command.Command{
			CampaignID: "camp-1",
			Type:       CommandTypeJoin,
			ActorType:  command.ActorTypeSystem,
			PayloadJSON: []byte(fmt.Sprintf(
				`{"participant_id":"%s","user_id":"user-42","name":"Alice","role":"PLAYER"}`,
				participantID,
			)),
		}

		decision := Decide(State{}, cmd, nil)
		if len(decision.Rejections) != 0 {
			t.Fatalf("participant %s rejections: %v", participantID, decision.Rejections)
		}
		if len(decision.Events) != 1 {
			t.Fatalf("participant %s event count = %d, want 1", participantID, len(decision.Events))
		}

		var payload JoinPayload
		if err := json.Unmarshal(decision.Events[0].PayloadJSON, &payload); err != nil {
			t.Fatalf("participant %s payload: %v", participantID, err)
		}
		if payload.AvatarSetID != expectedSetID {
			t.Fatalf(
				"participant %s avatar set = %q, want %q",
				participantID,
				payload.AvatarSetID,
				expectedSetID,
			)
		}
		if payload.AvatarAssetID != expectedAssetID {
			t.Fatalf(
				"participant %s avatar asset = %q, want %q",
				participantID,
				payload.AvatarAssetID,
				expectedAssetID,
			)
		}
	}
}
