// Package handler provides shared handler utilities used across entity-scoped
// transport subpackages: domain write helpers, pagination, mappers, actor
// resolution, social profile loading, and other cross-cutting handler concerns.
package handler

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
)

// DomainWriteDeps is the dependency bundle for domain write execution.
type DomainWriteDeps = domainwriteexec.Deps

// ExecuteAndApplyDomainCommand executes a domain command and applies the
// resulting events inline when enabled.
func ExecuteAndApplyDomainCommand(
	ctx context.Context,
	deps DomainWriteDeps,
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

// ExecuteWithoutInlineApply executes a domain command without inline event
// application.
func ExecuteWithoutInlineApply(
	ctx context.Context,
	deps DomainWriteDeps,
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

// ApplyErrorWithCodePreserve returns an error wrapper that maps apply errors
// while preserving domain status codes.
func ApplyErrorWithCodePreserve(message string) func(error) error {
	return grpcerror.ApplyErrorWithDomainCodePreserve(message)
}
