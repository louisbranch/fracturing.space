package domainwriteexec

import (
	"context"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeDeps struct {
	executor domainwrite.Executor
	runtime  *domainwrite.Runtime
}

func (d fakeDeps) DomainExecutor() domainwrite.Executor {
	return d.executor
}

func (d fakeDeps) DomainWriteRuntime() *domainwrite.Runtime {
	return d.runtime
}

type fakeExecutor struct {
	result engine.Result
	err    error
}

func (f fakeExecutor) Execute(context.Context, command.Command) (engine.Result, error) {
	return f.result, f.err
}

type fakeApplier struct {
	err error
}

func (f fakeApplier) Apply(context.Context, event.Event) error {
	return f.err
}

func TestExecuteAndApply_UsesFallbackRuntimeWhenDepsRuntimeNil(t *testing.T) {
	want := engine.Result{Decision: command.Decision{}}
	got, err := ExecuteAndApply(
		context.Background(),
		fakeDeps{executor: fakeExecutor{result: want}},
		fakeApplier{},
		command.Command{CampaignID: "camp-1", Type: command.Type("campaign.create")},
		domainwrite.Options{},
		grpcerror.NormalizeDomainWriteOptionsConfig{},
	)
	if err != nil {
		t.Fatalf("execute and apply with nil runtime: %v", err)
	}
	if got.Decision.Rejections != nil || len(got.Decision.Events) != len(want.Decision.Events) {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestExecuteAndApply_PreservesDomainCodeOnApplyWhenConfigured(t *testing.T) {
	runtime := domainwrite.NewRuntime()
	runtime.SetInlineApplyEnabled(true)
	runtime.SetShouldApply(func(event.Event) bool { return true })
	domainErr := apperrors.New(apperrors.CodeNotFound, "domain object not found")

	_, err := ExecuteAndApply(
		context.Background(),
		fakeDeps{
			executor: fakeExecutor{
				result: engine.Result{
					Decision: command.Decision{Events: []event.Event{{Type: event.Type("campaign.created")}}},
				},
			},
			runtime: runtime,
		},
		fakeApplier{err: domainErr},
		command.Command{CampaignID: "camp-1", Type: command.Type("campaign.create")},
		domainwrite.Options{},
		grpcerror.NormalizeDomainWriteOptionsConfig{PreserveDomainCodeOnApply: true},
	)
	if apperrors.GetCode(err) != apperrors.CodeNotFound {
		t.Fatalf("expected preserved domain error code %s, got %s (err=%v)", apperrors.CodeNotFound, apperrors.GetCode(err), err)
	}
}

func TestExecuteAndApply_MapsApplyErrorToInternalWithoutPreserveConfig(t *testing.T) {
	runtime := domainwrite.NewRuntime()
	runtime.SetInlineApplyEnabled(true)
	runtime.SetShouldApply(func(event.Event) bool { return true })
	domainErr := apperrors.New(apperrors.CodeNotFound, "domain object not found")

	_, err := ExecuteAndApply(
		context.Background(),
		fakeDeps{
			executor: fakeExecutor{
				result: engine.Result{
					Decision: command.Decision{Events: []event.Event{{Type: event.Type("campaign.created")}}},
				},
			},
			runtime: runtime,
		},
		fakeApplier{err: domainErr},
		command.Command{CampaignID: "camp-1", Type: command.Type("campaign.create")},
		domainwrite.Options{},
		grpcerror.NormalizeDomainWriteOptionsConfig{},
	)
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected internal status from non-preserve apply config, got %s (%v)", status.Code(err), err)
	}
}
