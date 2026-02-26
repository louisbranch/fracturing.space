package domain

// CampaignCreateInput represents the MCP tool input for campaign creation.
type CampaignCreateInput struct {
	Name         string `json:"name" jsonschema:"campaign name"`
	System       string `json:"system" jsonschema:"game system (DAGGERHEART)"`
	GmMode       string `json:"gm_mode" jsonschema:"gm mode (HUMAN, AI, HYBRID)"`
	Intent       string `json:"intent,omitempty" jsonschema:"campaign intent (STANDARD, STARTER, SANDBOX)"`
	AccessPolicy string `json:"access_policy,omitempty" jsonschema:"campaign access policy (PRIVATE, RESTRICTED, PUBLIC)"`
	ThemePrompt  string `json:"theme_prompt,omitempty" jsonschema:"optional theme prompt"`
	UserID       string `json:"user_id,omitempty" jsonschema:"creator user identifier"`
}

// CampaignStatusChangeInput represents the MCP tool input for campaign lifecycle changes.
type CampaignStatusChangeInput struct {
	CampaignID string `json:"campaign_id,omitempty" jsonschema:"campaign identifier (defaults to context)"`
}

// CampaignCreateResult represents the MCP tool output for campaign creation.
type CampaignCreateResult struct {
	ID                 string `json:"id" jsonschema:"campaign identifier"`
	OwnerParticipantID string `json:"owner_participant_id" jsonschema:"owner participant identifier for setting context"`
	Name               string `json:"name" jsonschema:"campaign name"`
	GmMode             string `json:"gm_mode" jsonschema:"gm mode"`
	Intent             string `json:"intent" jsonschema:"campaign intent"`
	AccessPolicy       string `json:"access_policy" jsonschema:"campaign access policy"`
	ParticipantCount   int    `json:"participant_count" jsonschema:"number of all participants (GM + PLAYER + future roles)"`
	CharacterCount     int    `json:"character_count" jsonschema:"number of all characters (PC + NPC + future kinds)"`
	GmFear             int    `json:"gm_fear" jsonschema:"campaign-scoped GM fear"`
	ThemePrompt        string `json:"theme_prompt" jsonschema:"theme prompt"`
	Status             string `json:"status" jsonschema:"campaign status"`
	CreatedAt          string `json:"created_at" jsonschema:"RFC3339 timestamp when campaign was created"`
	UpdatedAt          string `json:"updated_at" jsonschema:"RFC3339 timestamp when campaign was last updated"`
	CompletedAt        string `json:"completed_at,omitempty" jsonschema:"RFC3339 timestamp when campaign was completed"`
	ArchivedAt         string `json:"archived_at,omitempty" jsonschema:"RFC3339 timestamp when campaign was archived"`
}

// CampaignStatusResult represents the MCP tool output for campaign lifecycle changes.
type CampaignStatusResult struct {
	ID               string `json:"id" jsonschema:"campaign identifier"`
	Name             string `json:"name" jsonschema:"campaign name"`
	GmMode           string `json:"gm_mode" jsonschema:"gm mode"`
	Intent           string `json:"intent" jsonschema:"campaign intent"`
	AccessPolicy     string `json:"access_policy" jsonschema:"campaign access policy"`
	ParticipantCount int    `json:"participant_count" jsonschema:"number of all participants (GM + PLAYER + future roles)"`
	CharacterCount   int    `json:"character_count" jsonschema:"number of all characters (PC + NPC + future kinds)"`
	GmFear           int    `json:"gm_fear" jsonschema:"campaign-scoped GM fear"`
	ThemePrompt      string `json:"theme_prompt" jsonschema:"theme prompt"`
	Status           string `json:"status" jsonschema:"campaign status"`
	CreatedAt        string `json:"created_at" jsonschema:"RFC3339 timestamp when campaign was created"`
	UpdatedAt        string `json:"updated_at" jsonschema:"RFC3339 timestamp when campaign was last updated"`
	CompletedAt      string `json:"completed_at,omitempty" jsonschema:"RFC3339 timestamp when campaign was completed"`
	ArchivedAt       string `json:"archived_at,omitempty" jsonschema:"RFC3339 timestamp when campaign was archived"`
}

