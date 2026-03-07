package modulecompose

import "testing"

func TestValidatePrefix(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		wantErr bool
	}{
		{name: "valid", prefix: "/app/", wantErr: false},
		{name: "empty", prefix: "", wantErr: true},
		{name: "leading whitespace", prefix: " /app/", wantErr: true},
		{name: "missing leading slash", prefix: "app/", wantErr: true},
		{name: "missing trailing slash", prefix: "/app", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePrefix(tt.prefix)
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
