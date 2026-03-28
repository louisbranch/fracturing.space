package provider

import "context"

// InvocationAdapter handles provider-specific inference invocation.
type InvocationAdapter interface {
	Invoke(ctx context.Context, input InvokeInput) (InvokeResult, error)
}

// ModelAdapter handles provider-backed model discovery.
type ModelAdapter interface {
	ListModels(ctx context.Context, input ListModelsInput) ([]Model, error)
}

// InvokeInput contains provider invocation input fields.
type InvokeInput struct {
	Model           string
	Input           string
	Instructions    string
	ReasoningEffort string
	// AuthToken is resolved only at call-time and must never be logged.
	AuthToken string
}

// InvokeResult contains invocation output.
type InvokeResult struct {
	OutputText string
	Usage      Usage
}

// ListModelsInput contains provider model-listing input fields.
type ListModelsInput struct {
	// AuthToken is resolved only at call-time and must never be logged.
	AuthToken string
}

// Model contains one provider model option.
type Model struct {
	ID string
}
