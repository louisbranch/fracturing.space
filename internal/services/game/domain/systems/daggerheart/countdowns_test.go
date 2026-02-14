package daggerheart

import "testing"

func TestNormalizeCountdownKind(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{"progress", " progress ", CountdownKindProgress, false},
		{"consequence", "CONSEQUENCE", CountdownKindConsequence, false},
		{"empty", "  ", "", true},
		{"invalid", "other", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeCountdownKind(tt.value)
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
				t.Fatalf("kind = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeCountdownDirection(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{"increase", " increase ", CountdownDirectionIncrease, false},
		{"decrease", "DECREASE", CountdownDirectionDecrease, false},
		{"empty", "", "", true},
		{"invalid", "sideways", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeCountdownDirection(tt.value)
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
				t.Fatalf("direction = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplyCountdownUpdate(t *testing.T) {
	tests := []struct {
		name       string
		countdown  Countdown
		delta      int
		override   *int
		wantAfter  int
		wantDelta  int
		wantLooped bool
		wantErr    bool
	}{
		{
			name:      "delta increase",
			countdown: Countdown{Current: 2, Max: 6},
			delta:     2,
			wantAfter: 4,
			wantDelta: 2,
		},
		{
			name:      "delta clamp no loop",
			countdown: Countdown{Current: 5, Max: 6},
			delta:     3,
			wantAfter: 6,
			wantDelta: 1,
		},
		{
			name:       "delta underflow loop",
			countdown:  Countdown{Current: 1, Max: 4, Looping: true},
			delta:      -3,
			wantAfter:  4,
			wantDelta:  3,
			wantLooped: true,
		},
		{
			name:      "override wins",
			countdown: Countdown{Current: 2, Max: 6},
			override:  intPointer(5),
			wantAfter: 5,
			wantDelta: 3,
		},
		{
			name:      "invalid max",
			countdown: Countdown{Current: 0, Max: 0},
			delta:     1,
			wantErr:   true,
		},
		{
			name:      "invalid current",
			countdown: Countdown{Current: 5, Max: 3},
			delta:     1,
			wantErr:   true,
		},
		{
			name:      "missing update",
			countdown: Countdown{Current: 1, Max: 3},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update, err := ApplyCountdownUpdate(tt.countdown, tt.delta, tt.override)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if update.After != tt.wantAfter {
				t.Fatalf("after = %d, want %d", update.After, tt.wantAfter)
			}
			if update.Delta != tt.wantDelta {
				t.Fatalf("delta = %d, want %d", update.Delta, tt.wantDelta)
			}
			if update.Looped != tt.wantLooped {
				t.Fatalf("looped = %v, want %v", update.Looped, tt.wantLooped)
			}
			if update.Countdown.Current != tt.wantAfter {
				t.Fatalf("countdown current = %d, want %d", update.Countdown.Current, tt.wantAfter)
			}
		})
	}
}

func intPointer(value int) *int {
	return &value
}
