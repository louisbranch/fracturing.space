package app

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
	RoleReadOnly   bool                              `json:"roleReadOnly"`
	AccessOptions  []CampaignParticipantAccessOption `json:"accessOptions"`
	AccessReadOnly bool                              `json:"accessReadOnly"`
}

// CampaignAIAgentOption stores one AI binding option state.
type CampaignAIAgentOption struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
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
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Kind           string   `json:"kind"`
	Controller     string   `json:"controller"`
	Pronouns       string   `json:"pronouns"`
	Aliases        []string `json:"aliases"`
	AvatarURL      string   `json:"avatarUrl"`
	CanEdit        bool     `json:"canEdit"`
	EditReasonCode string   `json:"editReasonCode"`
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
	ID              string `json:"id"`
	ParticipantID   string `json:"participantId"`
	RecipientUserID string `json:"recipientUserId"`
	Status          string `json:"status"`
}
