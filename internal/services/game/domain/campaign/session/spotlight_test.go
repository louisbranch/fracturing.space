package session

import "testing"

func TestNormalizeSpotlightType(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    SpotlightType
		wantErr bool
	}{
		{"gm", " GM ", SpotlightTypeGM, false},
		{"character", "character", SpotlightTypeCharacter, false},
		{"empty", "  ", "", true},
		{"invalid", "npc", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeSpotlightType(tt.value)
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
				t.Fatalf("spotlight type = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateSpotlightTarget(t *testing.T) {
	tests := []struct {
		name      string
		spotType  SpotlightType
		character string
		wantErr   bool
	}{
		{"gm without character", SpotlightTypeGM, "", false},
		{"gm with character", SpotlightTypeGM, "char-1", true},
		{"character with id", SpotlightTypeCharacter, "char-1", false},
		{"character missing id", SpotlightTypeCharacter, " ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSpotlightTarget(tt.spotType, tt.character)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
