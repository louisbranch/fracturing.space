package character

import (
	"testing"
)

func TestCharacterKindFromLabel(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    CharacterKind
		wantErr bool
	}{
		{name: "short pc", input: "PC", want: CharacterKindPC},
		{name: "prefixed pc", input: "CHARACTER_KIND_PC", want: CharacterKindPC},
		{name: "short npc", input: "NPC", want: CharacterKindNPC},
		{name: "prefixed npc", input: "CHARACTER_KIND_NPC", want: CharacterKindNPC},
		{name: "lowercase pc", input: "pc", want: CharacterKindPC},
		{name: "lowercase npc", input: "npc", want: CharacterKindNPC},
		{name: "whitespace trimmed", input: "  PC  ", want: CharacterKindPC},
		{name: "mixed case", input: "Npc", want: CharacterKindNPC},
		{name: "empty string", input: "", wantErr: true},
		{name: "unknown value", input: "INVALID", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CharacterKindFromLabel(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}
