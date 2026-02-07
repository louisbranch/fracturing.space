package seed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// LoadFixtures reads scenario fixtures from a glob pattern and expands them into JSON-RPC steps.
func LoadFixtures(pattern string) ([]BlackboxFixture, error) {
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob fixtures: %w", err)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no fixtures found for %s", pattern)
	}
	sort.Strings(paths)
	fixtures := make([]BlackboxFixture, 0, len(paths))
	for _, path := range paths {
		fixture, err := LoadFixture(path)
		if err != nil {
			return nil, fmt.Errorf("load fixture %s: %w", path, err)
		}
		fixtures = append(fixtures, fixture)
	}
	return fixtures, nil
}

// LoadFixture reads a single scenario fixture and expands it into JSON-RPC steps.
func LoadFixture(path string) (BlackboxFixture, error) {
	file, err := os.Open(path)
	if err != nil {
		return BlackboxFixture{}, fmt.Errorf("open fixture: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.UseNumber()
	var scenario ScenarioFixture
	if err := decoder.Decode(&scenario); err != nil {
		return BlackboxFixture{}, fmt.Errorf("decode fixture: %w", err)
	}
	fixture, err := ExpandScenario(scenario)
	if err != nil {
		return BlackboxFixture{}, err
	}
	if len(fixture.Steps) == 0 {
		return BlackboxFixture{}, fmt.Errorf("fixture has no steps")
	}
	return fixture, nil
}

// expandedStep binds a scenario step to resolved variables.
type expandedStep struct {
	step ScenarioStep
	vars map[string]string
}

// ExpandScenario expands a scenario fixture into JSON-RPC steps.
func ExpandScenario(scenario ScenarioFixture) (BlackboxFixture, error) {
	expanded, err := expandScenarioSteps(scenario.Steps, scenario.Blocks, nil, nil)
	if err != nil {
		return BlackboxFixture{}, err
	}
	steps := make([]BlackboxStep, 0, len(expanded))
	requestID := 1
	for _, entry := range expanded {
		step, err := buildBlackboxStep(entry.step, entry.vars, &requestID)
		if err != nil {
			return BlackboxFixture{}, err
		}
		steps = append(steps, step)
	}
	return BlackboxFixture{Name: scenario.Name, Steps: steps, ExpectSSE: scenario.ExpectSSE}, nil
}

// expandScenarioSteps inlines block references and carries variables forward.
func expandScenarioSteps(steps []ScenarioStep, blocks map[string][]ScenarioStep, vars map[string]string, stack []string) ([]expandedStep, error) {
	var expanded []expandedStep
	for _, step := range steps {
		if step.Use == "" {
			expanded = append(expanded, expandedStep{step: step, vars: vars})
			continue
		}
		blockSteps, ok := blocks[step.Use]
		if !ok {
			return nil, fmt.Errorf("unknown block %q", step.Use)
		}
		for _, name := range stack {
			if name == step.Use {
				return nil, fmt.Errorf("recursive block reference %q", step.Use)
			}
		}
		mergedVars := mergeVars(vars, step.With)
		childStack := append(append([]string{}, stack...), step.Use)
		childExpanded, err := expandScenarioSteps(blockSteps, blocks, mergedVars, childStack)
		if err != nil {
			return nil, err
		}
		expanded = append(expanded, childExpanded...)
	}
	return expanded, nil
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
func buildBlackboxStep(step ScenarioStep, vars map[string]string, requestID *int) (BlackboxStep, error) {
	name := step.Name
	if name == "" {
		name = step.Action
	}
	name = renderVarsInString(name, vars)

	action := step.Action
	if action == "" && step.Request != nil {
		action = "raw"
	}

	request, hasID, err := buildRequest(action, step, vars, requestID)
	if err != nil {
		return BlackboxStep{}, err
	}
	if request == nil {
		return BlackboxStep{}, fmt.Errorf("step %q has no request", name)
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
	if rendered := RenderVars(step.ExpectPaths, vars); rendered != nil {
		expectOverrides, ok := rendered.(map[string]any)
		if !ok {
			return BlackboxStep{}, fmt.Errorf("expect_paths must be an object")
		}
		if expectPaths == nil {
			expectPaths = map[string]any{}
		}
		for key, value := range expectOverrides {
			expectPaths[key] = value
		}
	}

	var expectContains map[string]any
	if rendered := RenderVars(step.ExpectContains, vars); rendered != nil {
		containsOverrides, ok := rendered.(map[string]any)
		if !ok {
			return BlackboxStep{}, fmt.Errorf("expect_contains must be an object")
		}
		if len(containsOverrides) > 0 {
			expectContains = containsOverrides
		}
	}

	captures, err := ParseCaptureSpec(RenderVars(step.Capture, vars))
	if err != nil {
		return BlackboxStep{}, err
	}

	return BlackboxStep{
		Name:           name,
		ExpectStatus:   expectStatus,
		Request:        request,
		ExpectPaths:    expectPaths,
		ExpectContains: expectContains,
		Captures:       captures,
	}, nil
}

// buildRequest constructs the JSON-RPC request and assigns IDs when needed.
func buildRequest(action string, step ScenarioStep, vars map[string]string, requestID *int) (map[string]any, bool, error) {
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
				"uri": RenderVars(step.URI, vars),
			},
		}
		assignID = true
	case "read_resource":
		request = map[string]any{
			"jsonrpc": "2.0",
			"method":  "resources/read",
			"params": map[string]any{
				"uri": RenderVars(step.URI, vars),
			},
		}
		assignID = true
	case "tool_call":
		request = map[string]any{
			"jsonrpc": "2.0",
			"method":  "tools/call",
			"params": map[string]any{
				"name":      step.Tool,
				"arguments": RenderVars(step.Args, vars),
			},
		}
		assignID = true
	case "call":
		if step.Method == "" {
			return nil, false, fmt.Errorf("call action missing method")
		}
		request = map[string]any{
			"jsonrpc": "2.0",
			"method":  step.Method,
			"params":  RenderVars(step.Params, vars),
		}
		assignID = true
	case "raw":
		requestValue := RenderVars(step.Request, vars)
		requestMap, ok := requestValue.(map[string]any)
		if !ok {
			return nil, false, fmt.Errorf("raw request must be an object")
		}
		request = requestMap
	default:
		return nil, false, fmt.Errorf("unknown action %q", action)
	}

	if request == nil {
		return nil, false, nil
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
	return request, hasID, nil
}

// ParseCaptureSpec resolves capture shortcuts into concrete JSON paths.
func ParseCaptureSpec(value any) (map[string][]string, error) {
	if value == nil {
		return nil, nil
	}
	input, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("capture must be an object")
	}
	if len(input) == 0 {
		return nil, nil
	}
	output := make(map[string][]string, len(input))
	for key, raw := range input {
		switch typed := raw.(type) {
		case string:
			if paths, ok := CaptureDefaults[typed]; ok {
				output[key] = paths
				continue
			}
			output[key] = []string{typed}
		case []any:
			paths := make([]string, 0, len(typed))
			for _, item := range typed {
				text, ok := item.(string)
				if !ok {
					return nil, fmt.Errorf("capture %q path is not a string", key)
				}
				paths = append(paths, text)
			}
			output[key] = paths
		default:
			return nil, fmt.Errorf("capture %q has unsupported type", key)
		}
	}
	return output, nil
}

// RenderVars substitutes ${var} tokens in strings using vars.
func RenderVars(value any, vars map[string]string) any {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = RenderVars(child, vars)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			out[i] = RenderVars(child, vars)
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

// RenderPlaceholders substitutes {{token}} values in strings using captures.
func RenderPlaceholders(value any, captures map[string]string) (any, error) {
	switch typed := value.(type) {
	case map[string]any:
		if refValue, ok := typed["ref"]; ok && len(typed) == 1 {
			refKey, ok := refValue.(string)
			if !ok {
				return nil, fmt.Errorf("ref must be a string")
			}
			resolved, ok := captures[refKey]
			if !ok {
				return nil, fmt.Errorf("missing capture %q", refKey)
			}
			return resolved, nil
		}
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			rendered, err := RenderPlaceholders(child, captures)
			if err != nil {
				return nil, err
			}
			out[key] = rendered
		}
		return out, nil
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			rendered, err := RenderPlaceholders(child, captures)
			if err != nil {
				return nil, err
			}
			out[i] = rendered
		}
		return out, nil
	case string:
		resolved := typed
		for key, value := range captures {
			token := "{{" + key + "}}"
			resolved = strings.ReplaceAll(resolved, token, value)
		}
		return resolved, nil
	default:
		return value, nil
	}
}

