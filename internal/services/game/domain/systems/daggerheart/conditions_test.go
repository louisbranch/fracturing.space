package daggerheart

import "testing"

func TestNormalizeConditions(t *testing.T) {
	tests := []struct {
		name      string
		input     []string
		want      []string
		wantError bool
	}{
		{
			name:  "empty",
			input: nil,
			want:  []string{},
		},
		{
			name:  "normalizes and orders",
			input: []string{"Hidden", "vulnerable", "hidden", " restrained "},
			want:  []string{ConditionHidden, ConditionRestrained, ConditionVulnerable},
		},
		{
			name:      "rejects unknown",
			input:     []string{"mystery"},
			wantError: true,
		},
		{
			name:      "rejects empty",
			input:     []string{" "},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeConditions(tt.input)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !ConditionsEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDiffConditions(t *testing.T) {
	before := []string{ConditionHidden, ConditionRestrained}
	after := []string{ConditionHidden, ConditionVulnerable}
	added, removed := DiffConditions(before, after)
	if !ConditionsEqual(added, []string{ConditionVulnerable}) {
		t.Fatalf("added = %v, want %v", added, []string{ConditionVulnerable})
	}
	if !ConditionsEqual(removed, []string{ConditionRestrained}) {
		t.Fatalf("removed = %v, want %v", removed, []string{ConditionRestrained})
	}
}
