package naming

import "testing"

func TestNormalizeSystemNamespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "whitespace only", input: "   ", want: ""},
		{name: "simple lowercase", input: "alpha", want: "alpha"},
		{name: "mixed case", input: "Alpha", want: "alpha"},
		{name: "with legacy prefix", input: "GAME_SYSTEM_ALPHA", want: "alpha"},
		{name: "legacy prefix case insensitive", input: "game_system_beta", want: "beta"},
		{name: "hyphens become underscores", input: "my-system", want: "my_system"},
		{name: "consecutive specials collapse", input: "my--system", want: "my_system"},
		{name: "leading trailing specials trimmed", input: "-alpha-", want: "alpha"},
		{name: "digits preserved", input: "system1", want: "system1"},
		{name: "legacy prefix with hyphens", input: "GAME_SYSTEM_My-System", want: "my_system"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeSystemNamespace(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeSystemNamespace(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateSystemNamespace(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		systemID string
		wantErr  bool
	}{
		{name: "match", typeName: "sys.alpha.action.tested", systemID: "GAME_SYSTEM_ALPHA", wantErr: false},
		{name: "mismatch", typeName: "sys.alpha.action.tested", systemID: "GAME_SYSTEM_BETA", wantErr: true},
		{name: "non-system type", typeName: "campaign.created", systemID: "GAME_SYSTEM_ALPHA", wantErr: false},
		{name: "empty systemID", typeName: "sys.alpha.action.tested", systemID: "", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSystemNamespace(tt.typeName, tt.systemID)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestNamespaceFromType(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
		wantOK bool
	}{
		{name: "system prefixed", input: "sys.alpha.action.tested", want: "alpha", wantOK: true},
		{name: "core event", input: "campaign.created", want: "", wantOK: false},
		{name: "too few parts", input: "sys.alpha", want: "", wantOK: false},
		{name: "empty", input: "", want: "", wantOK: false},
		{name: "non-sys prefix", input: "core.alpha.tested", want: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := NamespaceFromType(tt.input)
			if ok != tt.wantOK {
				t.Errorf("NamespaceFromType(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("NamespaceFromType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
