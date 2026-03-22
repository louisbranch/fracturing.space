package render

// CampaignDetailBaseView carries the workspace fields shared by campaign
// detail render surfaces.
type CampaignDetailBaseView struct {
	CampaignID            string
	Name                  string
	Theme                 string
	System                string
	GMMode                string
	Status                string
	Locale                string
	LocaleValue           string
	Intent                string
	AccessPolicy          string
	ActionsLocked         bool
	CanEditCampaign       bool
	CanManageParticipants bool
	CanManageSession      bool
	CanManageInvites      bool
	CanCreateCharacter    bool
}
