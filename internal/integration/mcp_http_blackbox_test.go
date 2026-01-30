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
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

const blackboxFixtureGlob = "internal/integration/fixtures/blackbox_*.json"

// scenarioFixture defines an action-focused scenario that expands into JSON-RPC steps.
type scenarioFixture struct {
	Name      string                    `json:"name"`
	ExpectSSE bool                      `json:"expect_sse"`
	Blocks    map[string][]scenarioStep `json:"blocks"`
	Steps     []scenarioStep            `json:"steps"`
}

// scenarioStep defines a single human-oriented action step.
type scenarioStep struct {
	Name            string         `json:"name"`
	Use             string         `json:"use"`
	With            map[string]any `json:"with"`
	Action          string         `json:"action"`
	Tool            string         `json:"tool"`
	Args            map[string]any `json:"args"`
	URI             any            `json:"uri"`
	Expect          string         `json:"expect"`
	ExpectStatus    int            `json:"expect_status"`
	ExpectPaths     map[string]any `json:"expect_paths"`
	Capture         map[string]any `json:"capture"`
	Request         map[string]any `json:"request"`
	Method          string         `json:"method"`
	Params          map[string]any `json:"params"`
	Client          map[string]any `json:"client"`
	ProtocolVersion string         `json:"protocol_version"`
}

// blackboxStep defines a single JSON-RPC request/expectation pair.
type blackboxStep struct {
	Name         string              `json:"name"`
	ExpectStatus int                 `json:"expect_status"`
	Request      map[string]any      `json:"request"`
	ExpectPaths  map[string]any      `json:"expect_paths"`
	Captures     map[string][]string `json:"captures"`
}

type blackboxFixture struct {
	Name      string         `json:"name"`
	ExpectSSE bool           `json:"expect_sse"`
	Steps     []blackboxStep `json:"steps"`
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
func loadBlackboxFixtures(t *testing.T, pattern string) []blackboxFixture {
	t.Helper()

	paths, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}
	if len(paths) == 0 {
		t.Fatalf("no fixtures found for %s", pattern)
	}
	sort.Strings(paths)
	fixtures := make([]blackboxFixture, 0, len(paths))
	for _, path := range paths {
		fixtures = append(fixtures, loadBlackboxFixture(t, path))
	}
	return fixtures
}

// loadBlackboxFixture reads the scenario fixture and expands it into JSON-RPC steps.
func loadBlackboxFixture(t *testing.T, path string) blackboxFixture {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.UseNumber()
	var scenario scenarioFixture
	if err := decoder.Decode(&scenario); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	fixture := expandScenario(t, scenario)
	if len(fixture.Steps) == 0 {
		t.Fatal("fixture has no steps")
	}
	return fixture
}

// expandedStep binds a scenario step to resolved variables.
type expandedStep struct {
	step scenarioStep
	vars map[string]string
}

// captureDefaults defines common capture shortcuts to structuredContent ID paths.
var captureDefaults = map[string][]string{
	"campaign":    {"result.structuredContent.id", "result.structured_content.id"},
	"participant": {"result.structuredContent.id", "result.structured_content.id"},
	"character":   {"result.structuredContent.id", "result.structured_content.id"},
	"session":     {"result.structuredContent.id", "result.structured_content.id"},
}

// expandScenario expands a scenario fixture into JSON-RPC steps.
func expandScenario(t *testing.T, scenario scenarioFixture) blackboxFixture {
	t.Helper()

	expanded := expandScenarioSteps(t, scenario.Steps, scenario.Blocks, nil, nil)
	steps := make([]blackboxStep, 0, len(expanded))
	requestID := 1
	for _, entry := range expanded {
		steps = append(steps, buildBlackboxStep(t, entry.step, entry.vars, &requestID))
	}
	return blackboxFixture{Name: scenario.Name, Steps: steps, ExpectSSE: scenario.ExpectSSE}
}

// expandScenarioSteps inlines block references and carries variables forward.
func expandScenarioSteps(t *testing.T, steps []scenarioStep, blocks map[string][]scenarioStep, vars map[string]string, stack []string) []expandedStep {
	t.Helper()

	var expanded []expandedStep
	for _, step := range steps {
		if step.Use == "" {
			expanded = append(expanded, expandedStep{step: step, vars: vars})
			continue
		}
		blockSteps, ok := blocks[step.Use]
		if !ok {
			t.Fatalf("unknown block %q", step.Use)
		}
		for _, name := range stack {
			if name == step.Use {
				t.Fatalf("recursive block reference %q", step.Use)
			}
		}
		mergedVars := mergeVars(vars, step.With)
		childStack := append(append([]string{}, stack...), step.Use)
		expanded = append(expanded, expandScenarioSteps(t, blockSteps, blocks, mergedVars, childStack)...)
	}
	return expanded
}

