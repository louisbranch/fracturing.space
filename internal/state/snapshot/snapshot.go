package snapshot

// Snapshot represents the system-agnostic continuity state for a campaign.
// Character-level state (HP, Hope, Stress) is now system-specific.
// See storage.DaggerheartCharacterState for Daggerheart character state.
type Snapshot struct {
	CampaignID string
	GmFear     GmFear
}

// NewSnapshot creates a new snapshot for a campaign.
func NewSnapshot(campaignID string) Snapshot {
	return Snapshot{
		CampaignID: campaignID,
		GmFear: GmFear{
			CampaignID: campaignID,
			Value:      0,
		},
	}
}

// SetGmFear sets the GM fear value.
func (s *Snapshot) SetGmFear(value int) {
	s.GmFear.Value = value
}
