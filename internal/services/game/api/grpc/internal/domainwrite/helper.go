package domainwrite

import (
	"context"
	"sync/atomic"

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

// ErrorHandlerOptions controls how ExecuteAndApply error callbacks are
// normalized when callers want shared default behavior with optional overrides.
type ErrorHandlerOptions struct {
	ExecuteErr        func(error) error
	ApplyErr          func(error) error
	RejectErr         func(string) error
	ExecuteErrMessage string
	ApplyErrMessage   string
}

// Runtime owns request-path write execution flags shared by service helpers.
type Runtime struct {
	inlineApplyEnabled atomic.Bool
	shouldApply        atomic.Value // stores func(event.Event) bool
}

// NewRuntime creates a runtime with inline apply enabled and fail-closed
// intent filtering.
func NewRuntime() *Runtime {
	runtime := &Runtime{}
	runtime.inlineApplyEnabled.Store(true)
	runtime.shouldApply.Store(func(event.Event) bool { return false })
	return runtime
}

// SetInlineApplyEnabled sets whether write helpers apply events inline.
func (r *Runtime) SetInlineApplyEnabled(enabled bool) {
	if r == nil {
		return
	}
	r.inlineApplyEnabled.Store(enabled)
}

// SetShouldApply sets the runtime event intent filter.
func (r *Runtime) SetShouldApply(filter func(event.Event) bool) {
	if r == nil {
		return
	}
	if filter == nil {
		filter = func(event.Event) bool { return false }
	}
	r.shouldApply.Store(filter)
}

// SetIntentFilter sets the runtime event intent filter from the registry.
func (r *Runtime) SetIntentFilter(registry *event.Registry) {
	r.SetShouldApply(NewIntentFilter(registry))
}

// ShouldApply returns the runtime event intent filter.
func (r *Runtime) ShouldApply() func(event.Event) bool {
	if r == nil {
		return func(event.Event) bool { return false }
	}
	filter, ok := r.shouldApply.Load().(func(event.Event) bool)
	if !ok || filter == nil {
		return func(event.Event) bool { return false }
	}
	return filter
}

// InlineApplyEnabled reports whether inline apply is enabled.
func (r *Runtime) InlineApplyEnabled() bool {
	if r == nil {
		return false
	}
	return r.inlineApplyEnabled.Load()
}

// ExecuteAndApply executes a command using runtime apply configuration.
func (r *Runtime) ExecuteAndApply(
	ctx context.Context,
	executor Executor,
	applier EventApplier,
	cmd command.Command,
	options Options,
) (engine.Result, error) {
	options.InlineApplyEnabled = r.InlineApplyEnabled()
	if options.ShouldApply == nil {
		options.ShouldApply = r.ShouldApply()
	}
	return ExecuteAndApply(ctx, executor, applier, cmd, options)
}

// ExecuteWithoutInlineApply executes a command without applying projections.
func (r *Runtime) ExecuteWithoutInlineApply(
	ctx context.Context,
	executor Executor,
	cmd command.Command,
	options Options,
) (engine.Result, error) {
	options.InlineApplyEnabled = false
	options.ShouldApply = nil
	return ExecuteAndApply(ctx, executor, nilEventApplier{}, cmd, options)
}

type nilEventApplier struct{}

func (nilEventApplier) Apply(context.Context, event.Event) error {
	return nil
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

// NewIntentFilter returns a filter function that checks event intent against
// the provided registry, skipping audit-only and replay-only events. If the
// registry is nil or the event type is unknown, the filter fails closed
// (returns false).
func NewIntentFilter(registry *event.Registry) func(event.Event) bool {
	if registry == nil {
		return func(_ event.Event) bool { return false }
	}
	// Pre-build an intent index for O(1) lookup at request time.
	definitions := registry.ListDefinitions()
	index := make(map[event.Type]event.Intent, len(definitions))
	for _, def := range definitions {
		index[def.Type] = def.Intent
	}
	return func(evt event.Event) bool {
		intent, ok := index[evt.Type]
		if !ok {
			return false
		}
		return intent == event.IntentProjectionAndReplay
	}
}

func normalizeOptions(options Options) Options {
	executeErr, applyErr, rejectErr := NormalizeErrorHandlers(ErrorHandlerOptions{
		ExecuteErr: options.ExecuteErr,
		ApplyErr:   options.ApplyErr,
		RejectErr:  options.RejectErr,
	})
	options.ExecuteErr = executeErr
	options.ApplyErr = applyErr
	options.RejectErr = rejectErr
	return options
}

// NormalizeErrorHandlers returns execute/apply/reject handlers with shared
// status defaults while allowing callers to override any callback.
func NormalizeErrorHandlers(options ErrorHandlerOptions) (
	executeErr func(error) error,
	applyErr func(error) error,
	rejectErr func(string) error,
) {
	executeErr = options.ExecuteErr
	applyErr = options.ApplyErr
	rejectErr = options.RejectErr

	if executeErr == nil {
		message := options.ExecuteErrMessage
		if message == "" {
			message = "execute domain command"
		}
		executeErr = func(err error) error {
			return status.Errorf(codes.Internal, "%s: %v", message, err)
		}
	}
	if applyErr == nil {
		message := options.ApplyErrMessage
		if message == "" {
			message = "apply event"
		}
		applyErr = func(err error) error {
			return status.Errorf(codes.Internal, "%s: %v", message, err)
		}
	}
	if rejectErr == nil {
		rejectErr = func(message string) error {
			return status.Error(codes.FailedPrecondition, message)
		}
	}

	return executeErr, applyErr, rejectErr
}
