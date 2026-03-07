package domainwriteexec

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
)

// Deps provides the domain execution/runtime dependencies consumed by
// transport write helpers.
type Deps interface {
	DomainExecutor() domainwrite.Executor
	DomainWriteRuntime() *domainwrite.Runtime
}

// ExecuteAndApply normalizes transport write options and executes one command
// using runtime-controlled inline apply behavior.
func ExecuteAndApply(
	ctx context.Context,
	deps Deps,
	applier domainwrite.EventApplier,
	cmd command.Command,
	options domainwrite.Options,
	normalizeConfig grpcerror.NormalizeDomainWriteOptionsConfig,
) (engine.Result, error) {
	grpcerror.NormalizeDomainWriteOptions(&options, normalizeConfig)
	runtime := deps.DomainWriteRuntime()
	if runtime == nil {
		runtime = domainwrite.NewRuntime()
	}
	return runtime.ExecuteAndApply(ctx, deps.DomainExecutor(), applier, cmd, options)
}

// ExecuteWithoutInlineApply normalizes transport write options and executes one
// command while forcing projection apply to happen out-of-band.
func ExecuteWithoutInlineApply(
	ctx context.Context,
	deps Deps,
	cmd command.Command,
	options domainwrite.Options,
	normalizeConfig grpcerror.NormalizeDomainWriteOptionsConfig,
) (engine.Result, error) {
	grpcerror.NormalizeDomainWriteOptions(&options, normalizeConfig)
	runtime := deps.DomainWriteRuntime()
	if runtime == nil {
		runtime = domainwrite.NewRuntime()
	}
	return runtime.ExecuteWithoutInlineApply(ctx, deps.DomainExecutor(), cmd, options)
}