// CampaignListEntry represents a readable campaign metadata entry.
type CampaignListEntry struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Status           string `json:"status"`
	GmMode           string `json:"gm_mode"`
	Intent           string `json:"intent"`
	AccessPolicy     string `json:"access_policy"`
	ParticipantCount int    `json:"participant_count"`
	CharacterCount   int    `json:"character_count"`
	GmFear           int    `json:"gm_fear"`
	ThemePrompt      string `json:"theme_prompt"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
	CompletedAt      string `json:"completed_at,omitempty"`
	ArchivedAt       string `json:"archived_at,omitempty"`
}

// CampaignListPayload represents the MCP resource payload for campaign listings.
type CampaignListPayload struct {
	Campaigns []CampaignListEntry `json:"campaigns"`
}

// CampaignPayload represents the MCP resource payload for a single campaign.
type CampaignPayload struct {
	Campaign CampaignListEntry `json:"campaign"`
}

// ParticipantCreateInput represents the MCP tool input for participant creation.
type ParticipantCreateInput struct {
	CampaignID string `json:"campaign_id" jsonschema:"campaign identifier"`
	UserID     string `json:"user_id,omitempty" jsonschema:"optional user id bound to participant"`
	Name       string `json:"name" jsonschema:"display name for the participant"`
	Role       string `json:"role" jsonschema:"participant role (GM, PLAYER)"`
	Controller string `json:"controller,omitempty" jsonschema:"controller type (HUMAN, AI); optional, defaults to HUMAN if unspecified"`
	Pronouns   string `json:"pronouns,omitempty" jsonschema:"optional participant pronouns"`
}

// ParticipantUpdateInput represents the MCP tool input for participant updates.
type ParticipantUpdateInput struct {
	CampaignID    string  `json:"campaign_id" jsonschema:"campaign identifier"`
	ParticipantID string  `json:"participant_id" jsonschema:"participant identifier"`
	Name          *string `json:"name,omitempty" jsonschema:"optional display name"`
	Role          *string `json:"role,omitempty" jsonschema:"optional participant role (GM, PLAYER)"`
	Controller    *string `json:"controller,omitempty" jsonschema:"optional controller (HUMAN, AI)"`
	Pronouns      *string `json:"pronouns,omitempty" jsonschema:"optional participant pronouns"`
}

// ParticipantDeleteInput represents the MCP tool input for participant deletion.
type ParticipantDeleteInput struct {
	CampaignID    string `json:"campaign_id" jsonschema:"campaign identifier"`
	ParticipantID string `json:"participant_id" jsonschema:"participant identifier"`
	Reason        string `json:"reason,omitempty" jsonschema:"optional reason for deletion"`
}

// ParticipantCreateResult represents the MCP tool output for participant creation.
type ParticipantCreateResult struct {
	ID         string `json:"id" jsonschema:"participant identifier"`
	CampaignID string `json:"campaign_id" jsonschema:"campaign identifier"`
	Name       string `json:"name" jsonschema:"display name for the participant"`
	Role       string `json:"role" jsonschema:"participant role"`
	Controller string `json:"controller" jsonschema:"controller type"`
	Pronouns   string `json:"pronouns" jsonschema:"participant pronouns"`
	CreatedAt  string `json:"created_at" jsonschema:"RFC3339 timestamp when participant was created"`
	UpdatedAt  string `json:"updated_at" jsonschema:"RFC3339 timestamp when participant was last updated"`
}

// ParticipantUpdateResult represents the MCP tool output for participant updates.
type ParticipantUpdateResult struct {
	ID         string `json:"id" jsonschema:"participant identifier"`
	CampaignID string `json:"campaign_id" jsonschema:"campaign identifier"`
	Name       string `json:"name" jsonschema:"display name for the participant"`
	Role       string `json:"role" jsonschema:"participant role"`
	Controller string `json:"controller" jsonschema:"controller type"`
	Pronouns   string `json:"pronouns" jsonschema:"participant pronouns"`
	CreatedAt  string `json:"created_at" jsonschema:"RFC3339 timestamp when participant was created"`
	UpdatedAt  string `json:"updated_at" jsonschema:"RFC3339 timestamp when participant was last updated"`
}

// ParticipantDeleteResult represents the MCP tool output for participant deletion.
type ParticipantDeleteResult struct {
	ID         string `json:"id" jsonschema:"participant identifier"`
	CampaignID string `json:"campaign_id" jsonschema:"campaign identifier"`
	Name       string `json:"name" jsonschema:"display name for the participant"`
	Role       string `json:"role" jsonschema:"participant role"`
	Controller string `json:"controller" jsonschema:"controller type"`
	Pronouns   string `json:"pronouns" jsonschema:"participant pronouns"`
	CreatedAt  string `json:"created_at" jsonschema:"RFC3339 timestamp when participant was created"`
	UpdatedAt  string `json:"updated_at" jsonschema:"RFC3339 timestamp when participant was last updated"`
}

// ParticipantListEntry represents a readable participant entry.
type ParticipantListEntry struct {
	ID         string `json:"id"`
	CampaignID string `json:"campaign_id"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	Controller string `json:"controller"`
	Pronouns   string `json:"pronouns"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// ParticipantListPayload represents the MCP resource payload for participant listings.
type ParticipantListPayload struct {
	Participants []ParticipantListEntry `json:"participants"`
}

// CharacterListEntry represents a readable character entry.
type CharacterListEntry struct {
	ID         string   `json:"id"`
	CampaignID string   `json:"campaign_id"`
	Name       string   `json:"name"`
	Kind       string   `json:"kind"`
	Notes      string   `json:"notes"`
	Pronouns   string   `json:"pronouns"`
	Aliases    []string `json:"aliases"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
}

