//go:build integration

// Package integration includes blackbox MCP transport tests that validate the public
// request/response surface for campaign setup, session context, and action rolls.
// They serve as a baseline for transport-focused suites (HTTP and stdio).
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

const blackboxFixturePath = "internal/integration/fixtures/blackbox_mcp.json"

// blackboxFixture defines a sequence of JSON-RPC steps for HTTP transport testing.
type blackboxFixture struct {
	Name  string         `json:"name"`
	Steps []blackboxStep `json:"steps"`
}

// blackboxStep defines a single JSON-RPC request/expectation pair.
type blackboxStep struct {
	Name         string              `json:"name"`
	ExpectStatus int                 `json:"expect_status"`
	Request      map[string]any      `json:"request"`
	ExpectPaths  map[string]any      `json:"expect_paths"`
	Captures     map[string][]string `json:"captures"`
}

// TestMCPHTTPBlackbox validates HTTP transport behavior using raw JSON-RPC payloads.
func TestMCPHTTPBlackbox(t *testing.T) {
	grpcAddr, stopGRPC := startGRPCServer(t)
	defer stopGRPC()

	httpAddr := pickUnusedAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mcpCmd, err := startMCPHTTPServer(ctx, t, grpcAddr, httpAddr)
	if err != nil {
		t.Fatalf("start MCP HTTP server: %v", err)
	}
	defer stopMCPProcess(t, cancel, mcpCmd)

	baseURL := "http://" + httpAddr
	client := newHTTPClient(t)
	waitForHTTPHealth(t, client, baseURL+"/mcp/health")

	fixture := loadBlackboxFixture(t, filepath.Join(repoRoot(t), blackboxFixturePath))
	captures := make(map[string]string)
	var sseResp *http.Response
	var sseRecorder *sseCapture
	for index, step := range fixture.Steps {
		executeBlackboxStep(t, client, baseURL+"/mcp", step, captures)
		if index == 0 {
			sseClient := newSSEClient(t, client.Jar)
			sseResp, sseRecorder = openSSE(t, sseClient, baseURL+"/mcp")
		}
	}
	if sseRecorder == nil {
		t.Fatal("SSE recorder not initialized")
	}
	finishSSERecorder(t, sseResp, sseRecorder)
	assertSSEIdle(t, sseRecorder)
}

// loadBlackboxFixture reads the JSON fixture from disk with number preservation.
func loadBlackboxFixture(t *testing.T, path string) blackboxFixture {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.UseNumber()
	var fixture blackboxFixture
	if err := decoder.Decode(&fixture); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	if len(fixture.Steps) == 0 {
		t.Fatal("fixture has no steps")
	}
	return fixture
}

// executeBlackboxStep issues the HTTP request and validates expectations and captures.
func executeBlackboxStep(t *testing.T, client *http.Client, url string, step blackboxStep, captures map[string]string) {
	t.Helper()

	request := renderPlaceholders(step.Request, captures)
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request for %s: %v", step.Name, err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("build request for %s: %v", step.Name, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request %s: %v", step.Name, err)
	}
	defer resp.Body.Close()

	if step.ExpectStatus != 0 && resp.StatusCode != step.ExpectStatus {
		payload, _ := io.ReadAll(resp.Body)
		t.Fatalf("%s status = %d, want %d: %s", step.Name, resp.StatusCode, step.ExpectStatus, string(payload))
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response for %s: %v", step.Name, err)
	}
	if len(step.ExpectPaths) == 0 && len(step.Captures) == 0 {
		return
	}
	if len(body) == 0 {
		t.Fatalf("%s response body empty", step.Name)
	}

	response := decodeJSONValue(t, body)
	for path, expected := range step.ExpectPaths {
		actual, err := lookupJSONPath(response, path)
		if err != nil {
			errorDetails := formatJSONRPCError(response)
			if errorDetails != "" {
				t.Fatalf("%s lookup %s: %v (error=%s)", step.Name, path, err, errorDetails)
			}
			t.Fatalf("%s lookup %s: %v (response=%s)", step.Name, path, err, string(body))
		}
		resolvedExpected := renderPlaceholders(expected, captures)
		if !valuesEqual(actual, resolvedExpected) {
			t.Fatalf("%s expected %s = %v, got %v (response=%s)", step.Name, path, resolvedExpected, actual, string(body))
		}
	}

	for key, paths := range step.Captures {
		value, err := captureFromPaths(response, paths)
		if err != nil {
			t.Fatalf("%s capture %s: %v (response=%s)", step.Name, key, err, string(body))
		}
		if value == "" {
			t.Fatalf("%s capture %s: empty value", step.Name, key)
		}
		captures[key] = value
	}
}

