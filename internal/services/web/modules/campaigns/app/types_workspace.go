package app

import "golang.org/x/text/language"

// CampaignSummary is a transport-safe summary for discovery entrys.
type CampaignSummary struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Theme             string `json:"theme"`
	CoverImageURL     string `json:"coverImageUrl"`
	ParticipantCount  string `json:"participantCount"`
	CharacterCount    string `json:"characterCount"`
	CreatedAtUnixNano int64  `json:"createdAtUnixNano"`
	UpdatedAtUnixNano int64  `json:"updatedAtUnixNano"`
}

// CampaignWorkspace stores campaign details used by campaign workspace routes.
type CampaignWorkspace struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Theme            string `json:"theme"`
	System           string `json:"system"`
	GMMode           string `json:"gmMode"`
	AIAgentID        string `json:"aiAgentId"`
	Status           string `json:"status"`
	Locale           string `json:"locale"`
	Intent           string `json:"intent"`
	AccessPolicy     string `json:"accessPolicy"`
	ParticipantCount string `json:"participantCount"`
	CharacterCount   string `json:"characterCount"`
	CoverPreviewURL  string `json:"coverPreviewUrl"`
	CoverImageURL    string `json:"coverImageUrl"`
}

// CampaignParticipant stores participant details used by campaign participants pages.
type CampaignParticipant struct {
	ID             string `json:"id"`
	UserID         string `json:"userId"`
	Name           string `json:"name"`
	Role           string `json:"role"`
	CampaignAccess string `json:"campaignAccess"`
	Controller     string `json:"controller"`
	Pronouns       string `json:"pronouns"`
	AvatarURL      string `json:"avatarUrl"`
	CanEdit        bool   `json:"canEdit"`
	EditReasonCode string `json:"editReasonCode"`
}

// CampaignParticipantAccessOption stores one campaign-access option state.
type CampaignParticipantAccessOption struct {
	Value   string `json:"value"`
	Allowed bool   `json:"allowed"`
}

// CampaignParticipantEditor stores participant edit page data.
type CampaignParticipantEditor struct {
	Participant    CampaignParticipant               `json:"participant"`
	AllowGMRole    bool                              `json:"allowGMRole"`
	RoleReadOnly   bool                              `json:"roleReadOnly"`
	AccessOptions  []CampaignParticipantAccessOption `json:"accessOptions"`
	AccessReadOnly bool                              `json:"accessReadOnly"`
}

// CampaignParticipantCreator stores participant create page data.
type CampaignParticipantCreator struct {
	Name           string                            `json:"name"`
	Role           string                            `json:"role"`
	CampaignAccess string                            `json:"campaignAccess"`
	AllowGMRole    bool                              `json:"allowGMRole"`
	AccessOptions  []CampaignParticipantAccessOption `json:"accessOptions"`
}

// CampaignAIAgentOption stores one AI binding option state.
type CampaignAIAgentOption struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Enabled  bool   `json:"enabled"`
	Selected bool   `json:"selected"`
}

// CampaignAIBindingEditor stores AI-binding form state for the participant edit page.
type CampaignAIBindingEditor struct {
	Visible     bool                    `json:"visible"`
	Enabled     bool                    `json:"enabled"`
	Unavailable bool                    `json:"unavailable"`
	CurrentID   string                  `json:"currentId"`
	Options     []CampaignAIAgentOption `json:"options"`
}

// CampaignCharacter stores character details used by campaign characters pages.
type CampaignCharacter struct {
	ID                      string                               `json:"id"`
	Name                    string                               `json:"name"`
	Kind                    string                               `json:"kind"`
	Controller              string                               `json:"controller"`
	ControllerParticipantID string                               `json:"controllerParticipantId"`
	Pronouns                string                               `json:"pronouns"`
	Aliases                 []string                             `json:"aliases"`
	AvatarURL               string                               `json:"avatarUrl"`
	OwnedByViewer           bool                                 `json:"ownedByViewer"`
	CanEdit                 bool                                 `json:"canEdit"`
	EditReasonCode          string                               `json:"editReasonCode"`
	Daggerheart             *CampaignCharacterDaggerheartSummary `json:"daggerheart,omitempty"`
}

// CampaignCharacterDaggerheartSummary stores Daggerheart-specific card summary fields.
type CampaignCharacterDaggerheartSummary struct {
	Level         int32  `json:"level"`
	ClassName     string `json:"className"`
	SubclassName  string `json:"subclassName"`
	AncestryName  string `json:"ancestryName"`
	CommunityName string `json:"communityName"`
}

// CharacterReadContext keeps system-aware character read dependencies explicit.
type CharacterReadContext struct {
	System       string
	Locale       language.Tag
	ViewerUserID string
}

// CampaignCharacterEditor stores character edit page data.
type CampaignCharacterEditor struct {
	Character CampaignCharacter `json:"character"`
}

// CampaignCharacterControlOption stores one selectable character-controller option.
type CampaignCharacterControlOption struct {
	ParticipantID string `json:"participantId"`
	Label         string `json:"label"`
	Selected      bool   `json:"selected"`
}

// CampaignCharacterControl stores character-detail control state.
type CampaignCharacterControl struct {
	CurrentParticipantName string                           `json:"currentParticipantName"`
	CanSelfClaim           bool                             `json:"canSelfClaim"`
	CanSelfRelease         bool                             `json:"canSelfRelease"`
	CanManageControl       bool                             `json:"canManageControl"`
	Options                []CampaignCharacterControlOption `json:"options"`
}

// CampaignSession stores session details used by campaign sessions pages.
type CampaignSession struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	StartedAt string `json:"startedAt"`
	UpdatedAt string `json:"updatedAt"`
	EndedAt   string `json:"endedAt"`
}

// CampaignSessionReadinessBlocker stores one session-start readiness blocker.
type CampaignSessionReadinessBlocker struct {
	Code     string            `json:"code"`
	Message  string            `json:"message"`
	Metadata map[string]string `json:"metadata"`
}

// CampaignSessionReadiness stores campaign readiness details for session start.
type CampaignSessionReadiness struct {
	Ready    bool                              `json:"ready"`
	Blockers []CampaignSessionReadinessBlocker `json:"blockers"`
}

// CampaignInvite stores invite details used by campaign invites pages.
type CampaignInvite struct {
	ID                string `json:"id"`
	ParticipantID     string `json:"participantId"`
	ParticipantName   string `json:"participantName"`
	RecipientUserID   string `json:"recipientUserId"`
	RecipientUsername string `json:"recipientUsername"`
	HasRecipient      bool   `json:"hasRecipient"`
	Status            string `json:"status"`
}

// InviteUserSearchResult stores one invite typeahead option.
type InviteUserSearchResult struct {
	UserID        string `json:"user_id"`
	Username      string `json:"username"`
	Name          string `json:"name"`
	AvatarSetID   string `json:"avatar_set_id"`
	AvatarAssetID string `json:"avatar_asset_id"`
	IsContact     bool   `json:"is_contact"`
}
