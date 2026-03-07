package bridge

import "testing"

func TestNormalizeSystemID(t *testing.T) {
	tests := []struct {
		input string
		want  SystemID
		ok    bool
	}{
		{"daggerheart", SystemIDDaggerheart, true},
		{"DAGGERHEART", SystemIDDaggerheart, true},
		{"GAME_SYSTEM_DAGGERHEART", SystemIDDaggerheart, true},
		{"  daggerheart  ", SystemIDDaggerheart, true},
		{"", SystemIDUnspecified, false},
		{"   ", SystemIDUnspecified, false},
		{"unknown", SystemIDUnspecified, false},
	}
	for _, tt := range tests {
		got, ok := NormalizeSystemID(tt.input)
		if got != tt.want || ok != tt.ok {
			t.Errorf("NormalizeSystemID(%q) = (%q, %v), want (%q, %v)", tt.input, got, ok, tt.want, tt.ok)
		}
	}
}

func TestSystemIDString(t *testing.T) {
	if got := SystemIDDaggerheart.String(); got != "daggerheart" {
		t.Errorf("SystemIDDaggerheart.String() = %q, want %q", got, "daggerheart")
	}
}
