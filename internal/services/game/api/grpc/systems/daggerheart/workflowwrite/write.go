package workflowwrite

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowruntime"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
)

// ExecuteAndApply runs one command through the shared domain-write helper using
// the Daggerheart transport policy for preserving domain codes on apply
// failures.
func ExecuteAndApply(
	ctx context.Context,
	deps domainwriteexec.Deps,
	applier domainwrite.EventApplier,
	cmd command.Command,
	options domainwrite.Options,
) (engine.Result, error) {
	result, err := domainwriteexec.ExecuteAndApply(
		ctx,
		deps,
		applier,
		cmd,
		options,
		grpcerror.NormalizeDomainWriteOptionsConfig{
			PreserveDomainCodeOnApply: true,
		},
	)
	if err != nil {
		return result, grpcerror.EnsureStatus(err)
	}
	return result, nil
}

// NewRuntime builds the shared workflow runtime backed by the provided write
// path and Daggerheart stores.
func NewRuntime(
	deps domainwriteexec.Deps,
	eventStore workflowruntime.EventStore,
	daggerheartStore projectionstore.Store,
) *workflowruntime.Runtime {
	return workflowruntime.New(workflowruntime.Dependencies{
		Event:       eventStore,
		Daggerheart: daggerheartStore,
		ExecuteDomainCommand: func(ctx context.Context, cmd command.Command, applier domainwrite.EventApplier, options domainwrite.Options) error {
			_, err := ExecuteAndApply(ctx, deps, applier, cmd, options)
			return err
		},
	})
}
