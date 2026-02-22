package daggerheart

import (
	"context"
	"sync/atomic"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

var (
	inlineProjectionApplyEnabled atomic.Bool

	// intentFilter is the event intent filter used to decide which emitted events
	// should be applied inline to projections. Set once at startup via
	// SetIntentFilter; defaults to fail-closed (no events applied).
	intentFilter = func(event.Event) bool { return false }
)

func init() {
	inlineProjectionApplyEnabled.Store(true)
}

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
	inlineProjectionApplyEnabled.Store(enabled)
}

// SetIntentFilter configures the event intent filter built from the event
// registry. Call this once at server startup; the filter is used by every
// request-path domain command helper.
func SetIntentFilter(registry *event.Registry) {
	intentFilter = domainwrite.NewIntentFilter(registry)
}

func (s *DaggerheartService) executeAndApplyDomainCommand(
	ctx context.Context,
	cmd command.Command,
	applier eventApplier,
	options domainCommandApplyOptions,
) (engine.Result, error) {
	options = normalizeDomainCommandOptions(options)
	return domainwrite.ExecuteAndApply(ctx, s.stores.Domain, applier, cmd, domainwrite.Options{
		RequireEvents:      options.requireEvents,
		MissingEventMsg:    options.missingEventMsg,
		InlineApplyEnabled: inlineProjectionApplyEnabled.Load(),
		ShouldApply:        intentFilter,
		ExecuteErr:         options.executeErr,
		ApplyErr:           options.applyErr,
		RejectErr:          options.rejectErr,
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