// decodeJSONValue parses JSON into a map with preserved numbers.
func decodeJSONValue(t *testing.T, data []byte) any {
	t.Helper()

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("decode JSON response: %v", err)
	}
	return value
}

// renderPlaceholders substitutes {{token}} values in strings using captures.
func renderPlaceholders(value any, captures map[string]string) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = renderPlaceholders(child, captures)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			out[i] = renderPlaceholders(child, captures)
		}
		return out
	case string:
		resolved := typed
		for key, value := range captures {
			token := "{{" + key + "}}"
			resolved = strings.ReplaceAll(resolved, token, value)
		}
		return resolved
	default:
		return value
	}
}

// captureFromPaths tries multiple JSON paths until one succeeds.
func captureFromPaths(value any, paths []string) (string, error) {
	if len(paths) == 0 {
		return "", fmt.Errorf("no capture paths provided")
	}
	var lastErr error
	for _, path := range paths {
		found, err := lookupJSONPath(value, path)
		if err != nil {
			lastErr = err
			continue
		}
		return fmt.Sprint(found), nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("capture path not found")
	}
	return "", lastErr
}

// lookupJSONPath resolves dot paths with optional array indexing and JSON decoding.
func lookupJSONPath(value any, path string) (any, error) {
	current := value
	parts := strings.Split(path, "|json")
	for index, part := range parts {
		part = strings.TrimPrefix(part, ".")
		if part != "" {
			var err error
			current, err = lookupDotPath(current, part)
			if err != nil {
				return nil, err
			}
		}
		if index < len(parts)-1 {
			text, ok := current.(string)
			if !ok {
				return nil, fmt.Errorf("expected string for json decode at %q", part)
			}
			decoded, err := decodeJSONValueInternal(text)
			if err != nil {
				return nil, err
			}
			current = decoded
		}
	}
	return current, nil
}

// lookupDotPath resolves dot-separated keys with optional array indices.
func lookupDotPath(value any, path string) (any, error) {
	current := value
	segments := strings.Split(path, ".")
	for _, segment := range segments {
		if segment == "" {
			continue
		}
		field, indexes, err := parseSegment(segment)
		if err != nil {
			return nil, err
		}
		if field != "" {
			object, ok := current.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("expected object for %q", segment)
			}
			child, exists := object[field]
			if !exists {
				return nil, fmt.Errorf("missing field %q", field)
			}
			current = child
		}
		for _, idx := range indexes {
			array, ok := current.([]any)
			if !ok {
				return nil, fmt.Errorf("expected array for %q", segment)
			}
			if idx < 0 || idx >= len(array) {
				return nil, fmt.Errorf("index %d out of range", idx)
			}
			current = array[idx]
		}
	}
	return current, nil
}

// parseSegment extracts a field name and array indices from a path segment.
func parseSegment(segment string) (string, []int, error) {
	open := strings.Index(segment, "[")
	if open == -1 {
		return segment, nil, nil
	}
	if open == 0 {
		return "", nil, fmt.Errorf("missing field name in %q", segment)
	}
	field := segment[:open]
	rest := segment[open:]
	indexes := []int{}
	for len(rest) > 0 {
		if !strings.HasPrefix(rest, "[") {
			return "", nil, fmt.Errorf("invalid index syntax in %q", segment)
		}
		closeIdx := strings.Index(rest, "]")
		if closeIdx == -1 {
			return "", nil, fmt.Errorf("unclosed index in %q", segment)
		}
		indexValue := rest[1:closeIdx]
		parsed, err := strconv.Atoi(indexValue)
		if err != nil {
			return "", nil, fmt.Errorf("invalid index %q", indexValue)
		}
		indexes = append(indexes, parsed)
		rest = rest[closeIdx+1:]
	}
	return field, indexes, nil
}

