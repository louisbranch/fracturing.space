package workflowwrite

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowruntime"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

// ExecuteDomainCommand executes one Daggerheart system-domain command through
// the shared workflow write path.
func ExecuteDomainCommand(
	ctx context.Context,
	deps domainwrite.Deps,
	store projectionstore.Store,
	in DomainCommandInput,
) error {
	_, err := ExecuteAndApply(
		ctx,
		deps,
		daggerheart.NewAdapter(store),
		commandbuild.SystemCommand(commandbuild.SystemCommandInput{
			CampaignID:    in.CampaignID,
			Type:          in.CommandType,
			SystemID:      daggerheart.SystemID,
			SystemVersion: daggerheart.SystemVersion,
			SessionID:     in.SessionID,
			SceneID:       in.SceneID,
			RequestID:     in.RequestID,
			InvocationID:  in.InvocationID,
			EntityType:    in.EntityType,
			EntityID:      in.EntityID,
			PayloadJSON:   in.PayloadJSON,
		}),
		domainwrite.RequireEventsWithDiagnostics(in.MissingEventMsg, in.ApplyErrMessage),
	)
	return err
}

// ExecuteCoreCommand executes one core-domain system-actor command through the
// shared workflow write path and returns the resulting decision.
func ExecuteCoreCommand(
	ctx context.Context,
	deps domainwrite.Deps,
	applier domainwrite.EventApplier,
	in CoreCommandInput,
) (engine.Result, error) {
	return ExecuteAndApply(
		ctx,
		deps,
		applier,
		commandbuild.CoreSystem(commandbuild.CoreSystemInput{
			CampaignID:    in.CampaignID,
			Type:          in.CommandType,
			SessionID:     in.SessionID,
			SceneID:       in.SceneID,
			RequestID:     in.RequestID,
			InvocationID:  in.InvocationID,
			CorrelationID: in.CorrelationID,
			EntityType:    in.EntityType,
			EntityID:      in.EntityID,
			PayloadJSON:   in.PayloadJSON,
		}),
		domainwrite.RequireEventsWithDiagnostics(in.MissingEventMsg, in.ApplyErrMessage),
	)
}

// ExecuteSystemCommand executes one Daggerheart system command using the shared
// workflow runtime.
func ExecuteSystemCommand(
	ctx context.Context,
	deps domainwrite.Deps,
	eventStore workflowruntime.EventStore,
	store projectionstore.Store,
	in workflowruntime.SystemCommandInput,
) error {
	return NewRuntime(deps, eventStore, store).ExecuteSystemCommand(ctx, in)
}
