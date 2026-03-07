package scene

import "testing"

func TestNormalizeSpotlightType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    SpotlightType
		wantErr bool
	}{
		{name: "gm", input: "gm", want: SpotlightTypeGM},
		{name: "GM uppercase", input: "GM", want: SpotlightTypeGM},
		{name: "character", input: "character", want: SpotlightTypeCharacter},
		{name: "padded", input: "  character  ", want: SpotlightTypeCharacter},
		{name: "empty", input: "", wantErr: true},
		{name: "unsupported", input: "npc", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeSpotlightType(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeGateType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "valid", input: "Decision", want: "decision"},
		{name: "padded", input: "  GM_Consequence  ", want: "gm_consequence"},
		{name: "empty", input: "", wantErr: true},
		{name: "spaces only", input: "   ", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeGateType(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
