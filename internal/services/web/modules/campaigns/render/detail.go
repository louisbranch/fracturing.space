package render

// DetailView is the module-owned render model for campaign detail pages.
type DetailView struct {
	Marker                   string
	CampaignID               string
	SessionID                string
	ParticipantID            string
	CharacterID              string
	Name                     string
	Theme                    string
	System                   string
	GMMode                   string
	Status                   string
	Locale                   string
	LocaleValue              string
	Intent                   string
	AccessPolicy             string
	ActionsLocked            bool
	CanEditCampaign          bool
	CanManageParticipants    bool
	CanManageInvites         bool
	CanCreateCharacter       bool
	Participants             []ParticipantView
	ParticipantCreator       ParticipantCreatorView
	ParticipantEditor        ParticipantEditorView
	AIBindingEditor          AIBindingEditorView
	Characters               []CharacterView
	CharacterEditor          CharacterEditorView
	CharacterControl         CharacterControlView
	Sessions                 []SessionView
	SessionReadiness         SessionReadinessView
	Invites                  []InviteView
	InviteSeatOptions        []InviteSeatOptionView
	CharacterCreationEnabled bool
	CharacterCreation        CampaignCharacterCreationView
}

// ParticipantView carries participant rows without forcing handlers to depend
// on shared template models directly.
type ParticipantView struct {
	ID             string
	Name           string
	Role           string
	CampaignAccess string
	Controller     string
	Pronouns       string
	AvatarURL      string
	IsViewer       bool
	CanEdit        bool
	EditReasonCode string
}

// ParticipantAccessOptionView keeps participant-access select options local to
// the campaigns detail render seam.
type ParticipantAccessOptionView struct {
	Value   string
	Allowed bool
}

// AIAgentOptionView carries AI agent choices for participant AI-binding forms.
type AIAgentOptionView struct {
	ID       string
	Name     string
	Enabled  bool
	Selected bool
}

// AIBindingEditorView keeps AI-binding form state local to campaigns detail
// rendering.
type AIBindingEditorView struct {
	Visible     bool
	Enabled     bool
	Unavailable bool
	CurrentID   string
	Options     []AIAgentOptionView
}

// ParticipantEditorView carries participant edit form state for campaign
// detail pages.
type ParticipantEditorView struct {
	ID             string
	Name           string
	Role           string
	Controller     string
	Pronouns       string
	CampaignAccess string
	AllowGMRole    bool
	RoleReadOnly   bool
	AccessOptions  []ParticipantAccessOptionView
	AccessReadOnly bool
}

// ParticipantCreatorView carries participant creation form state for campaign
// detail pages.
type ParticipantCreatorView struct {
	Name           string
	Role           string
	CampaignAccess string
	AllowGMRole    bool
	AccessOptions  []ParticipantAccessOptionView
}

// CharacterView carries character summary rows for campaign detail pages.
type CharacterView struct {
	ID                      string
	Name                    string
	Kind                    string
	Controller              string
	ControllerParticipantID string
	Pronouns                string
	Aliases                 []string
	AvatarURL               string
	OwnedByViewer           bool
	CanEdit                 bool
	EditReasonCode          string
	Daggerheart             *CharacterDaggerheartSummaryView
}

// CharacterDaggerheartSummaryView keeps optional Daggerheart card metadata
// local to the campaigns detail render seam.
type CharacterDaggerheartSummaryView struct {
	Level         int32
	ClassName     string
	SubclassName  string
	AncestryName  string
	CommunityName string
}

// CharacterEditorView carries character edit form state for campaign detail
// pages.
type CharacterEditorView struct {
	ID       string
	Name     string
	Pronouns string
	Kind     string
}

// CharacterControlOptionView carries control-target options for one character.
type CharacterControlOptionView struct {
	ParticipantID string
	Label         string
	Selected      bool
}

// CharacterControlView carries character control affordances for the selected
// character detail page.
type CharacterControlView struct {
	CurrentParticipantName string
	CanSelfClaim           bool
	CanSelfRelease         bool
	CanManageControl       bool
	Options                []CharacterControlOptionView
}

// SessionView carries session rows for campaign detail pages.
type SessionView struct {
	ID        string
	Name      string
	Status    string
	StartedAt string
	UpdatedAt string
	EndedAt   string
}

// SessionReadinessBlockerView preserves readiness blocker copy for the session
// start affordance.
type SessionReadinessBlockerView struct {
	Code    string
	Message string
}

// SessionReadinessView carries session start state for campaigns detail pages.
type SessionReadinessView struct {
	Ready    bool
	Blockers []SessionReadinessBlockerView
}

// InviteView carries invite rows for the invites detail page.
type InviteView struct {
	ID                string
	ParticipantID     string
	ParticipantName   string
	RecipientUsername string
	HasRecipient      bool
	PublicURL         string
	Status            string
}

// InviteSeatOptionView carries eligible invite-seat targets for invite forms.
type InviteSeatOptionView struct {
	ParticipantID string
	Label         string
}
