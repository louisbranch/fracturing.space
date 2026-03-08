package daggerheart

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
)

func (s *DaggerheartService) executeAndApplyDomainCommand(
	ctx context.Context,
	cmd command.Command,
	applier domainwrite.EventApplier,
	options domainwrite.Options,
) (engine.Result, error) {
	result, err := domainwriteexec.ExecuteAndApply(
		ctx,
		s.stores.Write,
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
