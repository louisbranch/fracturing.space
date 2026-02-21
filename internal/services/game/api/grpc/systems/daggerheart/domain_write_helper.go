package daggerheart

import (
	"context"
	"sync/atomic"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
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
	return domainwrite.ExecuteAndApply(ctx, s.stores.Domain, applier, cmd, domainwrite.Options{
		RequireEvents:      options.requireEvents,
		MissingEventMsg:    options.missingEventMsg,
		InlineApplyEnabled: inlineProjectionApplyEnabled.Load(),
		ShouldApply:        domainwrite.ShouldApplyProjectionInline,
		ExecuteErr: func(err error) error {
			return status.Errorf(codes.Internal, "%s: %v", options.executeErrMsg, err)
		},
		ApplyErr: func(err error) error {
			return status.Errorf(codes.Internal, "%s: %v", options.applyErrMessage, err)
		},
		RejectErr: options.rejectErr,
	})
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
