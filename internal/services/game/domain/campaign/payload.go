package campaign

// CreatePayload captures the payload for campaign.create commands and campaign.created events.
type CreatePayload struct {
	Name         string `json:"name"`
	Locale       string `json:"locale"`
	GameSystem   string `json:"game_system"`
	GmMode       string `json:"gm_mode"`
	Intent       string `json:"intent,omitempty"`
	AccessPolicy string `json:"access_policy,omitempty"`
	ThemePrompt  string `json:"theme_prompt,omitempty"`
	CoverAssetID string `json:"cover_asset_id,omitempty"`
	CoverSetID   string `json:"cover_set_id,omitempty"`
}

// UpdatePayload captures the payload for campaign.update commands and campaign.updated events.
type UpdatePayload struct {
	Fields map[string]string `json:"fields"`
}

// ForkPayload captures the payload for campaign.fork commands and campaign.forked events.
type ForkPayload struct {
	ParentCampaignID string `json:"parent_campaign_id"`
	ForkEventSeq     uint64 `json:"fork_event_seq"`
	OriginCampaignID string `json:"origin_campaign_id"`
	CopyParticipants bool   `json:"copy_participants"`
}
