package handler

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// CommandActorTypeForEventActor maps event actor types to command actor types.
func CommandActorTypeForEventActor(actorType event.ActorType) command.ActorType {
	switch actorType {
	case event.ActorTypeParticipant:
		return command.ActorTypeParticipant
	case event.ActorTypeGM:
		return command.ActorTypeGM
	default:
		return command.ActorTypeSystem
	}
}