// mergeVars merges base vars with step overrides.
func mergeVars(base map[string]string, overrides map[string]any) map[string]string {
	merged := make(map[string]string)
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range overrides {
		merged[key] = fmt.Sprint(value)
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}

// buildBlackboxStep converts a scenario step into a JSON-RPC step.
func buildBlackboxStep(t *testing.T, step scenarioStep, vars map[string]string, requestID *int) blackboxStep {
	t.Helper()

	name := step.Name
	if name == "" {
		name = step.Action
	}
	name = renderVarsInString(name, vars)

	action := step.Action
	if action == "" && step.Request != nil {
		action = "raw"
	}

	request, hasID := buildRequest(t, action, step, vars, requestID)
	if request == nil {
		t.Fatalf("step %q has no request", name)
	}

	expect := step.Expect
	if expect == "" {
		if action == "initialized" {
			expect = "no_response"
		} else if hasID {
			expect = "ok"
		} else {
			expect = "none"
		}
	}

	expectStatus := step.ExpectStatus
	if expectStatus == 0 {
		switch expect {
		case "no_response":
			expectStatus = http.StatusNoContent
		default:
			expectStatus = http.StatusOK
		}
	}

	var expectPaths map[string]any
	if expect == "ok" && hasID {
		expectPaths = map[string]any{}
		if value, ok := request["jsonrpc"]; ok {
			expectPaths["jsonrpc"] = value
		}
		if value, ok := request["id"]; ok {
			expectPaths["id"] = value
		}
	}
	if rendered := renderVars(step.ExpectPaths, vars); rendered != nil {
		expectOverrides, ok := rendered.(map[string]any)
		if !ok {
			t.Fatalf("expect_paths must be an object")
		}
		if expectPaths == nil {
			expectPaths = map[string]any{}
		}
		for key, value := range expectOverrides {
			expectPaths[key] = value
		}
	}

	captures := parseCaptureSpec(t, renderVars(step.Capture, vars))

	return blackboxStep{
		Name:         name,
		ExpectStatus: expectStatus,
		Request:      request,
		ExpectPaths:  expectPaths,
		Captures:     captures,
	}
}

// buildRequest constructs the JSON-RPC request and assigns IDs when needed.
func buildRequest(t *testing.T, action string, step scenarioStep, vars map[string]string, requestID *int) (map[string]any, bool) {
	t.Helper()

	var request map[string]any
	assignID := false

	switch action {
	case "initialize":
		protocolVersion := step.ProtocolVersion
		if protocolVersion == "" {
			protocolVersion = "2024-11-05"
		}
		client := step.Client
		if client == nil {
			client = map[string]any{"name": "blackbox-client", "version": "0.1.0"}
		}
		request = map[string]any{
			"jsonrpc": "2.0",
			"method":  "initialize",
			"params": map[string]any{
				"protocolVersion": protocolVersion,
				"clientInfo":      client,
			},
		}
		assignID = true
	case "initialized":
		request = map[string]any{
			"jsonrpc": "2.0",
			"method":  "initialized",
			"params":  map[string]any{},
		}
	case "subscribe":
		request = map[string]any{
			"jsonrpc": "2.0",
			"method":  "resources/subscribe",
			"params": map[string]any{
				"uri": renderVars(step.URI, vars),
			},
		}
		assignID = true
	case "read_resource":
		request = map[string]any{
			"jsonrpc": "2.0",
			"method":  "resources/read",
			"params": map[string]any{
				"uri": renderVars(step.URI, vars),
			},
		}
		assignID = true
	case "tool_call":
		request = map[string]any{
			"jsonrpc": "2.0",
			"method":  "tools/call",
			"params": map[string]any{
				"name":      step.Tool,
				"arguments": renderVars(step.Args, vars),
			},
		}
		assignID = true
	case "call":
		if step.Method == "" {
			t.Fatalf("call action missing method")
		}
		request = map[string]any{
			"jsonrpc": "2.0",
			"method":  step.Method,
			"params":  renderVars(step.Params, vars),
		}
		assignID = true
	case "raw":
		requestValue := renderVars(step.Request, vars)
		requestMap, ok := requestValue.(map[string]any)
		if !ok {
			t.Fatalf("raw request must be an object")
		}
		request = requestMap
	default:
		t.Fatalf("unknown action %q", action)
	}

	if request == nil {
		return nil, false
	}
	if _, ok := request["jsonrpc"]; !ok {
		request["jsonrpc"] = "2.0"
	}
	if assignID {
		if _, ok := request["id"]; !ok {
			request["id"] = *requestID
			*requestID = *requestID + 1
		}
	}
	_, hasID := request["id"]
	return request, hasID
}

// parseCaptureSpec resolves capture shortcuts into concrete JSON paths.
func parseCaptureSpec(t *testing.T, value any) map[string][]string {
	t.Helper()

	if value == nil {
		return nil
	}
	input, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("capture must be an object")
	}
	if len(input) == 0 {
		return nil
	}
	output := make(map[string][]string, len(input))
	for key, raw := range input {
		switch typed := raw.(type) {
		case string:
			if paths, ok := captureDefaults[typed]; ok {
				output[key] = paths
				continue
			}
			output[key] = []string{typed}
		case []any:
			paths := make([]string, 0, len(typed))
			for _, item := range typed {
				text, ok := item.(string)
				if !ok {
					t.Fatalf("capture %q path is not a string", key)
				}
				paths = append(paths, text)
			}
			output[key] = paths
		default:
			t.Fatalf("capture %q has unsupported type", key)
		}
	}
	return output
}

