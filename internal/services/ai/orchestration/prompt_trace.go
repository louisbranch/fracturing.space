package orchestration

import "context"

type promptBuildTraceRecorderKey struct{}

// PromptBuildTraceRecorder records best-effort prompt augmentation metadata for
// the current build.
type PromptBuildTraceRecorder interface {
	RecordRetrievedContexts(contexts []RetrievedContext)
	RecordPromptContextPolicy(policy PromptContextPolicy)
	RecordPromptAugmentation(diagnostics PromptAugmentationDiagnostics)
}

// WithPromptBuildTraceRecorder attaches a trace recorder to the context used by
// prompt building.
func WithPromptBuildTraceRecorder(ctx context.Context, recorder PromptBuildTraceRecorder) context.Context {
	if recorder == nil {
		return ctx
	}
	return context.WithValue(ctx, promptBuildTraceRecorderKey{}, recorder)
}

// RecordRetrievedContexts records prompt augmentation metadata on the context's
// attached trace recorder when one is present.
func RecordRetrievedContexts(ctx context.Context, contexts []RetrievedContext) {
	if len(contexts) == 0 {
		return
	}
	recorder, _ := ctx.Value(promptBuildTraceRecorderKey{}).(PromptBuildTraceRecorder)
	if recorder == nil {
		return
	}
	recorder.RecordRetrievedContexts(contexts)
}

// RecordPromptContextPolicy records which optional raw artifact sections were
// enabled for prompt collection when a trace recorder is attached.
func RecordPromptContextPolicy(ctx context.Context, policy PromptContextPolicy) {
	recorder, _ := ctx.Value(promptBuildTraceRecorderKey{}).(PromptBuildTraceRecorder)
	if recorder == nil {
		return
	}
	recorder.RecordPromptContextPolicy(policy)
}

// RecordPromptAugmentation records best-effort OpenViking augmentation
// diagnostics when a trace recorder is attached.
func RecordPromptAugmentation(ctx context.Context, diagnostics PromptAugmentationDiagnostics) {
	recorder, _ := ctx.Value(promptBuildTraceRecorderKey{}).(PromptBuildTraceRecorder)
	if recorder == nil {
		return
	}
	recorder.RecordPromptAugmentation(diagnostics)
}
