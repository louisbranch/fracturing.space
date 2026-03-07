package game

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
)

type domainWriteDeps = domainwriteexec.Deps

func executeAndApplyDomainCommand(
	ctx context.Context,
	deps domainWriteDeps,
	applier projection.Applier,
	cmd command.Command,
	options domainwrite.Options,
) (engine.Result, error) {
	result, err := domainwriteexec.ExecuteAndApply(
		ctx,
		deps,
		applier,
		cmd,
		options,
		grpcerror.NormalizeDomainWriteOptionsConfig{},
	)
	if err != nil {
		return result, grpcerror.EnsureStatus(err)
	}
	return result, nil
}

func executeDomainCommandWithoutInlineApply(
	ctx context.Context,
	deps domainWriteDeps,
	cmd command.Command,
	options domainwrite.Options,
) (engine.Result, error) {
	result, err := domainwriteexec.ExecuteWithoutInlineApply(
		ctx,
		deps,
		cmd,
		options,
		grpcerror.NormalizeDomainWriteOptionsConfig{},
	)
	if err != nil {
		return result, grpcerror.EnsureStatus(err)
	}
	return result, nil
}

func domainApplyErrorWithCodePreserve(message string) func(error) error {
	return grpcerror.ApplyErrorWithDomainCodePreserve(message)
}
