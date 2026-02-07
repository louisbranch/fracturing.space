//go:build integration

// Package integration includes blackbox MCP transport tests that validate the public
// request/response surface for campaign setup, session context, and action rolls.
// They serve as a baseline for transport-focused suites (HTTP and stdio).
package integration

import (
	"bufio"
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
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/seed"
)

const blackboxFixtureGlob = "internal/test/integration/fixtures/blackbox_*.json"

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

	fixtures := loadBlackboxFixtures(t, filepath.Join(repoRoot(t), blackboxFixtureGlob))
	for _, fixture := range fixtures {
		captures := make(map[string]string)
		var sseResp *http.Response
		var sseRecorder *sseCapture
		for index, step := range fixture.Steps {
			executeBlackboxStep(t, client, baseURL+"/mcp", step, captures)
			if fixture.ExpectSSE && index == 0 {
				sseClient := newSSEClient(t, client.Jar)
				sseResp, sseRecorder = openSSE(t, sseClient, baseURL+"/mcp")
			}
		}
		if fixture.ExpectSSE {
			if sseRecorder == nil {
				t.Fatal("SSE recorder not initialized")
			}
			finishSSERecorder(t, sseResp, sseRecorder)
			assertSSEResourceUpdates(t, sseRecorder, expectedResourceURIs(captures))
		}
	}
}

// loadBlackboxFixtures reads scenario fixtures and expands them into JSON-RPC steps.
func loadBlackboxFixtures(t *testing.T, pattern string) []seed.BlackboxFixture {
	t.Helper()

	paths, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}
	if len(paths) == 0 {
		t.Fatalf("no fixtures found for %s", pattern)
	}
	sort.Strings(paths)
	fixtures := make([]seed.BlackboxFixture, 0, len(paths))
	for _, path := range paths {
		fixtures = append(fixtures, loadBlackboxFixture(t, path))
	}
	return fixtures
}

// loadBlackboxFixture reads the scenario fixture and expands it into JSON-RPC steps.
func loadBlackboxFixture(t *testing.T, path string) seed.BlackboxFixture {
	t.Helper()

	fixture, err := seed.LoadFixture(path)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	return fixture
}

