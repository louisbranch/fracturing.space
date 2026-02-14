package seed

import (
	"encoding/json"
	"testing"
)

func TestRenderVars(t *testing.T) {
	t.Run("nil value", func(t *testing.T) {
		if got := RenderVars(nil, map[string]string{"x": "1"}); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("string substitution", func(t *testing.T) {
		result := RenderVars("hello ${name}", map[string]string{"name": "world"})
		if result != "hello world" {
			t.Errorf("got %q, want %q", result, "hello world")
		}
	})

	t.Run("no vars returns unchanged", func(t *testing.T) {
		result := RenderVars("hello ${name}", nil)
		if result != "hello ${name}" {
			t.Errorf("got %q, want unchanged", result)
		}
	})

	t.Run("map recursion", func(t *testing.T) {
		input := map[string]any{"key": "${val}"}
		result := RenderVars(input, map[string]string{"val": "resolved"}).(map[string]any)
		if result["key"] != "resolved" {
			t.Errorf("got %v", result)
		}
	})

	t.Run("slice recursion", func(t *testing.T) {
		input := []any{"${x}", "${y}"}
		result := RenderVars(input, map[string]string{"x": "a", "y": "b"}).([]any)
		if result[0] != "a" || result[1] != "b" {
			t.Errorf("got %v", result)
		}
	})

	t.Run("non-string passthrough", func(t *testing.T) {
		result := RenderVars(42, map[string]string{"x": "y"})
		if result != 42 {
			t.Errorf("got %v, want 42", result)
		}
	})
}

func TestRenderPlaceholders(t *testing.T) {
	captures := map[string]string{"camp_id": "camp-123"}

	t.Run("string substitution", func(t *testing.T) {
		result, err := RenderPlaceholders("campaign={{camp_id}}", captures)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "campaign=camp-123" {
			t.Errorf("got %q", result)
		}
	})

	t.Run("ref object", func(t *testing.T) {
		input := map[string]any{"ref": "camp_id"}
		result, err := RenderPlaceholders(input, captures)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "camp-123" {
			t.Errorf("got %v", result)
		}
	})

	t.Run("ref not found", func(t *testing.T) {
		input := map[string]any{"ref": "missing"}
		_, err := RenderPlaceholders(input, captures)
		if err == nil {
			t.Fatal("expected error for missing ref")
		}
	})

	t.Run("ref non-string", func(t *testing.T) {
		input := map[string]any{"ref": 42}
		_, err := RenderPlaceholders(input, captures)
		if err == nil {
			t.Fatal("expected error for non-string ref")
		}
	})

	t.Run("map recursion", func(t *testing.T) {
		input := map[string]any{"id": "{{camp_id}}"}
		result, err := RenderPlaceholders(input, captures)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m := result.(map[string]any)
		if m["id"] != "camp-123" {
			t.Errorf("got %v", m)
		}
	})

	t.Run("slice recursion", func(t *testing.T) {
		input := []any{"{{camp_id}}", "static"}
		result, err := RenderPlaceholders(input, captures)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := result.([]any)
		if s[0] != "camp-123" || s[1] != "static" {
			t.Errorf("got %v", s)
		}
	})

	t.Run("non-string passthrough", func(t *testing.T) {
		result, err := RenderPlaceholders(42, captures)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != 42 {
			t.Errorf("got %v", result)
		}
	})
}

func TestLookupJSONPath(t *testing.T) {
	data := map[string]any{
		"result": map[string]any{
			"id":   "abc",
			"list": []any{"x", "y", "z"},
		},
	}

	t.Run("simple path", func(t *testing.T) {
		result, err := LookupJSONPath(data, "result.id")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "abc" {
			t.Errorf("got %v", result)
		}
	})

	t.Run("array index", func(t *testing.T) {
		result, err := LookupJSONPath(data, "result.list[1]")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "y" {
			t.Errorf("got %v", result)
		}
	})

	t.Run("missing field", func(t *testing.T) {
		_, err := LookupJSONPath(data, "result.nonexistent")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("json decode pipe", func(t *testing.T) {
		jsonData := map[string]any{
			"payload": `{"inner": "value"}`,
		}
		result, err := LookupJSONPath(jsonData, "payload|json.inner")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "value" {
			t.Errorf("got %v", result)
		}
	})

	t.Run("json decode on non-string", func(t *testing.T) {
		jsonData := map[string]any{"payload": 42}
		_, err := LookupJSONPath(jsonData, "payload|json.inner")
		if err == nil {
			t.Fatal("expected error for non-string json")
		}
	})
}

func TestLookupDotPath(t *testing.T) {
	t.Run("nested object", func(t *testing.T) {
		data := map[string]any{"a": map[string]any{"b": "c"}}
		result, err := lookupDotPath(data, "a.b")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "c" {
			t.Errorf("got %v", result)
		}
	})

	t.Run("not an object", func(t *testing.T) {
		_, err := lookupDotPath("not-object", "field")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("array out of range", func(t *testing.T) {
		data := map[string]any{"arr": []any{"a"}}
		_, err := lookupDotPath(data, "arr[5]")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("not an array", func(t *testing.T) {
		data := map[string]any{"val": "string"}
		_, err := lookupDotPath(data, "val[0]")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestParseSegment(t *testing.T) {
	tests := []struct {
		name      string
		segment   string
		wantField string
		wantLen   int
		wantErr   bool
	}{
		{"simple field", "name", "name", 0, false},
		{"with index", "items[0]", "items", 1, false},
		{"multiple indices", "matrix[0][1]", "matrix", 2, false},
		{"no field", "[0]", "", 0, true},
		{"unclosed", "items[0", "", 0, true},
		{"non-numeric", "items[abc]", "", 0, true},
		{"invalid syntax", "items[0]x", "", 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			field, indexes, err := parseSegment(tc.segment)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}
			if err != nil {
				return
			}
			if field != tc.wantField {
				t.Errorf("field = %q, want %q", field, tc.wantField)
			}
			if len(indexes) != tc.wantLen {
				t.Errorf("indexes len = %d, want %d", len(indexes), tc.wantLen)
			}
		})
	}
}

func TestDecodeJSONValue(t *testing.T) {
	t.Run("object", func(t *testing.T) {
		result, err := DecodeJSONValue([]byte(`{"key": "value"}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m := result.(map[string]any)
		if m["key"] != "value" {
			t.Errorf("got %v", m)
		}
	})

	t.Run("number preserved", func(t *testing.T) {
		result, err := DecodeJSONValue([]byte(`42`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := result.(json.Number); !ok {
			t.Errorf("expected json.Number, got %T", result)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := DecodeJSONValue([]byte(`{invalid}`))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestValuesEqual(t *testing.T) {
	tests := []struct {
		name     string
		actual   any
		expected any
		want     bool
	}{
		{"same strings", "hello", "hello", true},
		{"different strings", "hello", "world", false},
		{"json number match", json.Number("42"), json.Number("42"), true},
		{"json number to float", float64(42), json.Number("42"), true},
		{"json number to int", 42, json.Number("42"), true},
		{"string from sprint", 42, "42", true},
		{"fallback match", true, true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValuesEqual(tc.actual, tc.expected); got != tc.want {
				t.Errorf("ValuesEqual(%v, %v) = %v, want %v", tc.actual, tc.expected, got, tc.want)
			}
		})
	}
}

func TestAssertArrayContains(t *testing.T) {
	t.Run("not an array", func(t *testing.T) {
		if err := AssertArrayContains("string", "value"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("single match", func(t *testing.T) {
		arr := []any{"a", "b", "c"}
		if err := AssertArrayContains(arr, "b"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("single no match", func(t *testing.T) {
		arr := []any{"a", "b"}
		if err := AssertArrayContains(arr, "z"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("slice match", func(t *testing.T) {
		arr := []any{"a", "b", "c"}
		if err := AssertArrayContains(arr, []any{"a", "c"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("slice missing", func(t *testing.T) {
		arr := []any{"a", "b"}
		if err := AssertArrayContains(arr, []any{"a", "z"}); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestMatchJSONSubset(t *testing.T) {
	t.Run("map subset", func(t *testing.T) {
		actual := map[string]any{"a": "1", "b": "2", "c": "3"}
		expected := map[string]any{"a": "1", "c": "3"}
		if !matchJSONSubset(actual, expected) {
			t.Error("expected match")
		}
	})

	t.Run("map missing field", func(t *testing.T) {
		actual := map[string]any{"a": "1"}
		expected := map[string]any{"a": "1", "b": "2"}
		if matchJSONSubset(actual, expected) {
			t.Error("expected no match")
		}
	})

	t.Run("not a map", func(t *testing.T) {
		if matchJSONSubset("string", map[string]any{"a": "1"}) {
			t.Error("expected no match")
		}
	})

	t.Run("array subset", func(t *testing.T) {
		actual := []any{"a", "b", "c"}
		expected := []any{"a", "c"}
		if !matchJSONSubset(actual, expected) {
			t.Error("expected match")
		}
	})

	t.Run("not an array", func(t *testing.T) {
		if matchJSONSubset("string", []any{"a"}) {
			t.Error("expected no match")
		}
	})
}

func TestCompareJSONNumbers(t *testing.T) {
	t.Run("json number to json number", func(t *testing.T) {
		if !compareJSONNumbers(json.Number("42"), json.Number("42")) {
			t.Error("expected match")
		}
	})

	t.Run("float64 match", func(t *testing.T) {
		if !compareJSONNumbers(float64(3.14), json.Number("3.14")) {
			t.Error("expected match")
		}
	})

	t.Run("int match", func(t *testing.T) {
		if !compareJSONNumbers(int(5), json.Number("5")) {
			t.Error("expected match")
		}
	})

	t.Run("int64 match", func(t *testing.T) {
		if !compareJSONNumbers(int64(5), json.Number("5")) {
			t.Error("expected match")
		}
	})

	t.Run("string fallback", func(t *testing.T) {
		if !compareJSONNumbers("42", json.Number("42")) {
			t.Error("expected match via string")
		}
	})
}

func TestCaptureFromPaths(t *testing.T) {
	data := map[string]any{
		"result": map[string]any{"id": "abc"},
	}

	t.Run("first path succeeds", func(t *testing.T) {
		result, err := CaptureFromPaths(data, []string{"result.id"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "abc" {
			t.Errorf("got %q", result)
		}
	})

	t.Run("fallback path", func(t *testing.T) {
		result, err := CaptureFromPaths(data, []string{"missing.path", "result.id"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "abc" {
			t.Errorf("got %q", result)
		}
	})

	t.Run("no paths", func(t *testing.T) {
		_, err := CaptureFromPaths(data, nil)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("all paths fail", func(t *testing.T) {
		_, err := CaptureFromPaths(data, []string{"missing1", "missing2"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestCaptureHints(t *testing.T) {
	t.Run("non-map", func(t *testing.T) {
		if hints := CaptureHints("string"); hints != nil {
			t.Errorf("expected nil, got %v", hints)
		}
	})

	t.Run("no result", func(t *testing.T) {
		if hints := CaptureHints(map[string]any{}); hints != nil {
			t.Errorf("expected nil, got %v", hints)
		}
	})

	t.Run("result not map", func(t *testing.T) {
		if hints := CaptureHints(map[string]any{"result": "string"}); hints != nil {
			t.Errorf("expected nil, got %v", hints)
		}
	})

	t.Run("structuredContent id", func(t *testing.T) {
		data := map[string]any{
			"result": map[string]any{
				"structuredContent": map[string]any{"id": "123"},
			},
		}
		hints := CaptureHints(data)
		if len(hints) != 1 || hints[0] != "result.structuredContent.id" {
			t.Errorf("unexpected hints: %v", hints)
		}
	})

	t.Run("structured_content id", func(t *testing.T) {
		data := map[string]any{
			"result": map[string]any{
				"structured_content": map[string]any{"id": "123"},
			},
		}
		hints := CaptureHints(data)
		if len(hints) != 1 || hints[0] != "result.structured_content.id" {
			t.Errorf("unexpected hints: %v", hints)
		}
	})
}

func TestFormatCaptureHints(t *testing.T) {
	if got := FormatCaptureHints([]string{"a", "b"}); got != "a, b" {
		t.Errorf("got %q", got)
	}
}

func TestFormatJSONRPCError(t *testing.T) {
	t.Run("non-map", func(t *testing.T) {
		if got := FormatJSONRPCError("string"); got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("no error field", func(t *testing.T) {
		if got := FormatJSONRPCError(map[string]any{"result": "ok"}); got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("with error", func(t *testing.T) {
		data := map[string]any{
			"error": map[string]any{"code": -32600, "message": "invalid"},
		}
		result := FormatJSONRPCError(data)
		if result == "" {
			t.Error("expected non-empty error string")
		}
	})
}

func TestMergeVars(t *testing.T) {
	t.Run("nil base", func(t *testing.T) {
		result := mergeVars(nil, map[string]any{"a": "1"})
		if result["a"] != "1" {
			t.Errorf("got %v", result)
		}
	})

	t.Run("override", func(t *testing.T) {
		base := map[string]string{"a": "1", "b": "2"}
		override := map[string]any{"a": "overridden"}
		result := mergeVars(base, override)
		if result["a"] != "overridden" {
			t.Errorf("got %v", result)
		}
		if result["b"] != "2" {
			t.Errorf("got %v", result)
		}
	})

	t.Run("both nil", func(t *testing.T) {
		result := mergeVars(nil, nil)
		if result != nil {
			t.Errorf("expected nil for empty merge, got %v", result)
		}
	})
}

func TestExpandScenarioSteps(t *testing.T) {
	t.Run("simple steps", func(t *testing.T) {
		steps := []ScenarioStep{
			{Method: "tools/call", Params: map[string]any{"name": "test"}},
		}
		expanded, err := expandScenarioSteps(steps, nil, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(expanded) != 1 {
			t.Errorf("expected 1 step, got %d", len(expanded))
		}
	})

	t.Run("unknown block", func(t *testing.T) {
		steps := []ScenarioStep{{Use: "nonexistent"}}
		_, err := expandScenarioSteps(steps, nil, nil, nil)
		if err == nil {
			t.Fatal("expected error for unknown block")
		}
	})

	t.Run("recursive block", func(t *testing.T) {
		blocks := map[string][]ScenarioStep{
			"loop": {{Use: "loop"}},
		}
		steps := []ScenarioStep{{Use: "loop"}}
		_, err := expandScenarioSteps(steps, blocks, nil, nil)
		if err == nil {
			t.Fatal("expected error for recursive block")
		}
	})

	t.Run("block inlining", func(t *testing.T) {
		blocks := map[string][]ScenarioStep{
			"setup": {
				{Method: "tools/call", Params: map[string]any{"name": "create"}},
			},
		}
		steps := []ScenarioStep{{Use: "setup"}}
		expanded, err := expandScenarioSteps(steps, blocks, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(expanded) != 1 {
			t.Errorf("expected 1 expanded step, got %d", len(expanded))
		}
	})
}
