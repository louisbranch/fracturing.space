// File campaigns.go defines view data for campaign templates.
package templates

// CampaignRow holds formatted campaign data for display.
type CampaignRow struct {
	// ID is the unique identifier for the campaign.
	ID string
	// Name is the display name of the campaign.
	Name string
	// GMMode is the display label for the GM mode.
	GMMode string
	// ParticipantCount is the formatted number of participants.
	ParticipantCount string
	// CharacterCount is the formatted number of characters.
	CharacterCount string
	// ThemePrompt is the truncated theme prompt text.
	ThemePrompt string
	// CreatedDate is the formatted creation date.
	CreatedDate string
}

// CampaignDetail holds formatted campaign data for the detail page.
type CampaignDetail struct {
	// ID is the unique identifier for the campaign.
	ID string
	// Name is the display name of the campaign.
	Name string
	// GMMode is the display label for the GM mode.
	GMMode string
	// ParticipantCount is the formatted number of participants.
	ParticipantCount string
	// CharacterCount is the formatted number of characters.
	CharacterCount string
	// ThemePrompt is the theme prompt text.
	ThemePrompt string
	// GMFear is the formatted GM fear value.
	GMFear string
	// CreatedAt is the formatted creation timestamp.
	CreatedAt string
	// UpdatedAt is the formatted update timestamp.
	UpdatedAt string
}
