package domainwrite

import (
	"context"
	"errors"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type nonRetryableTestError struct {
	err error
}

func (e nonRetryableTestError) Error() string      { return e.err.Error() }
func (e nonRetryableTestError) Unwrap() error      { return e.err }
func (e nonRetryableTestError) NonRetryable() bool { return true }

type fakeDeps struct {
	executor Executor
	runtime  *Runtime
}

func (d fakeDeps) DomainExecutor() Executor     { return d.executor }
func (d fakeDeps) DomainWriteRuntime() *Runtime { return d.runtime }

type fakeTransportExecutor struct {
	result engine.Result
	err    error
}

func (f fakeTransportExecutor) Execute(context.Context, command.Command) (engine.Result, error) {
	return f.result, f.err
}

type fakeTransportApplier struct {
	err error
}

func (f fakeTransportApplier) Apply(context.Context, event.Event) error {
	return f.err
}

func TestTransportExecuteAndApply_UsesFallbackRuntimeWhenDepsRuntimeNil(t *testing.T) {
	want := engine.Result{Decision: command.Decision{}}
	got, err := TransportExecuteAndApply(
		context.Background(),
		fakeDeps{executor: fakeTransportExecutor{result: want}},
		fakeTransportApplier{},
		command.Command{CampaignID: "camp-1", Type: command.Type("campaign.create")},
		Options{},
		NormalizeDomainWriteOptionsConfig{},
	)
	if err != nil {
		t.Fatalf("execute and apply with nil runtime: %v", err)
	}
	if got.Decision.Rejections != nil || len(got.Decision.Events) != len(want.Decision.Events) {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestTransportExecuteAndApply_PreservesDomainCodeOnApplyWhenConfigured(t *testing.T) {
	runtime := NewRuntime()
	runtime.SetInlineApplyEnabled(true)
	runtime.SetShouldApply(func(event.Event) bool { return true })
	domainErr := apperrors.New(apperrors.CodeNotFound, "domain object not found")

	_, err := TransportExecuteAndApply(
		context.Background(),
		fakeDeps{
			executor: fakeTransportExecutor{
				result: engine.Result{
					Decision: command.Decision{Events: []event.Event{{Type: event.Type("campaign.created")}}},
				},
			},
			runtime: runtime,
		},
		fakeTransportApplier{err: domainErr},
		command.Command{CampaignID: "camp-1", Type: command.Type("campaign.create")},
		Options{},
		NormalizeDomainWriteOptionsConfig{PreserveDomainCodeOnApply: true},
	)
	if apperrors.GetCode(err) != apperrors.CodeNotFound {
		t.Fatalf("expected preserved domain error code %s, got %s (err=%v)", apperrors.CodeNotFound, apperrors.GetCode(err), err)
	}
}

func TestTransportExecuteAndApply_MapsApplyErrorToInternalWithoutPreserveConfig(t *testing.T) {
	runtime := NewRuntime()
	runtime.SetInlineApplyEnabled(true)
	runtime.SetShouldApply(func(event.Event) bool { return true })
	domainErr := apperrors.New(apperrors.CodeNotFound, "domain object not found")

	_, err := TransportExecuteAndApply(
		context.Background(),
		fakeDeps{
			executor: fakeTransportExecutor{
				result: engine.Result{
					Decision: command.Decision{Events: []event.Event{{Type: event.Type("campaign.created")}}},
				},
			},
			runtime: runtime,
		},
		fakeTransportApplier{err: domainErr},
		command.Command{CampaignID: "camp-1", Type: command.Type("campaign.create")},
		Options{},
		NormalizeDomainWriteOptionsConfig{},
	)
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected internal status from non-preserve apply config, got %s (%v)", status.Code(err), err)
	}
}

func TestNormalizeDomainWriteOptionsDefaults(t *testing.T) {
	options := Options{}
	NormalizeDomainWriteOptions(context.Background(), &options, NormalizeDomainWriteOptionsConfig{})

	if options.ExecuteErr == nil || options.ApplyErr == nil || options.RejectErr == nil {
		t.Fatal("expected execute/apply/reject handlers to be initialized")
	}

	execErr := options.ExecuteErr(nonRetryableTestError{err: errors.New("checkpoint failed")})
	if status.Code(execErr) != codes.FailedPrecondition {
		t.Fatalf("execute code = %s, want %s", status.Code(execErr), codes.FailedPrecondition)
	}

	applyErr := options.ApplyErr(errors.New("apply failed"))
	if status.Code(applyErr) != codes.Internal {
		t.Fatalf("apply code = %s, want %s", status.Code(applyErr), codes.Internal)
	}

	rejectErr := options.RejectErr("SOME_CODE", "rejected")
	if status.Code(rejectErr) != codes.FailedPrecondition {
		t.Fatalf("reject code = %s, want %s", status.Code(rejectErr), codes.FailedPrecondition)
	}
}

func TestNormalizeDomainWriteOptionsPreservesDomainApplyCode(t *testing.T) {
	options := Options{}
	NormalizeDomainWriteOptions(context.Background(), &options, NormalizeDomainWriteOptionsConfig{
		PreserveDomainCodeOnApply: true,
	})

	domainErr := apperrors.New(apperrors.CodeNotFound, "not found")
	if got := options.ApplyErr(domainErr); got != domainErr {
		t.Fatalf("apply err should preserve domain error instance")
	}
}
