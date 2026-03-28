package debugtrace

import "testing"

func TestParseStatus(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want Status
	}{
		{name: "running", raw: "running", want: StatusRunning},
		{name: "succeeded trimmed", raw: " Succeeded ", want: StatusSucceeded},
		{name: "failed mixed case", raw: "FaILeD", want: StatusFailed},
		{name: "unknown", raw: "other", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseStatus(tt.raw); got != tt.want {
				t.Fatalf("ParseStatus(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestParseEntryKind(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want EntryKind
	}{
		{name: "model response", raw: "model_response", want: EntryKindModelResponse},
		{name: "tool call trimmed", raw: " tool_call ", want: EntryKindToolCall},
		{name: "tool result mixed case", raw: "TOOL_RESULT", want: EntryKindToolResult},
		{name: "unknown", raw: "other", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseEntryKind(tt.raw); got != tt.want {
				t.Fatalf("ParseEntryKind(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
