package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestExecuteStep_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // cancel immediately

	client := &fakeMCPClient{}
	step := BlackboxStep{
		Name:    "test step",
		Request: map[string]any{"id": 1, "method": "tools/call"},
	}
	err := executeStep(ctx, client, step, map[string]string{}, false, "user-1")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("expected context canceled error, got: %v", err)
	}
}

func TestExecuteStep_WriteError(t *testing.T) {
	client := &fakeMCPClient{
		writeMessage: func(_ any) error {
			return fmt.Errorf("write failed")
		},
	}
	step := BlackboxStep{
		Name:    "test step",
		Request: map[string]any{"id": 1, "method": "tools/call"},
	}
	err := executeStep(t.Context(), client, step, map[string]string{}, false, "user-1")
	if err == nil {
		t.Fatal("expected error for write failure")
	}
	if !strings.Contains(err.Error(), "write failed") {
		t.Fatalf("expected write error, got: %v", err)
	}
}

func TestExecuteStep_ReadError(t *testing.T) {
	client := &fakeMCPClient{
		readResponseForID: func(_ any, _ time.Duration) (any, []byte, error) {
			return nil, nil, fmt.Errorf("read failed")
		},
	}
	step := BlackboxStep{
		Name:    "test step",
		Request: map[string]any{"id": 1, "method": "tools/call"},
	}
	err := executeStep(t.Context(), client, step, map[string]string{}, false, "user-1")
	if err == nil {
		t.Fatal("expected error for read failure")
	}
	if !strings.Contains(err.Error(), "read failed") {
		t.Fatalf("expected read error, got: %v", err)
	}
}

func TestExecuteStep_NoRequestID(t *testing.T) {
	writeCalled := false
	client := &fakeMCPClient{
		writeMessage: func(_ any) error {
			writeCalled = true
			return nil
		},
		readResponseForID: func(_ any, _ time.Duration) (any, []byte, error) {
			t.Fatal("ReadResponseForID should not be called for fire-and-forget steps")
			return nil, nil, nil
		},
	}
	// Request without "id" field — fire-and-forget
	step := BlackboxStep{
		Name:    "notification",
		Request: map[string]any{"method": "notifications/initialized"},
	}
	err := executeStep(t.Context(), client, step, map[string]string{}, false, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !writeCalled {
		t.Fatal("expected WriteMessage to be called")
	}
}

func TestExecuteStep_JSONRPCError(t *testing.T) {
	client := &fakeMCPClient{
		readResponseForID: func(_ any, _ time.Duration) (any, []byte, error) {
			resp := map[string]any{
				"id":    1,
				"error": map[string]any{"code": -32600, "message": "Invalid Request"},
			}
			b, _ := json.Marshal(resp)
			return resp, b, nil
		},
	}
	step := BlackboxStep{
		Name:    "test step",
		Request: map[string]any{"id": 1, "method": "tools/call"},
	}
	err := executeStep(t.Context(), client, step, map[string]string{}, false, "user-1")
	if err == nil {
		t.Fatal("expected error for JSON-RPC error response")
	}
	if !strings.Contains(err.Error(), "JSON-RPC error") {
		t.Fatalf("expected JSON-RPC error, got: %v", err)
	}
}

func TestExecuteStep_CaptureSuccess(t *testing.T) {
	client := &fakeMCPClient{
		readResponseForID: func(_ any, _ time.Duration) (any, []byte, error) {
			resp := map[string]any{
				"id":     1,
				"result": map[string]any{"structuredContent": map[string]any{"id": "captured-id"}},
			}
			b, _ := json.Marshal(resp)
			return resp, b, nil
		},
	}
	captures := map[string]string{}
	step := BlackboxStep{
		Name:     "test step",
		Request:  map[string]any{"id": 1, "method": "tools/call"},
		Captures: map[string][]string{"campaign": {"result.structuredContent.id"}},
	}
	err := executeStep(t.Context(), client, step, captures, false, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captures["campaign"] != "captured-id" {
		t.Fatalf("expected capture campaign=captured-id, got %q", captures["campaign"])
	}
}

func TestRunFixtures_WriteError(t *testing.T) {
	client := &fakeMCPClient{
		writeMessage: func(_ any) error {
			return fmt.Errorf("write failed")
		},
	}
	fixtures := []BlackboxFixture{
		{Name: "fixture1", Steps: []BlackboxStep{
			{Name: "step1", Request: map[string]any{"id": 1, "method": "tools/call"}},
		}},
	}
	err := runFixtures(t.Context(), client, fixtures, false, "user-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "write failed") {
		t.Fatalf("expected write error, got: %v", err)
	}
}

func TestInjectCampaignCreatorUserID(t *testing.T) {
	tests := []struct {
		name     string
		request  map[string]any
		userID   string
		wantUID  string // expected user_id in arguments after call; empty means no change
		noChange bool   // if true, request should be unmodified
	}{
		{
			name:     "nil request",
			request:  nil,
			userID:   "u1",
			noChange: true,
		},
		{
			name:     "wrong method",
			request:  map[string]any{"method": "resources/read"},
			userID:   "u1",
			noChange: true,
		},
		{
			name:     "missing params",
			request:  map[string]any{"method": "tools/call"},
			userID:   "u1",
			noChange: true,
		},
		{
			name:     "params not a map",
			request:  map[string]any{"method": "tools/call", "params": "bad"},
			userID:   "u1",
			noChange: true,
		},
		{
			name: "wrong tool name",
			request: map[string]any{
				"method": "tools/call",
				"params": map[string]any{"name": "session_start", "arguments": map[string]any{}},
			},
			userID:   "u1",
			noChange: true,
		},
		{
			name: "arguments not a map",
			request: map[string]any{
				"method": "tools/call",
				"params": map[string]any{"name": "campaign_create", "arguments": "bad"},
			},
			userID:   "u1",
			noChange: true,
		},
		{
			name: "user_id already set",
			request: map[string]any{
				"method": "tools/call",
				"params": map[string]any{"name": "campaign_create", "arguments": map[string]any{"user_id": "existing"}},
			},
			userID:  "u1",
			wantUID: "existing",
		},
		{
			name: "injects user_id",
			request: map[string]any{
				"method": "tools/call",
				"params": map[string]any{"name": "campaign_create", "arguments": map[string]any{"title": "Test"}},
			},
			userID:  "u1",
			wantUID: "u1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			injectCampaignCreatorUserID(tt.request, tt.userID)

			if tt.noChange {
				return
			}

			args := tt.request["params"].(map[string]any)["arguments"].(map[string]any)
			got, ok := args["user_id"].(string)
			if !ok {
				t.Fatalf("user_id not a string: %v", args["user_id"])
			}
			if got != tt.wantUID {
				t.Fatalf("expected user_id=%q, got %q", tt.wantUID, got)
			}
		})
	}
}

func TestRunFixtures_ReadError(t *testing.T) {
	client := &fakeMCPClient{
		readResponseForID: func(_ any, _ time.Duration) (any, []byte, error) {
			return nil, nil, fmt.Errorf("read failed")
		},
	}
	fixtures := []BlackboxFixture{
		{Name: "fixture1", Steps: []BlackboxStep{
			{Name: "step1", Request: map[string]any{"id": 1, "method": "tools/call"}},
		}},
	}
	err := runFixtures(t.Context(), client, fixtures, false, "user-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "read failed") {
		t.Fatalf("expected read error, got: %v", err)
	}
}
