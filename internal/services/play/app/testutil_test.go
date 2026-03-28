package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func assertJSONError(t *testing.T, rr *httptest.ResponseRecorder, wantStatus int, wantMessage string) {
	t.Helper()
	if rr.Code != wantStatus {
		t.Fatalf("status = %d, want %d body=%s", rr.Code, wantStatus, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if got := strings.TrimSpace(payload["error"].(string)); got != wantMessage {
		t.Fatalf("error = %q, want %q", got, wantMessage)
	}
}

type syncedFrameBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncedFrameBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncedFrameBuffer) drain() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()

	data := append([]byte(nil), b.buf.Bytes()...)
	b.buf.Reset()
	return data
}

func drainWSFrames(t *testing.T, buffer *syncedFrameBuffer) []wsFrame {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(buffer.drain()))
	frames := []wsFrame{}
	for {
		var frame wsFrame
		if err := decoder.Decode(&frame); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Fatalf("decode ws frame: %v", err)
		}
		frames = append(frames, frame)
	}
	return frames
}
