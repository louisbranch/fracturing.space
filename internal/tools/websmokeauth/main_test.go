package main

import "testing"

func TestSanitizeEmailPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: "web-smoke"},
		{name: "spaces only", input: "   ", want: "web-smoke"},
		{name: "alpha numeric", input: "Web Smoke 123", want: "web-smoke-123"},
		{name: "symbol collapse", input: "a***b___c", want: "a-b-c"},
		{name: "leading trailing symbols", input: "---smoke---", want: "smoke"},
		{name: "no ascii letters", input: "测试", want: "web-smoke"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := sanitizeEmailPrefix(tc.input); got != tc.want {
				t.Fatalf("sanitizeEmailPrefix(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
