package templates

// SessionDetail holds data for rendering a session detail page.
type SessionDetail struct {
	CampaignID   string
	CampaignName string
	ID           string
	Name         string
	Status       string
	StatusBadge  string
	StartedAt    string
	EndedAt      string
	EventCount   int32
	Events       []EventRow
	NextToken    string
	PrevToken    string
}