// CharacterListPayload represents the MCP resource payload for character listings.
type CharacterListPayload struct {
	Characters []CharacterListEntry `json:"characters"`
}

// CharacterCreateInput represents the MCP tool input for character creation.
type CharacterCreateInput struct {
	CampaignID string   `json:"campaign_id" jsonschema:"campaign identifier"`
	Name       string   `json:"name" jsonschema:"display name for the character"`
	Kind       string   `json:"kind" jsonschema:"character kind (PC, NPC)"`
	Notes      string   `json:"notes,omitempty" jsonschema:"optional free-form notes about the character"`
	Pronouns   string   `json:"pronouns,omitempty" jsonschema:"optional character pronouns"`
	Aliases    []string `json:"aliases,omitempty" jsonschema:"optional character aliases"`
}

// CharacterCreateResult represents the MCP tool output for character creation.
type CharacterCreateResult struct {
	ID         string   `json:"id" jsonschema:"character identifier"`
	CampaignID string   `json:"campaign_id" jsonschema:"campaign identifier"`
	Name       string   `json:"name" jsonschema:"display name for the character"`
	Kind       string   `json:"kind" jsonschema:"character kind"`
	Notes      string   `json:"notes" jsonschema:"free-form notes about the character"`
	Pronouns   string   `json:"pronouns" jsonschema:"character pronouns"`
	Aliases    []string `json:"aliases" jsonschema:"character aliases"`
	CreatedAt  string   `json:"created_at" jsonschema:"RFC3339 timestamp when character was created"`
	UpdatedAt  string   `json:"updated_at" jsonschema:"RFC3339 timestamp when character was last updated"`
}