// decodeJSONValueInternal parses JSON from a string using number preservation.
func decodeJSONValueInternal(data string) (any, error) {
	decoder := json.NewDecoder(strings.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

// valuesEqual compares JSON values with basic numeric coercion.
func valuesEqual(actual, expected any) bool {
	switch exp := expected.(type) {
	case json.Number:
		return compareJSONNumbers(actual, exp)
	case string:
		actStr, ok := actual.(string)
		if ok {
			return actStr == exp
		}
		return fmt.Sprint(actual) == exp
	default:
		return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
	}
}

// compareJSONNumbers compares a JSON number against an actual value.
func compareJSONNumbers(actual any, expected json.Number) bool {
	actNumber, ok := actual.(json.Number)
	if ok {
		return actNumber.String() == expected.String()
	}
	if floatValue, err := expected.Float64(); err == nil {
		switch act := actual.(type) {
		case float64:
			return act == floatValue
		case int:
			return float64(act) == floatValue
		case int64:
			return float64(act) == floatValue
		}
	}
	return fmt.Sprint(actual) == expected.String()
}

// formatJSONRPCError extracts an error payload from a JSON-RPC response if present.
func formatJSONRPCError(value any) string {
	response, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	entry, exists := response["error"]
	if !exists {
		return ""
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Sprint(entry)
	}
	return string(data)
}

// newHTTPClient builds an HTTP client with cookie support for MCP sessions.
func newHTTPClient(t *testing.T) *http.Client {
	t.Helper()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar: %v", err)
	}
	return &http.Client{Jar: jar, Timeout: 10 * time.Second}
}

// newSSEClient builds an HTTP client without timeouts for SSE streaming.
func newSSEClient(t *testing.T, jar http.CookieJar) *http.Client {
	t.Helper()

	if jar == nil {
		jar = newHTTPClient(t).Jar
	}
	return &http.Client{Jar: jar}
}

// pickUnusedAddress returns a local address with a free port.
func pickUnusedAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()
	return listener.Addr().String()
}

// startMCPHTTPServer launches the MCP process in HTTP mode.
func startMCPHTTPServer(ctx context.Context, t *testing.T, grpcAddr, httpAddr string) (*exec.Cmd, error) {
	t.Helper()

	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/mcp", "-transport=http", "-http-addr="+httpAddr, "-addr="+grpcAddr)
	cmd.Dir = repoRoot(t)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

// stopMCPProcess terminates the MCP process and waits for exit.
func stopMCPProcess(t *testing.T, cancel context.CancelFunc, cmd *exec.Cmd) {
	t.Helper()

	cancel()
	if cmd == nil {
		return
	}
	if cmd.Process == nil {
		return
	}

	processGroupID := -cmd.Process.Pid
	_ = syscall.Kill(processGroupID, syscall.SIGINT)

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	select {
	case err := <-waitDone:
		if err != nil && !errors.Is(err, context.Canceled) {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) && exitErr.ExitCode() == -1 {
				return
			}
		}
	case <-time.After(5 * time.Second):
		_ = syscall.Kill(processGroupID, syscall.SIGKILL)
		<-waitDone
	}
}

// waitForHTTPHealth polls the MCP health endpoint until it is ready.
func waitForHTTPHealth(t *testing.T, client *http.Client, url string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("MCP HTTP health check did not become ready")
}

// sseCapture records any SSE bytes until the stream is closed.
// Buffer access is synchronized by waiting on Done before reading.
type sseCapture struct {
	Buffer bytes.Buffer
	Done   chan struct{}
	Err    error
}

// openSSE connects to the SSE endpoint and begins recording the stream.
func openSSE(t *testing.T, client *http.Client, url string) (*http.Response, *sseCapture) {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("build SSE request: %v", err)
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("open SSE: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("SSE status = %d, want %d: %s", resp.StatusCode, http.StatusOK, string(payload))
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		payload, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("SSE content-type = %q, payload=%s", resp.Header.Get("Content-Type"), string(payload))
	}

	capture := &sseCapture{Done: make(chan struct{})}
	go func() {
		_, err := io.Copy(&capture.Buffer, resp.Body)
		capture.Err = err
		close(capture.Done)
	}()
	return resp, capture
}

// finishSSERecorder closes the SSE stream and waits for the reader to exit.
func finishSSERecorder(t *testing.T, resp *http.Response, capture *sseCapture) {
	t.Helper()

	if resp != nil {
		_ = resp.Body.Close()
	}
	if capture == nil {
		return
	}
	select {
	case <-capture.Done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SSE reader to stop")
	}
}

// assertSSEIdle ensures no SSE events were emitted during the test.
func assertSSEIdle(t *testing.T, capture *sseCapture) {
	t.Helper()
	if capture == nil {
		return
	}
	if capture.Buffer.Len() != 0 {
		t.Fatalf("unexpected SSE output: %q", capture.Buffer.String())
	}
}
