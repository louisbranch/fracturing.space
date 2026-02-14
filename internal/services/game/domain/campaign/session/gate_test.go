package session

import "testing"

func TestNormalizeGateType(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{"trim and lower", "  Investigation ", "investigation", false},
		{"empty", "  ", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeGateType(tt.value)
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
				t.Fatalf("gate type = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeGateReason(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"trim", "  because ", "because"},
		{"empty", "  ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeGateReason(tt.value); got != tt.want {
				t.Fatalf("gate reason = %q, want %q", got, tt.want)
			}
		})
	}
}