// CharacterUpdateInput represents the MCP tool input for character updates.
type CharacterUpdateInput struct {
	CampaignID  string    `json:"campaign_id" jsonschema:"campaign identifier"`
	CharacterID string    `json:"character_id" jsonschema:"character identifier"`
	Name        *string   `json:"name,omitempty" jsonschema:"optional display name for the character"`
	Kind        *string   `json:"kind,omitempty" jsonschema:"optional character kind (PC, NPC)"`
	Notes       *string   `json:"notes,omitempty" jsonschema:"optional free-form notes about the character"`
	Pronouns    *string   `json:"pronouns,omitempty" jsonschema:"optional character pronouns"`
	Aliases     *[]string `json:"aliases,omitempty" jsonschema:"optional character aliases"`
}

// CharacterUpdateResult represents the MCP tool output for character updates.
type CharacterUpdateResult struct {
	ID         string   `json:"id" jsonschema:"character identifier"`
	CampaignID string   `json:"campaign_id" jsonschema:"campaign identifier"`
	Name       string   `json:"name" jsonschema:"display name for the character"`
	Kind       string   `json:"kind" jsonschema:"character kind"`
	Notes      string   `json:"notes" jsonschema:"free-form notes about the character"`
	Pronouns   string   `json:"pronouns" jsonschema:"character pronouns"`
	Aliases    []string `json:"aliases" jsonschema:"character aliases"`
	CreatedAt  string   `json:"created_at" jsonschema:"RFC3339 timestamp when character was created"`
	UpdatedAt  string   `json:"updated_at" jsonschema:"RFC3339 timestamp when character was last updated"`
}

// CharacterDeleteInput represents the MCP tool input for character deletion.
type CharacterDeleteInput struct {
	CampaignID  string `json:"campaign_id" jsonschema:"campaign identifier"`
	CharacterID string `json:"character_id" jsonschema:"character identifier"`
	Reason      string `json:"reason,omitempty" jsonschema:"optional reason for deletion"`
}

// CharacterDeleteResult represents the MCP tool output for character deletion.
type CharacterDeleteResult struct {
	ID         string   `json:"id" jsonschema:"character identifier"`
	CampaignID string   `json:"campaign_id" jsonschema:"campaign identifier"`
	Name       string   `json:"name" jsonschema:"display name for the character"`
	Kind       string   `json:"kind" jsonschema:"character kind"`
	Notes      string   `json:"notes" jsonschema:"free-form notes about the character"`
	Pronouns   string   `json:"pronouns" jsonschema:"character pronouns"`
	Aliases    []string `json:"aliases" jsonschema:"character aliases"`
	CreatedAt  string   `json:"created_at" jsonschema:"RFC3339 timestamp when character was created"`
	UpdatedAt  string   `json:"updated_at" jsonschema:"RFC3339 timestamp when character was last updated"`
}

// CharacterControlSetInput represents the MCP tool input for setting character control.
type CharacterControlSetInput struct {
	CampaignID    string `json:"campaign_id" jsonschema:"campaign identifier"`
	CharacterID   string `json:"character_id" jsonschema:"character identifier"`
	ParticipantID string `json:"participant_id" jsonschema:"participant id to control the character (empty to unassign)"`
}

// CharacterControlSetResult represents the MCP tool output for setting character control.
type CharacterControlSetResult struct {
	CampaignID    string `json:"campaign_id" jsonschema:"campaign identifier"`
	CharacterID   string `json:"character_id" jsonschema:"character identifier"`
	ParticipantID string `json:"participant_id" jsonschema:"participant id assigned to the character"`
}

// CharacterSheetGetInput represents the MCP tool input for getting a character sheet.
type CharacterSheetGetInput struct {
	CharacterID string `json:"character_id" jsonschema:"character identifier"`
}

// CharacterSheetGetResult represents the MCP tool output for getting a character sheet.
type CharacterSheetGetResult struct {
	Character CharacterCreateResult  `json:"character" jsonschema:"character metadata"`
	Profile   CharacterProfileResult `json:"profile" jsonschema:"character profile"`
	State     CharacterStateResult   `json:"state" jsonschema:"character state"`
}

