package game

import (
	"context"
	"encoding/json"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
)

// executeSessionGateCommandAndLoad centralizes the canonical session-gate write
// path so communication and session transport do not drift in payload
// marshalling, actor attribution, or post-write load behavior.
func executeSessionGateCommandAndLoad[T any](
	ctx context.Context,
	write domainwriteexec.WritePath,
	applier projection.Applier,
	commandType command.Type,
	campaignID string,
	sessionID string,
	gateID string,
	payload any,
	requireEventsLabel string,
	load func(context.Context) (T, error),
) (T, error) {
	var zero T

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return zero, grpcerror.Internal("encode payload", err)
	}

	actorID, actorType := resolveCommandActor(ctx)
	_, err = executeAndApplyDomainCommand(
		ctx,
		write,
		applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandType,
			ActorType:    actorType,
			ActorID:      actorID,
			SessionID:    sessionID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "session_gate",
			EntityID:     gateID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.RequireEvents(requireEventsLabel+" did not emit an event"),
	)
	if err != nil {
		return zero, err
	}

	return load(ctx)
}
