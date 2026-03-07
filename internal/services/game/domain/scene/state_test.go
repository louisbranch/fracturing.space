package scene

import "testing"

func TestHasPC(t *testing.T) {
	tests := []struct {
		name       string
		characters map[string]bool
		pcs        map[string]bool
		want       bool
	}{
		{
			name:       "matching PC",
			characters: map[string]bool{"c1": true, "c2": true},
			pcs:        map[string]bool{"c2": true, "c3": true},
			want:       true,
		},
		{
			name:       "no matching PC",
			characters: map[string]bool{"c1": true},
			pcs:        map[string]bool{"c2": true},
			want:       false,
		},
		{
			name:       "empty characters",
			characters: nil,
			pcs:        map[string]bool{"c1": true},
			want:       false,
		},
		{
			name:       "empty pcs",
			characters: map[string]bool{"c1": true},
			pcs:        nil,
			want:       false,
		},
		{
			name:       "both empty",
			characters: nil,
			pcs:        nil,
			want:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := State{Characters: tt.characters}
			if got := s.HasPC(tt.pcs); got != tt.want {
				t.Fatalf("HasPC() = %v, want %v", got, tt.want)
			}
		})
	}
}
