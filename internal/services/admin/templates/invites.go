package templates

// InviteRow represents a single row in the invites table.
type InviteRow struct {
	ID            string
	CampaignID    string
	CampaignName  string
	Participant   string
	Recipient     string
	Status        string
	StatusVariant string
	CreatedAt     string
	UpdatedAt     string
}
