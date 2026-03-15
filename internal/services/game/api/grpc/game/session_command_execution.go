package game

import (
	"context"
	"encoding/json"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
)

// sessionCommandExecutionInput captures the transport-owned write metadata for a
// session aggregate command so lifecycle and spotlight flows do not rebuild the
// domain-write path inline.
type sessionCommandExecutionInput struct {
	CommandType command.Type
	CampaignID  string
	SessionID   string
	Payload     any
	Options     domainwrite.Options
}

// sessionCommandExecutor centralizes the canonical session write path for
// lifecycle and spotlight commands.
type sessionCommandExecutor struct {
	write   domainwriteexec.WritePath
	applier projection.Applier
}

func newSessionCommandExecutor(write domainwriteexec.WritePath, applier projection.Applier) sessionCommandExecutor {
	return sessionCommandExecutor{
		write:   write,
		applier: applier,
	}
}

func (e sessionCommandExecutor) Execute(ctx context.Context, input sessionCommandExecutionInput) error {
	payloadJSON, err := json.Marshal(input.Payload)
	if err != nil {
		return grpcerror.Internal("encode payload", err)
	}

	actorID, actorType := handler.ResolveCommandActor(ctx)
	_, err = handler.ExecuteAndApplyDomainCommand(
		ctx,
		e.write,
		e.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   input.CampaignID,
			Type:         input.CommandType,
			ActorType:    actorType,
			ActorID:      actorID,
			SessionID:    input.SessionID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "session",
			EntityID:     input.SessionID,
			PayloadJSON:  payloadJSON,
		}),
		input.Options,
	)
	return err
}
