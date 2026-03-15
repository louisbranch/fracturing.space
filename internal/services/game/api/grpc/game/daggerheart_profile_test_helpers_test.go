package game

import (
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func testDaggerheartProfile(overrides func(*daggerheart.CharacterProfile)) daggerheart.CharacterProfile {
	profile := daggerheart.CharacterProfile{
		Level:           1,
		HpMax:           6,
		StressMax:       6,
		Evasion:         10,
		MajorThreshold:  1,
		SevereThreshold: 2,
		Proficiency:     1,
		ArmorScore:      0,
		ArmorMax:        0,
	}
	if overrides != nil {
		overrides(&profile)
	}
	return profile
}

func testDaggerheartProfileReplacedEvent(
	t *testing.T,
	now time.Time,
	campaignID, characterID string,
	actorType event.ActorType,
	actorID string,
	profile daggerheart.CharacterProfile,
) event.Event {
	t.Helper()
	return event.Event{
		CampaignID:    ids.CampaignID(campaignID),
		Type:          daggerheart.EventTypeCharacterProfileReplaced,
		Timestamp:     now,
		ActorType:     actorType,
		ActorID:       actorID,
		EntityType:    "character",
		EntityID:      characterID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON: mustJSON(t, daggerheart.CharacterProfileReplacedPayload{
			CharacterID: ids.CharacterID(characterID),
			Profile:     profile,
		}),
	}
}

func testCreateCharacterResults(
	t *testing.T,
	now time.Time,
	campaignID, characterID string,
	actorType event.ActorType,
	actorID string,
	createPayload any,
) map[command.Type]engine.Result {
	t.Helper()
	return map[command.Type]engine.Result{
		handler.CommandTypeCharacterCreate: {
			Decision: command.Accept(event.Event{
				CampaignID:  ids.CampaignID(campaignID),
				Type:        event.Type("character.created"),
				Timestamp:   now,
				ActorType:   actorType,
				ActorID:     actorID,
				EntityType:  "character",
				EntityID:    characterID,
				PayloadJSON: mustJSON(t, createPayload),
			}),
		},
	}
}
