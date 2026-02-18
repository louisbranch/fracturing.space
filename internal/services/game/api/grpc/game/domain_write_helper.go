package game

import (
	"context"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type domainCommandApplyOptions struct {
	requireEvents     bool
	missingEventMsg   string
	applyErr          func(error) error
	executeErr        func(error) error
	rejectErr         func(string) error
	executeErrMessage string
	applyErrMessage   string
}

func executeAndApplyDomainCommand(
	ctx context.Context,
	domain Domain,
	applier projection.Applier,
	cmd command.Command,
	options domainCommandApplyOptions,
) (engine.Result, error) {
	options = normalizeDomainCommandOptions(options)

	if domain == nil {
		return engine.Result{}, status.Error(codes.Internal, "domain engine is not configured")
	}

	result, err := domain.Execute(ctx, cmd)
	if err != nil {
		return engine.Result{}, options.executeErr(err)
	}
	if len(result.Decision.Rejections) > 0 {
		return engine.Result{}, options.rejectErr(result.Decision.Rejections[0].Message)
	}
	if options.requireEvents && len(result.Decision.Events) == 0 {
		return engine.Result{}, status.Error(codes.Internal, options.missingEventMsg)
	}
	if err := applyDomainDecisionEvents(ctx, applier, result.Decision.Events, options.applyErr); err != nil {
		return engine.Result{}, err
	}

	return result, nil
}

func applyDomainDecisionEvents(ctx context.Context, applier projection.Applier, events []event.Event, mapErr func(error) error) error {
	for _, evt := range events {
		if err := applier.Apply(ctx, evt); err != nil {
			return mapErr(err)
		}
	}
	return nil
}

func normalizeDomainCommandOptions(options domainCommandApplyOptions) domainCommandApplyOptions {
	if options.executeErr == nil {
		message := options.executeErrMessage
		if message == "" {
			message = "execute domain command"
		}
		options.executeErr = func(err error) error {
			return status.Errorf(codes.Internal, "%s: %v", message, err)
		}
	}
	if options.applyErr == nil {
		message := options.applyErrMessage
		if message == "" {
			message = "apply event"
		}
		options.applyErr = func(err error) error {
			return status.Errorf(codes.Internal, "%s: %v", message, err)
		}
	}
	if options.rejectErr == nil {
		options.rejectErr = func(message string) error {
			return status.Error(codes.FailedPrecondition, message)
		}
	}
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
