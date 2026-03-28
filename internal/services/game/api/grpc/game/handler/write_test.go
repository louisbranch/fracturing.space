package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeHandlerExecutor struct {
	execute  func(context.Context, command.Command) (engine.Result, error)
	commands []command.Command
}

func (f *fakeHandlerExecutor) Execute(ctx context.Context, cmd command.Command) (engine.Result, error) {
	f.commands = append(f.commands, cmd)
	if f.execute != nil {
		return f.execute(ctx, cmd)
	}
	return engine.Result{}, nil
}

func TestExecuteAndApplyDomainCommand(t *testing.T) {
	t.Parallel()

	executor := &fakeHandlerExecutor{
		execute: func(_ context.Context, cmd command.Command) (engine.Result, error) {
			if cmd.Type != commandids.CampaignUpdate {
				t.Fatalf("command type = %q, want %q", cmd.Type, commandids.CampaignUpdate)
			}
			return engine.Result{Decision: command.Decision{}}, nil
		},
	}

	result, err := ExecuteAndApplyDomainCommand(
		context.Background(),
		domainwrite.WritePath{Executor: executor, Runtime: domainwrite.NewRuntime()},
		projection.Applier{},
		command.Command{CampaignID: "camp-1", Type: commandids.CampaignUpdate},
		domainwrite.Options{},
	)
	if err != nil {
		t.Fatalf("ExecuteAndApplyDomainCommand() error = %v", err)
	}
	if len(executor.commands) != 1 {
		t.Fatalf("len(commands) = %d, want 1", len(executor.commands))
	}
	if len(result.Decision.Events) != 0 || len(result.Decision.Rejections) != 0 {
		t.Fatalf("result = %#v, want zero decision", result)
	}
}

func TestExecuteWithoutInlineApplyMapsErrorsToStatus(t *testing.T) {
	t.Parallel()

	t.Run("plain errors map to internal", func(t *testing.T) {
		t.Parallel()

		_, err := ExecuteWithoutInlineApply(
			context.Background(),
			domainwrite.WritePath{
				Executor: &fakeHandlerExecutor{
					execute: func(context.Context, command.Command) (engine.Result, error) {
						return engine.Result{}, errors.New("boom")
					},
				},
				Runtime: domainwrite.NewRuntime(),
			},
			command.Command{CampaignID: "camp-1", Type: commandids.CampaignUpdate},
			domainwrite.Options{},
		)
		if status.Code(err) != codes.Internal {
			t.Fatalf("status code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("grpc status errors are preserved", func(t *testing.T) {
		t.Parallel()

		_, err := ExecuteWithoutInlineApply(
			context.Background(),
			domainwrite.WritePath{
				Executor: &fakeHandlerExecutor{
					execute: func(context.Context, command.Command) (engine.Result, error) {
						return engine.Result{}, status.Error(codes.NotFound, "missing")
					},
				},
				Runtime: domainwrite.NewRuntime(),
			},
			command.Command{CampaignID: "camp-1", Type: commandids.CampaignUpdate},
			domainwrite.Options{
				ExecuteErr: func(err error) error { return err },
			},
		)
		if status.Code(err) != codes.NotFound {
			t.Fatalf("status code = %v, want NotFound", status.Code(err))
		}
	})
}
