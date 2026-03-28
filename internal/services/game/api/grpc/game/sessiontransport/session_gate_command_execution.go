package sessiontransport

import (
	"context"
	"encoding/json"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
)

// sessionGateCommandExecutor centralizes the canonical session-gate write path
// so session- and communication-owned handlers share one orchestration seam.
type sessionGateCommandExecutor struct {
	write   domainwrite.WritePath
	applier projection.Applier
}

func newSessionGateCommandExecutor(write domainwrite.WritePath, applier projection.Applier) sessionGateCommandExecutor {
	return sessionGateCommandExecutor{
		write:   write,
		applier: applier,
	}
}

func (e sessionGateCommandExecutor) Execute(
	ctx context.Context,
	commandType command.Type,
	campaignID string,
	sessionID string,
	gateID string,
	payload any,
	requireEventsLabel string,
) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return grpcerror.Internal("encode payload", err)
	}

	actorID, actorType := handler.ResolveCommandActor(ctx)
	_, err = handler.ExecuteAndApplyDomainCommand(
		ctx,
		e.write,
		e.applier,
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
	return err
}

func executeSessionGateCommandAndLoad[T any](
	ctx context.Context,
	executor sessionGateCommandExecutor,
	commandType command.Type,
	campaignID string,
	sessionID string,
	gateID string,
	payload any,
	requireEventsLabel string,
	load func(context.Context) (T, error),
) (T, error) {
	var zero T

	if err := executor.Execute(
		ctx,
		commandType,
		campaignID,
		sessionID,
		gateID,
		payload,
		requireEventsLabel,
	); err != nil {
		return zero, err
	}

	return load(ctx)
}
