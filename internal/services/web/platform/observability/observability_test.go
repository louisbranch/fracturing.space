package observability

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestLoggerLogsMethodAndPath(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	logger := log.New(&buffer, "", 0)
	h := RequestLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/discover/campaigns", nil)
	req.Header.Set("X-Request-ID", "req-123")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
	logLine := buffer.String()
	for _, marker := range []string{"method=GET", "path=/discover/campaigns", "status=204", "request_id=req-123"} {
		if !strings.Contains(logLine, marker) {
			t.Fatalf("log line missing marker %q: %q", marker, logLine)
		}
	}
}

func TestRequestLoggerCapturesImplicitStatusOKAndBytes(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	logger := log.New(&buffer, "", 0)
	h := RequestLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	logLine := buffer.String()
	for _, marker := range []string{"method=GET", "path=/health", "status=200", "bytes=2"} {
		if !strings.Contains(logLine, marker) {
			t.Fatalf("log line missing marker %q: %q", marker, logLine)
		}
	}
	if !strings.Contains(logLine, "latency=") {
		t.Fatalf("unexpected log line %q", logLine)
	}
}
