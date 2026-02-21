package game

import (
	"context"
	"strings"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
)

// Domain executes domain commands and returns the result.
type Domain interface {
	Execute(ctx context.Context, cmd command.Command) (engine.Result, error)
}

// resolveCommandActor returns the actor identity and type from gRPC context metadata.
// Returns ActorTypeParticipant when a participant ID is present, ActorTypeSystem otherwise.
func resolveCommandActor(ctx context.Context) (string, command.ActorType) {
	actorID := strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
	if actorID != "" {
		return actorID, command.ActorTypeParticipant
	}
	return "", command.ActorTypeSystem
}
