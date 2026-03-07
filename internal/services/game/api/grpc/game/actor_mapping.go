package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func commandActorTypeForEventActor(actorType event.ActorType) command.ActorType {
	switch actorType {
	case event.ActorTypeParticipant:
		return command.ActorTypeParticipant
	case event.ActorTypeGM:
		return command.ActorTypeGM
	default:
		return command.ActorTypeSystem
	}
}
