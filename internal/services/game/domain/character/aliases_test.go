package character

import (
	"reflect"
	"testing"
)

func TestNormalizeAliases(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{name: "empty input", input: nil, want: nil},
		{name: "all blank", input: []string{" ", "\t"}, want: nil},
		{
			name:  "trim and dedupe preserving first occurrence",
			input: []string{" Aria ", "Aria", " ", "Nova", "Nova", " Aria"},
			want:  []string{"Aria", "Nova"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeAliases(tc.input); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("normalizeAliases(%v) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestNormalizeAliasesField(t *testing.T) {
	got, err := normalizeAliasesField("  [\" Aria \",\"Aria\", \"\", \"Nova\"] ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, []string{"Aria", "Nova"}) {
		t.Fatalf("normalizeAliasesField() = %v, want %v", got, []string{"Aria", "Nova"})
	}

	got, err = normalizeAliasesField(" ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("normalizeAliasesField(empty) = %v, want nil", got)
	}

	_, err = normalizeAliasesField("{\"not\":\"array\"}")
	if err == nil {
		t.Fatal("expected json array validation error")
	}
	if err.Error() != "aliases must be a JSON array of strings" {
		t.Fatalf("error = %q, want %q", err.Error(), "aliases must be a JSON array of strings")
	}
}
