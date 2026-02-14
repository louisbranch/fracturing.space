package templates

// DashboardStats holds aggregate statistics for the dashboard.
type DashboardStats struct {
	TotalSystems      string
	TotalCampaigns    string
	TotalSessions     string
	TotalCharacters   string
	TotalParticipants string
	TotalUsers        string
}

// ActivityEvent represents a recent event for the activity feed.
type ActivityEvent struct {
	CampaignID   string
	CampaignName string
	EventType    string
	Timestamp    string
	Description  string
}
