package seed

import (
	"bufio"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestReadResponseForID_RejectsNilContext(t *testing.T) {
	client := &StdioClient{
		reader: bufio.NewReader(strings.NewReader(`{"id":1}` + "\n")),
	}
	_, _, err := client.ReadResponseForID(nil, "1", 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected error for nil context")
	}
	if err.Error() != "context is nil" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadResponseForID_RespectsCallerContext(t *testing.T) {
	reader, writer := io.Pipe()
	t.Cleanup(func() {
		_ = writer.Close()
	})
	client := &StdioClient{
		reader: bufio.NewReader(reader),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := client.ReadResponseForID(ctx, "1", time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled error, got: %v", err)
	}
}

func TestReadResponseForID_SkipsNonMatchingResponses(t *testing.T) {
	responseData := `{"id":2}` + "\n" + `{"id":1,"result":{"ok":true}}` + "\n"
	client := &StdioClient{
		reader: bufio.NewReader(strings.NewReader(responseData)),
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
}
