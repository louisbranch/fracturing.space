package app

// CampaignStarterPreview stores protected starter preview page state.
type CampaignStarterPreview struct {
	EntryID              string
	TemplateCampaignID   string
	Title                string
	Description          string
	CampaignTheme        string
	Hook                 string
	PlaystyleLabel       string
	CharacterName        string
	CharacterSummary     string
	System               string
	Difficulty           string
	Duration             string
	GmMode               string
	Players              string
	Tags                 []string
	AIAgentOptions       []CampaignAIAgentOption
	HasAvailableAIAgents bool
}

// LaunchStarterInput stores the protected starter launch form input.
type LaunchStarterInput struct {
	AIAgentID string
}

// StarterLaunchResult stores the newly created campaign created from a starter fork.
type StarterLaunchResult struct {
	CampaignID string
}