// CaptureFromPaths tries multiple JSON paths until one succeeds.
func CaptureFromPaths(value any, paths []string) (string, error) {
	if len(paths) == 0 {
		return "", fmt.Errorf("no capture paths provided")
	}
	var lastErr error
	for _, path := range paths {
		found, err := LookupJSONPath(value, path)
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

// CaptureHints surfaces common capture paths for debugging failures.
func CaptureHints(value any) []string {
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

// FormatCaptureHints joins hint paths for diagnostics.
func FormatCaptureHints(hints []string) string {
	return strings.Join(hints, ", ")
}

// LookupJSONPath resolves dot paths with optional array indexing and JSON decoding.
func LookupJSONPath(value any, path string) (any, error) {
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
			decoded, err := DecodeJSONValue([]byte(text))
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

// DecodeJSONValue parses JSON into a map with preserved numbers.
func DecodeJSONValue(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

// ValuesEqual compares JSON values with basic numeric coercion.
func ValuesEqual(actual, expected any) bool {
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

// AssertArrayContains verifies that actual contains the expected subset.
func AssertArrayContains(actual, expected any) error {
	array, ok := actual.([]any)
	if !ok {
		return fmt.Errorf("expected array, got %T", actual)
	}

	expectedSlice, isSlice := expected.([]any)
	if !isSlice {
		if arrayMatches(array, expected) {
			return nil
		}
		return fmt.Errorf("no matching entries")
	}

	for _, expItem := range expectedSlice {
		if !arrayMatches(array, expItem) {
			return fmt.Errorf("missing entry %v", expItem)
		}
	}
	return nil
}

func arrayMatches(array []any, expected any) bool {
	for _, item := range array {
		if matchJSONSubset(item, expected) {
			return true
		}
	}
	return false
}

// matchJSONSubset checks whether actual satisfies all fields in expected.
func matchJSONSubset(actual, expected any) bool {
	switch exp := expected.(type) {
	case map[string]any:
		actMap, ok := actual.(map[string]any)
		if !ok {
			return false
		}
		for key, expValue := range exp {
			actValue, ok := actMap[key]
			if !ok {
				return false
			}
			if !matchJSONSubset(actValue, expValue) {
				return false
			}
		}
		return true
	case []any:
		actSlice, ok := actual.([]any)
		if !ok {
			return false
		}
		for _, expItem := range exp {
			if !arrayMatches(actSlice, expItem) {
				return false
			}
		}
		return true
	default:
		return ValuesEqual(actual, expected)
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

// FormatJSONRPCError extracts an error payload from a JSON-RPC response if present.
func FormatJSONRPCError(value any) string {
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
