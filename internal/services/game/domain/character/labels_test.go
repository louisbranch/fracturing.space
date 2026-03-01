package character

import "testing"

func TestNormalizeKind(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      Kind
		wantValid bool
	}{
		{name: "empty", input: " ", want: KindUnspecified, wantValid: false},
		{name: "pc short", input: "PC", want: KindPC, wantValid: true},
		{name: "pc enum", input: "character_kind_pc", want: KindPC, wantValid: true},
		{name: "npc short", input: "npc", want: KindNPC, wantValid: true},
		{name: "npc enum", input: "CHARACTER_KIND_NPC", want: KindNPC, wantValid: true},
		{name: "invalid", input: "boss", want: KindUnspecified, wantValid: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := NormalizeKind(tc.input)
			if got != tc.want || ok != tc.wantValid {
				t.Fatalf("NormalizeKind(%q) = (%q, %v), want (%q, %v)", tc.input, got, ok, tc.want, tc.wantValid)
			}
		})
	}
}