// executeBlackboxStep issues the HTTP request and validates expectations and captures.
func executeBlackboxStep(t *testing.T, client *http.Client, url string, step seed.BlackboxStep, captures map[string]string) {
	t.Helper()

	request, err := seed.RenderPlaceholders(step.Request, captures)
	if err != nil {
		t.Fatalf("render placeholders for %s: %v", step.Name, err)
	}
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
	if len(step.ExpectPaths) == 0 && len(step.ExpectContains) == 0 && len(step.Captures) == 0 {
		return
	}
	if len(body) == 0 {
		t.Fatalf("%s response body empty", step.Name)
	}

	response, err := seed.DecodeJSONValue(body)
	if err != nil {
		t.Fatalf("decode JSON response for %s: %v", step.Name, err)
	}
	for path, expected := range step.ExpectPaths {
		actual, err := seed.LookupJSONPath(response, path)
		if err != nil {
			errorDetails := seed.FormatJSONRPCError(response)
			if errorDetails != "" {
				t.Fatalf("%s lookup %s: %v (error=%s)", step.Name, path, err, errorDetails)
			}
			t.Fatalf("%s lookup %s: %v (response=%s)", step.Name, path, err, string(body))
		}
		resolvedExpected, err := seed.RenderPlaceholders(expected, captures)
		if err != nil {
			t.Fatalf("render expected for %s: %v", step.Name, err)
		}
		if !seed.ValuesEqual(actual, resolvedExpected) {
			t.Fatalf("%s expected %s = %v, got %v (response=%s)", step.Name, path, resolvedExpected, actual, string(body))
		}
	}

	for path, expected := range step.ExpectContains {
		actual, err := seed.LookupJSONPath(response, path)
		if err != nil {
			errorDetails := seed.FormatJSONRPCError(response)
			if errorDetails != "" {
				t.Fatalf("%s lookup %s: %v (error=%s)", step.Name, path, err, errorDetails)
			}
			t.Fatalf("%s lookup %s: %v (response=%s)", step.Name, path, err, string(body))
		}
		resolvedExpected, err := seed.RenderPlaceholders(expected, captures)
		if err != nil {
			t.Fatalf("render expected for %s: %v", step.Name, err)
		}
		if err := seed.AssertArrayContains(actual, resolvedExpected); err != nil {
			t.Fatalf("%s expected %s to contain %v: %v (response=%s)", step.Name, path, resolvedExpected, err, string(body))
		}
	}

	for key, paths := range step.Captures {
		value, err := seed.CaptureFromPaths(response, paths)
		if err != nil {
			hints := seed.CaptureHints(response)
			if len(hints) > 0 {
				t.Fatalf("%s capture %s: %v (hints=%s, response=%s)", step.Name, key, err, seed.FormatCaptureHints(hints), string(body))
			}
			t.Fatalf("%s capture %s: %v (response=%s)", step.Name, key, err, string(body))
		}
		if value == "" {
			t.Fatalf("%s capture %s: empty value", step.Name, key)
		}
		captures[key] = value
	}
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

// expectedResourceURIs builds the set of resource URIs that should emit updates.
func expectedResourceURIs(captures map[string]string) []string {
	campaignID := captures["campaign_id"]
	return []string{
		"campaigns://list",
		fmt.Sprintf("campaign://%s", campaignID),
		fmt.Sprintf("campaign://%s/participants", campaignID),
		fmt.Sprintf("campaign://%s/characters", campaignID),
		fmt.Sprintf("campaign://%s/sessions", campaignID),
		"context://current",
	}
}

// assertSSEResourceUpdates ensures resource update notifications were emitted.
func assertSSEResourceUpdates(t *testing.T, capture *sseCapture, expectedURIs []string) {
	t.Helper()
	if capture == nil {
		t.Fatal("SSE capture missing")
	}
	notifications := parseSSENotifications(t, capture)
	if len(notifications) == 0 {
		t.Fatal("expected SSE notifications but none were captured")
	}

	seen := make(map[string]struct{})
	for _, message := range notifications {
		if message.Method != "notifications/resources/updated" {
			continue
		}
		if message.URI != "" {
			seen[message.URI] = struct{}{}
		}
	}
	if len(seen) == 0 {
		t.Fatalf("no resources/updated notifications found; raw=%q", capture.Buffer.String())
	}
	for _, uri := range expectedURIs {
		if _, ok := seen[uri]; !ok {
			t.Fatalf("missing resources/updated notification for %q (seen=%v)", uri, sortedKeys(seen))
		}
	}
}

type sseNotification struct {
	Method string
	URI    string
}

func parseSSENotifications(t *testing.T, capture *sseCapture) []sseNotification {
	t.Helper()
	data := capture.Buffer.String()
	if data == "" {
		return nil
	}

	scanner := bufio.NewScanner(strings.NewReader(data))
	var payloadLines []string
	var notifications []sseNotification
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if len(payloadLines) == 0 {
				continue
			}
			payload := strings.Join(payloadLines, "\n")
			payloadLines = nil
			notifications = append(notifications, decodeSSENotification(t, payload))
			continue
		}
		if strings.HasPrefix(line, "data: ") {
			payloadLines = append(payloadLines, strings.TrimPrefix(line, "data: "))
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan SSE stream: %v", err)
	}
	if len(payloadLines) != 0 {
		payload := strings.Join(payloadLines, "\n")
		notifications = append(notifications, decodeSSENotification(t, payload))
	}
	return notifications
}

func decodeSSENotification(t *testing.T, payload string) sseNotification {
	t.Helper()
	var envelope struct {
		Method string         `json:"method"`
		Params map[string]any `json:"params"`
	}
	if err := json.Unmarshal([]byte(payload), &envelope); err != nil {
		t.Fatalf("decode SSE payload: %v", err)
	}
	notification := sseNotification{Method: envelope.Method}
	if uriValue, ok := envelope.Params["uri"]; ok {
		if uri, ok := uriValue.(string); ok {
			notification.URI = uri
		}
	}
	return notification
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
