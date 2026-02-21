package domainwrite

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Executor executes a domain command and returns the domain result.
type Executor interface {
	Execute(context.Context, command.Command) (engine.Result, error)
}

// EventApplier applies emitted events into projection stores.
type EventApplier interface {
	Apply(context.Context, event.Event) error
}

// Options controls command execution and emitted-event application behavior.
type Options struct {
	RequireEvents      bool
	MissingEventMsg    string
	InlineApplyEnabled bool
	ShouldApply        func(event.Event) bool
	ExecuteErr         func(error) error
	ApplyErr           func(error) error
	RejectErr          func(string) error
}

// ExecuteAndApply executes the command, handles rejections, and applies events.
func ExecuteAndApply(
	ctx context.Context,
	executor Executor,
	applier EventApplier,
	cmd command.Command,
	options Options,
) (engine.Result, error) {
	options = normalizeOptions(options)

	if executor == nil {
		return engine.Result{}, status.Error(codes.Internal, "domain engine is not configured")
	}

	result, err := executor.Execute(ctx, cmd)
	if err != nil {
		return engine.Result{}, options.ExecuteErr(err)
	}
	if len(result.Decision.Rejections) > 0 {
		return engine.Result{}, options.RejectErr(result.Decision.Rejections[0].Message)
	}
	if options.RequireEvents && len(result.Decision.Events) == 0 {
		return engine.Result{}, status.Error(codes.Internal, options.MissingEventMsg)
	}
	if options.InlineApplyEnabled {
		for _, evt := range result.Decision.Events {
			if options.ShouldApply != nil && !options.ShouldApply(evt) {
				continue
			}
			if err := applier.Apply(ctx, evt); err != nil {
				return engine.Result{}, options.ApplyErr(err)
			}
		}
	}

	return result, nil
}

// ShouldApplyProjectionInline enforces request-path inline-apply policy.
//
// Action roll/outcome records and story notes are journal-only and are not
// projected via projection.Applier in request handlers.
func ShouldApplyProjectionInline(evt event.Event) bool {
	switch evt.Type {
	case event.Type("action.roll_resolved"),
		event.Type("action.outcome_applied"),
		event.Type("action.outcome_rejected"),
		event.Type("story.note_added"):
		return false
	default:
		return true
	}
}

func normalizeOptions(options Options) Options {
	if options.ExecuteErr == nil {
		options.ExecuteErr = func(err error) error {
			return status.Errorf(codes.Internal, "execute domain command: %v", err)
		}
	}
	if options.ApplyErr == nil {
		options.ApplyErr = func(err error) error {
			return status.Errorf(codes.Internal, "apply event: %v", err)
		}
	}
	if options.RejectErr == nil {
		options.RejectErr = func(message string) error {
			return status.Error(codes.FailedPrecondition, message)
		}
	}
	return options
}