// executeBlackboxStep issues the HTTP request and validates expectations and captures.
func executeBlackboxStep(t *testing.T, client *http.Client, url string, step blackboxStep, captures map[string]string) {
	t.Helper()

	request := renderPlaceholders(t, step.Request, captures)
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
		resolvedExpected := renderPlaceholders(t, expected, captures)
		if !valuesEqual(actual, resolvedExpected) {
			t.Fatalf("%s expected %s = %v, got %v (response=%s)", step.Name, path, resolvedExpected, actual, string(body))
		}
	}

	for key, paths := range step.Captures {
		value, err := captureFromPaths(response, paths)
		if err != nil {
			hints := captureHints(response)
			if len(hints) > 0 {
				t.Fatalf("%s capture %s: %v (hints=%s, response=%s)", step.Name, key, err, formatCaptureHints(hints), string(body))
			}
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

// renderVars substitutes ${var} tokens in strings using vars.
func renderVars(value any, vars map[string]string) any {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = renderVars(child, vars)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			out[i] = renderVars(child, vars)
		}
		return out
	case string:
		return renderVarsInString(typed, vars)
	default:
		return value
	}
}

// renderVarsInString replaces ${var} tokens inside a string.
func renderVarsInString(value string, vars map[string]string) string {
	if len(vars) == 0 {
		return value
	}
	resolved := value
	for key, val := range vars {
		token := "${" + key + "}"
		resolved = strings.ReplaceAll(resolved, token, val)
	}
	return resolved
}

// renderPlaceholders substitutes {{token}} values in strings using captures.
func renderPlaceholders(t *testing.T, value any, captures map[string]string) any {
	t.Helper()

	switch typed := value.(type) {
	case map[string]any:
		if refValue, ok := typed["ref"]; ok && len(typed) == 1 {
			refKey, ok := refValue.(string)
			if !ok {
				t.Fatalf("ref must be a string")
			}
			resolved, ok := captures[refKey]
			if !ok {
				t.Fatalf("missing capture %q", refKey)
			}
			return resolved
		}
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = renderPlaceholders(t, child, captures)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			out[i] = renderPlaceholders(t, child, captures)
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

// captureHints surfaces common capture paths for debugging failures.
func captureHints(value any) []string {
	response, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	resultValue, ok := response["result"]
	if !ok {
		return nil
	}
	result, ok := resultValue.(map[string]any)
	if !ok {
		return nil
	}
	var hints []string
	if structured, ok := result["structuredContent"].(map[string]any); ok {
		if _, ok := structured["id"]; ok {
			hints = append(hints, "result.structuredContent.id")
		}
	}
	if structured, ok := result["structured_content"].(map[string]any); ok {
		if _, ok := structured["id"]; ok {
			hints = append(hints, "result.structured_content.id")
		}
	}
	return hints
}

// formatCaptureHints joins hint paths for diagnostics.
func formatCaptureHints(hints []string) string {
	return strings.Join(hints, ", ")
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

// expectedResourceURIs builds the set of resource URIs that should emit updates.
func expectedResourceURIs(captures map[string]string) []string {
	campaignID := captures["campaign_id"]
	sessionID := captures["session_id"]
	return []string{
		"campaigns://list",
		fmt.Sprintf("campaign://%s", campaignID),
		fmt.Sprintf("campaign://%s/participants", campaignID),
		fmt.Sprintf("campaign://%s/characters", campaignID),
		fmt.Sprintf("campaign://%s/sessions", campaignID),
		"context://current",
		fmt.Sprintf("session://%s/events", sessionID),
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
