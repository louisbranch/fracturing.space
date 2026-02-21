package daggerheart

import (
	"context"
	"sync/atomic"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var inlineProjectionApplyEnabled atomic.Bool

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
	skipApply       bool
	executeErrMsg   string
	rejectErr       func(string) error
}

// SetInlineProjectionApplyEnabled controls whether request-path handlers apply
// emitted domain events to projections inline.
func SetInlineProjectionApplyEnabled(enabled bool) {
	inlineProjectionApplyEnabled.Store(enabled)
}

func (s *DaggerheartService) executeDomainCommand(ctx context.Context, cmd command.Command) (engine.Result, error) {
	if s.stores.Domain == nil {
		return engine.Result{}, status.Error(codes.Internal, "domain engine is not configured")
	}
	return s.stores.Domain.Execute(ctx, cmd)
}

func (s *DaggerheartService) applyEmittedEvents(ctx context.Context, applier eventApplier, events []event.Event, applyErrMessage string) error {
	if !inlineProjectionApplyEnabled.Load() {
		return nil
	}
	for _, evt := range events {
		if err := s.applyEmittedEvent(ctx, applier, evt, applyErrMessage); err != nil {
			return err
		}
	}
	return nil
}

func (s *DaggerheartService) applyEmittedEvent(ctx context.Context, applier eventApplier, evt event.Event, applyErrMessage string) error {
	if !inlineProjectionApplyEnabled.Load() {
		return nil
	}
	if err := applier.Apply(ctx, evt); err != nil {
		return status.Errorf(codes.Internal, "%s: %v", applyErrMessage, err)
	}
	return nil
}

func (s *DaggerheartService) executeAndApplyDomainCommand(
	ctx context.Context,
	cmd command.Command,
	applier eventApplier,
	options domainCommandApplyOptions,
) (engine.Result, error) {
	options = normalizeDomainCommandOptions(options)

	result, err := s.executeDomainCommand(ctx, cmd)
	if err != nil {
		return engine.Result{}, status.Errorf(codes.Internal, "%s: %v", options.executeErrMsg, err)
	}
	if len(result.Decision.Rejections) > 0 {
		return engine.Result{}, options.rejectErr(result.Decision.Rejections[0].Message)
	}
	if options.requireEvents && len(result.Decision.Events) == 0 {
		return engine.Result{}, status.Error(codes.Internal, options.missingEventMsg)
	}
	if !options.skipApply {
		if err := s.applyEmittedEvents(ctx, applier, result.Decision.Events, options.applyErrMessage); err != nil {
			return engine.Result{}, err
		}
	}
	return result, nil
}

func normalizeDomainCommandOptions(options domainCommandApplyOptions) domainCommandApplyOptions {
	if options.executeErrMsg == "" {
		options.executeErrMsg = "execute domain command"
	}
	if options.applyErrMessage == "" {
		options.applyErrMessage = "apply event"
	}
	if options.rejectErr == nil {
		options.rejectErr = func(message string) error {
			return status.Error(codes.FailedPrecondition, message)
		}
	}
	return options
}
