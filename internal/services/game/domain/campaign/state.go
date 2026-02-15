package campaign

// State captures the campaign facts derived from domain events.
type State struct {
	Created     bool
	Name        string
	GameSystem  string
	GmMode      string
	Status      Status
	ThemePrompt string
}
