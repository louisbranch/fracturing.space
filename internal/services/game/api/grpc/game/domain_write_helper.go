package game

import (
	"context"
	"sync/atomic"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

type domainCommandApplyOptions struct {
	requireEvents     bool
	missingEventMsg   string
	applyErr          func(error) error
	executeErr        func(error) error
	rejectErr         func(string) error
	executeErrMessage string
	applyErrMessage   string
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

func executeAndApplyDomainCommand(
	ctx context.Context,
	domain Domain,
	applier projection.Applier,
	cmd command.Command,
	options domainCommandApplyOptions,
) (engine.Result, error) {
	options = normalizeDomainCommandOptions(options)
	return domainwrite.ExecuteAndApply(ctx, domain, applier, cmd, domainwrite.Options{
		RequireEvents:      options.requireEvents,
		MissingEventMsg:    options.missingEventMsg,
		InlineApplyEnabled: inlineProjectionApplyEnabled.Load(),
		ShouldApply:        intentFilter,
		ExecuteErr:         options.executeErr,
		ApplyErr:           options.applyErr,
		RejectErr:          options.rejectErr,
	})
}

func executeDomainCommandWithoutInlineApply(
	ctx context.Context,
	domain Domain,
	cmd command.Command,
	options domainCommandApplyOptions,
) (engine.Result, error) {
	options = normalizeDomainCommandOptions(options)
	return domainwrite.ExecuteAndApply(ctx, domain, projection.Applier{}, cmd, domainwrite.Options{
		RequireEvents:      options.requireEvents,
		MissingEventMsg:    options.missingEventMsg,
		InlineApplyEnabled: false,
		ShouldApply:        nil,
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
		ExecuteErrMessage: options.executeErrMessage,
		ApplyErrMessage:   options.applyErrMessage,
	})
	options.executeErr = executeErr
	options.applyErr = applyErr
	options.rejectErr = rejectErr
	return options
}

func domainApplyErrorWithCodePreserve(message string) func(error) error {
	return func(err error) error {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return err
		}
		return status.Errorf(codes.Internal, "%s: %v", message, err)
	}
}
