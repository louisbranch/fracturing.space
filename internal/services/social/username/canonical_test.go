package username

import "testing"

func TestCanonicalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		want      string
		wantError bool
	}{
		{
			name:  "lowercases and preserves valid separators",
			input: "Alice_One",
			want:  "alice_one",
		},
		{
			name:  "trims spaces before validation",
			input: "  Bob-User  ",
			want:  "bob-user",
		},
		{
			name:      "rejects empty",
			input:     "",
			wantError: true,
		},
		{
			name:      "rejects non ascii",
			input:     "álvaro",
			wantError: true,
		},
		{
			name:      "rejects format mismatch",
			input:     "__",
			wantError: true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := Canonicalize(test.input)
			if test.wantError {
				if err == nil {
					t.Fatalf("Canonicalize(%q) error = nil, want non-nil", test.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("Canonicalize(%q) error = %v", test.input, err)
			}
			if got != test.want {
				t.Fatalf("Canonicalize(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestNormalizeSearchQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "strips handle punctuation and lowercases",
			input: "  @Al!  ",
			want:  "al",
		},
		{
			name:  "preserves valid separators",
			input: "Alice_One-2",
			want:  "alice_one-2",
		},
		{
			name:  "drops whitespace inside query",
			input: "@alice one",
			want:  "aliceone",
		},
		{
			name:  "returns empty for punctuation only",
			input: "@!!",
			want:  "",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := NormalizeSearchQuery(test.input); got != test.want {
				t.Fatalf("NormalizeSearchQuery(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}
