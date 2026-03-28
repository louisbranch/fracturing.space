package gametools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
)

// ListTools returns the production tool definitions owned by the registry.
func (s *DirectSession) ListTools(_ context.Context) ([]orchestration.Tool, error) {
	return s.registry.tools(), nil
}

// CallTool dispatches a tool call by name to the correct gRPC handler.
func (s *DirectSession) CallTool(ctx context.Context, name string, args any) (orchestration.ToolResult, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("marshal tool arguments: %w", err)
	}

	definition, ok := s.registry.lookup(name)
	if !ok {
		return orchestration.ToolResult{
			Output:  fmt.Sprintf("unknown tool %q", name),
			IsError: true,
		}, nil
	}

	result, err := definition.Execute(s, ctx, argsJSON)
	if err != nil {
		return orchestration.ToolResult{
			Output:  fmt.Sprintf("tool call failed: %v", err),
			IsError: true,
		}, nil
	}
	return result, nil
}

// toolResultJSON marshals the result value as a JSON tool result.
func toolResultJSON(v any) (orchestration.ToolResult, error) {
	data, _ := json.Marshal(v)
	return orchestration.ToolResult{Output: string(data)}, nil
}
