package check

import "testing"

func TestMeetsDifficulty(t *testing.T) {
	tests := []struct {
		name       string
		total      int
		difficulty int
		want       bool
	}{
		{"exact match", 10, 10, true},
		{"above difficulty", 15, 10, true},
		{"below difficulty", 5, 10, false},
		{"zero total zero difficulty", 0, 0, true},
		{"negative total", -5, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MeetsDifficulty(tt.total, tt.difficulty)
			if got != tt.want {
				t.Errorf("MeetsDifficulty(%d, %d) = %v, want %v", tt.total, tt.difficulty, got, tt.want)
			}
		})
	}
}

func TestMargin(t *testing.T) {
	tests := []struct {
		name       string
		total      int
		difficulty int
		want       int
	}{
		{"exact match", 10, 10, 0},
		{"above by 5", 15, 10, 5},
		{"below by 5", 5, 10, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Margin(tt.total, tt.difficulty)
			if got != tt.want {
				t.Errorf("Margin(%d, %d) = %v, want %v", tt.total, tt.difficulty, got, tt.want)
			}
		})
	}
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name       string
		total      int
		difficulty int
		want       Result
	}{
		{"success with margin", 15, 10, Result{Success: true, Margin: 5}},
		{"exact success", 10, 10, Result{Success: true, Margin: 0}},
		{"failure", 5, 10, Result{Success: false, Margin: -5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Check(tt.total, tt.difficulty)
			if got != tt.want {
				t.Errorf("Check(%d, %d) = %v, want %v", tt.total, tt.difficulty, got, tt.want)
			}
		})
	}
}
