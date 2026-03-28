package gametools

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
)

func TestDirectSessionCallToolUnknown(t *testing.T) {
	session := NewDirectSession(Clients{}, SessionContext{})

	result, err := session.CallTool(context.Background(), "missing_tool", map[string]any{"ok": true})
	if err != nil {
		t.Fatalf("CallTool unknown error = %v", err)
	}
	if !result.IsError {
		t.Fatalf("CallTool unknown result = %#v, want error result", result)
	}
	if !strings.Contains(result.Output, "unknown tool") {
		t.Fatalf("CallTool unknown output = %q", result.Output)
	}
}

func TestDirectSessionCallToolMarshalFailure(t *testing.T) {
	session := NewDirectSession(Clients{}, SessionContext{})

	_, err := session.CallTool(context.Background(), "missing_tool", map[string]any{"bad": make(chan int)})
	if err == nil {
		t.Fatal("CallTool marshal failure error = nil, want error")
	}
	if !strings.Contains(err.Error(), "marshal tool arguments") {
		t.Fatalf("CallTool marshal failure error = %v", err)
	}
}

func TestDirectSessionCallToolExecutorFailure(t *testing.T) {
	definition := productionToolDefinition{
		Tool: orchestration.Tool{Name: "failing_tool"},
		Execute: func(*DirectSession, context.Context, []byte) (orchestration.ToolResult, error) {
			return orchestration.ToolResult{}, errors.New("boom")
		},
	}
	session := newDirectSession(Clients{}, productionToolRegistry{
		definitions: []productionToolDefinition{definition},
		byName: map[string]productionToolDefinition{
			"failing_tool": definition,
		},
	}, SessionContext{})

	result, err := session.CallTool(context.Background(), "failing_tool", map[string]any{"ok": true})
	if err != nil {
		t.Fatalf("CallTool executor failure error = %v", err)
	}
	if !result.IsError {
		t.Fatalf("CallTool executor failure result = %#v, want error result", result)
	}
	if !strings.Contains(result.Output, "tool call failed: boom") {
		t.Fatalf("CallTool executor failure output = %q", result.Output)
	}
}

func TestDirectSessionCallToolSuccess(t *testing.T) {
	definition := productionToolDefinition{
		Tool: orchestration.Tool{Name: "ok_tool"},
		Execute: func(*DirectSession, context.Context, []byte) (orchestration.ToolResult, error) {
			return orchestration.ToolResult{Output: `{"ok":true}`}, nil
		},
	}
	session := newDirectSession(Clients{}, productionToolRegistry{
		definitions: []productionToolDefinition{definition},
		byName: map[string]productionToolDefinition{
			"ok_tool": definition,
		},
	}, SessionContext{})

	result, err := session.CallTool(context.Background(), "ok_tool", map[string]any{"ok": true})
	if err != nil {
		t.Fatalf("CallTool success error = %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool success result = %#v, want non-error result", result)
	}
	if result.Output != `{"ok":true}` {
		t.Fatalf("CallTool success output = %q", result.Output)
	}
}
