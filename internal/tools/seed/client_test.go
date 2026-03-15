package seed

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestReadResponseForID_RejectsNilContext(t *testing.T) {
	client := &ProcessClient{}
	_, _, err := client.ReadResponseForID(nil, "1", 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected error for nil context")
	}
	if err.Error() != "context is nil" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadResponseForID_RespectsCallerContext(t *testing.T) {
	client := &ProcessClient{responses: map[string]storedResponse{}}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := client.ReadResponseForID(ctx, "1", time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled error, got: %v", err)
	}
}

func TestReadResponseForID_ReturnsStoredResponse(t *testing.T) {
	client := &ProcessClient{
		responses: map[string]storedResponse{
			"1": {
				value: map[string]any{"id": 1, "result": map[string]any{"ok": true}},
				data:  []byte(`{"id":1,"result":{"ok":true}}`),
			},
		},
	}

	response, raw, err := client.ReadResponseForID(context.Background(), "1", time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response == nil {
		t.Fatal("expected response")
	}
	if string(raw) != `{"id":1,"result":{"ok":true}}` {
		t.Fatalf("unexpected raw response %q", string(raw))
	}
	if _, ok := client.responses["1"]; ok {
		t.Fatal("expected stored response to be removed after read")
	}
}

func TestMessageID(t *testing.T) {
	id, ok := messageID(map[string]any{"id": 7})
	if !ok || id != "7" {
		t.Fatalf("messageID() = %q, %v", id, ok)
	}
	if _, ok := messageID(map[string]any{"method": "tools/call"}); ok {
		t.Fatal("expected missing id to return ok=false")
	}
}
