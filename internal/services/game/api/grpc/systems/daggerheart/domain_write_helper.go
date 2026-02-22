package daggerheart

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

var writeRuntime = domainwrite.NewRuntime()

type eventApplier interface {
	Apply(context.Context, event.Event) error
}

type domainCommandApplyOptions struct {
	requireEvents   bool
	missingEventMsg string
	applyErrMessage string
	executeErrMsg   string
	applyErr        func(error) error
	executeErr      func(error) error
	rejectErr       func(string) error
}

// SetInlineProjectionApplyEnabled controls whether request-path helpers apply
// emitted domain events to projections inline.
func SetInlineProjectionApplyEnabled(enabled bool) {
	writeRuntime.SetInlineApplyEnabled(enabled)
}

// SetIntentFilter configures the event intent filter built from the event
// registry. Call this once at server startup; the filter is used by every
// request-path domain command helper.
func SetIntentFilter(registry *event.Registry) {
	writeRuntime.SetIntentFilter(registry)
}

func (s *DaggerheartService) executeAndApplyDomainCommand(
	ctx context.Context,
	cmd command.Command,
	applier eventApplier,
	options domainCommandApplyOptions,
) (engine.Result, error) {
	options = normalizeDomainCommandOptions(options)
	return writeRuntime.ExecuteAndApply(ctx, s.stores.Domain, applier, cmd, domainwrite.Options{
		RequireEvents:   options.requireEvents,
		MissingEventMsg: options.missingEventMsg,
		ExecuteErr:      options.executeErr,
		ApplyErr:        options.applyErr,
		RejectErr:       options.rejectErr,
	})
}

func normalizeDomainCommandOptions(options domainCommandApplyOptions) domainCommandApplyOptions {
	executeErr, applyErr, rejectErr := domainwrite.NormalizeErrorHandlers(domainwrite.ErrorHandlerOptions{
		ExecuteErr:        options.executeErr,
		ApplyErr:          options.applyErr,
		RejectErr:         options.rejectErr,
		ExecuteErrMessage: options.executeErrMsg,
		ApplyErrMessage:   options.applyErrMessage,
	})
	options.executeErr = executeErr
	options.applyErr = applyErr
	options.rejectErr = rejectErr
	return options
}
