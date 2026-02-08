package templates

// DashboardStats holds aggregate statistics for the dashboard.
type DashboardStats struct {
	TotalCampaigns    string
	ActiveSessions    string
	TotalCharacters   string
	TotalParticipants string
}

// ActivityEvent represents a recent event for the activity feed.
type ActivityEvent struct {
	CampaignID   string
	CampaignName string
	EventType    string
	Timestamp    string
	Description  string
}
