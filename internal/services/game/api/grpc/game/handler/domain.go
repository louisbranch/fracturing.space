package handler

import (
	"context"
	"strings"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
)

// Domain executes domain commands and returns the result.
type Domain interface {
	Execute(ctx context.Context, cmd command.Command) (engine.Result, error)
}

// ResolveCommandActor returns the actor identity and type from gRPC context metadata.
// Returns ActorTypeParticipant when a participant ID is present, ActorTypeSystem otherwise.
func ResolveCommandActor(ctx context.Context) (string, command.ActorType) {
	actorID := strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
	if actorID != "" {
		return actorID, command.ActorTypeParticipant
	}
	return "", command.ActorTypeSystem
}