// CharacterProfileResult represents character profile data in MCP responses.
type CharacterProfileResult struct {
	CharacterID     string `json:"character_id" jsonschema:"character identifier"`
	HpMax           int    `json:"hp_max" jsonschema:"maximum hit points"`
	StressMax       int    `json:"stress_max" jsonschema:"maximum stress"`
	Evasion         int    `json:"evasion" jsonschema:"evasion difficulty"`
	MajorThreshold  int    `json:"major_threshold" jsonschema:"major damage threshold"`
	SevereThreshold int    `json:"severe_threshold" jsonschema:"severe damage threshold"`
	// Daggerheart traits
	Agility   int `json:"agility" jsonschema:"agility trait (-2 to +4)"`
	Strength  int `json:"strength" jsonschema:"strength trait (-2 to +4)"`
	Finesse   int `json:"finesse" jsonschema:"finesse trait (-2 to +4)"`
	Instinct  int `json:"instinct" jsonschema:"instinct trait (-2 to +4)"`
	Presence  int `json:"presence" jsonschema:"presence trait (-2 to +4)"`
	Knowledge int `json:"knowledge" jsonschema:"knowledge trait (-2 to +4)"`
}

// CharacterStateResult represents character state data in MCP responses.
type CharacterStateResult struct {
	CharacterID string `json:"character_id" jsonschema:"character identifier"`
	Hope        int    `json:"hope" jsonschema:"hope value (0..6)"`
	Stress      int    `json:"stress" jsonschema:"current stress"`
	Hp          int    `json:"hp" jsonschema:"current hit points"`
}

// CharacterProfilePatchInput represents the MCP tool input for patching a character profile.
type CharacterProfilePatchInput struct {
	CharacterID     string `json:"character_id" jsonschema:"character identifier"`
	HpMax           *int   `json:"hp_max,omitempty" jsonschema:"optional hp_max"`
	StressMax       *int   `json:"stress_max,omitempty" jsonschema:"optional stress_max"`
	Evasion         *int   `json:"evasion,omitempty" jsonschema:"optional evasion"`
	MajorThreshold  *int   `json:"major_threshold,omitempty" jsonschema:"optional major_threshold"`
	SevereThreshold *int   `json:"severe_threshold,omitempty" jsonschema:"optional severe_threshold"`
	// Daggerheart traits (optional, -2 to +4)
	Agility   *int `json:"agility,omitempty" jsonschema:"optional agility trait"`
	Strength  *int `json:"strength,omitempty" jsonschema:"optional strength trait"`
	Finesse   *int `json:"finesse,omitempty" jsonschema:"optional finesse trait"`
	Instinct  *int `json:"instinct,omitempty" jsonschema:"optional instinct trait"`
	Presence  *int `json:"presence,omitempty" jsonschema:"optional presence trait"`
	Knowledge *int `json:"knowledge,omitempty" jsonschema:"optional knowledge trait"`
}

// CharacterProfilePatchResult represents the MCP tool output for patching a character profile.
type CharacterProfilePatchResult struct {
	Profile CharacterProfileResult `json:"profile" jsonschema:"updated character profile"`
}

// CharacterStatePatchInput represents the MCP tool input for patching a character state.
type CharacterStatePatchInput struct {
	CharacterID string `json:"character_id" jsonschema:"character identifier"`
	Hope        *int   `json:"hope,omitempty" jsonschema:"optional hope (0..6)"`
	Stress      *int   `json:"stress,omitempty" jsonschema:"optional stress"`
	Hp          *int   `json:"hp,omitempty" jsonschema:"optional hp"`
}

// CharacterStatePatchResult represents the MCP tool output for patching a character state.
type CharacterStatePatchResult struct {
	State CharacterStateResult `json:"state" jsonschema:"updated character state"`
}

// characterProfileResultFromProto converts a proto CharacterProfile to MCP result type.
// Extracts Daggerheart-specific fields from the oneof extension.
