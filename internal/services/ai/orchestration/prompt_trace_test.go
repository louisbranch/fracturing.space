package orchestration

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func TestWithPromptBuildTraceRecorderNilRecorderPreservesContext(t *testing.T) {
	ctx := context.Background()
	if got := WithPromptBuildTraceRecorder(ctx, nil); got != ctx {
		t.Fatal("expected nil recorder to preserve context")
	}
}

func TestRecordRetrievedContextsIgnoresEmptyOrMissingRecorder(t *testing.T) {
	RecordRetrievedContexts(context.Background(), nil)
	RecordRetrievedContexts(context.Background(), []RetrievedContext{{URI: "viking://resource"}})
}

func TestPromptTraceRecordingUsesAttachedRecorder(t *testing.T) {
	recorder := &promptTraceRecorderStub{}
	ctx := WithPromptBuildTraceRecorder(context.Background(), recorder)

	RecordRetrievedContexts(ctx, []RetrievedContext{{URI: "viking://resource"}})
	RecordPromptContextPolicy(ctx, PromptContextPolicy{IncludeStory: true})
	RecordPromptAugmentation(ctx, PromptAugmentationDiagnostics{Attempted: true})

	if len(recorder.contexts) != 1 {
		t.Fatalf("recorded contexts = %d, want 1", len(recorder.contexts))
	}
	if !recorder.policy.IncludeStory {
		t.Fatalf("recorded policy = %#v", recorder.policy)
	}
	if !recorder.augmentation.Attempted {
		t.Fatalf("recorded augmentation = %#v", recorder.augmentation)
	}
}

func TestPromptBuildTraceAccumulatesDiagnostics(t *testing.T) {
	trace := &promptBuildTrace{}

	trace.RecordPromptContextPolicy(PromptContextPolicy{IncludeMemory: true})
	trace.RecordPromptAugmentation(PromptAugmentationDiagnostics{
		Attempted:       true,
		Mode:            "docs_aligned_supplement",
		SearchAttempted: true,
		ResourceHits:    2,
		MemoryHits:      1,
		MirroredTargets: []string{"one", "one", "two"},
	})
	trace.RecordPromptAugmentation(PromptAugmentationDiagnostics{
		Degraded:          true,
		DegradationReason: "timeout",
	})

	if !trace.diagnostics.ContextPolicy.IncludeMemory {
		t.Fatalf("context policy = %#v", trace.diagnostics.ContextPolicy)
	}
	augmentation := trace.diagnostics.Augmentation
	if !augmentation.Attempted || !augmentation.SearchAttempted || !augmentation.Degraded {
		t.Fatalf("augmentation = %#v", augmentation)
	}
	if augmentation.Mode != "docs_aligned_supplement" || augmentation.ResourceHits != 2 || augmentation.MemoryHits != 1 {
		t.Fatalf("augmentation = %#v", augmentation)
	}
	if len(augmentation.MirroredTargets) != 2 {
		t.Fatalf("mirrored targets = %#v, want deduped items", augmentation.MirroredTargets)
	}
	if augmentation.DegradationReason != "timeout" {
		t.Fatalf("degradation reason = %q", augmentation.DegradationReason)
	}
}

func TestErrorHelpersWrapExpectedCodes(t *testing.T) {
	if err := errRunnerUnavailable(); err == nil {
		t.Fatal("expected runner unavailable error")
	}
	if err := errPromptBuilderUnavailable(); err == nil {
		t.Fatal("expected prompt builder unavailable error")
	}
	if err := errInvalidInput("bad"); err == nil {
		t.Fatal("expected invalid input error")
	}
	if err := errPromptBuild(errors.New("boom")); err == nil {
		t.Fatal("expected prompt build error")
	}
	if err := errExecution(errors.New("boom")); err == nil {
		t.Fatal("expected execution error")
	}
}

func TestRecordSpanErrorSetsStatus(t *testing.T) {
	span := &spanStub{}
	err := errors.New("boom")

	recordSpanError(nil, err)
	recordSpanError(span, nil)
	recordSpanError(span, err)

	if len(span.recordedErrors) != 1 {
		t.Fatalf("recorded errors = %d, want 1", len(span.recordedErrors))
	}
	if span.statusCode != codes.Error || span.statusDescription != "boom" {
		t.Fatalf("status = (%v, %q), want error boom", span.statusCode, span.statusDescription)
	}
}

type spanStub struct {
	trace.Span
	recordedErrors    []error
	statusCode        codes.Code
	statusDescription string
}

type inferNameSource struct{}

func (s *spanStub) RecordError(err error, _ ...trace.EventOption) {
	s.recordedErrors = append(s.recordedErrors, err)
}

func (s *spanStub) SetStatus(code codes.Code, description string) {
	s.statusCode = code
	s.statusDescription = description
}

func (inferNameSource) Collect(context.Context, Session, PromptInput) (BriefContribution, error) {
	return BriefContribution{}, nil
}

func TestFallbackContextSourceName(t *testing.T) {
	if got := fallbackContextSourceName(-1); got != "context_source" {
		t.Fatalf("fallbackContextSourceName(-1) = %q", got)
	}
	if got := fallbackContextSourceName(2); got != "context_source_3" {
		t.Fatalf("fallbackContextSourceName(2) = %q", got)
	}
}

func TestInferContextSourceNameHandlesNilAndStruct(t *testing.T) {
	if got := inferContextSourceName(nil, 1); got != "context_source_2" {
		t.Fatalf("inferContextSourceName(nil, 1) = %q", got)
	}

	if got := inferContextSourceName(inferNameSource{}, 0); got != "inferNameSource" {
		t.Fatalf("inferContextSourceName(inferNameSource{}, 0) = %q", got)
	}
}
