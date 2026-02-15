package check

// MeetsDifficulty returns true if total >= difficulty.
// This is the most common difficulty check in tabletop RPGs.
func MeetsDifficulty(total, difficulty int) bool {
	return total >= difficulty
}

// Margin calculates the margin of success or failure.
// Positive values indicate success, negative indicate failure.
func Margin(total, difficulty int) int {
	return total - difficulty
}

// Result represents the outcome of a difficulty check.
type Result struct {
	Success bool
	Margin  int
}

// Check performs a difficulty check and returns the result.
func Check(total, difficulty int) Result {
	return Result{
		Success: MeetsDifficulty(total, difficulty),
		Margin:  Margin(total, difficulty),
	}
}
