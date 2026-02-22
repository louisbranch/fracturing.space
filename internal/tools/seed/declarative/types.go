package declarative

import (
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

// Manifest defines a declarative seed graph.
type Manifest struct {
	Name      string             `json:"name"`
	Users     []ManifestUser     `json:"users"`
	Campaigns []ManifestCampaign `json:"campaigns"`
	Forks     []ManifestFork     `json:"forks"`
	Listings  []ManifestListing  `json:"listings"`
}

// ManifestUser defines one auth identity and optional social profile.
type ManifestUser struct {
	Key           string                `json:"key"`
	Email         string                `json:"email"`
	Locale        string                `json:"locale,omitempty"`
	PublicProfile ManifestPublicProfile `json:"public_profile"`
	Contacts      []string              `json:"contacts,omitempty"`
}

// ManifestPublicProfile defines one connections profile record.
type ManifestPublicProfile struct {
	Username      string `json:"username"`
	Name          string `json:"name"`
	AvatarSetID   string `json:"avatar_set_id,omitempty"`
	AvatarAssetID string `json:"avatar_asset_id,omitempty"`
	Bio           string `json:"bio,omitempty"`
}

// ManifestCampaign defines one campaign and optional nested records.
type ManifestCampaign struct {
	Key          string `json:"key"`
	OwnerUserKey string `json:"owner_user_key"`
	Name         string `json:"name"`
	System       string `json:"system,omitempty"`
	GmMode       string `json:"gm_mode,omitempty"`
	Intent       string `json:"intent,omitempty"`
	AccessPolicy string `json:"access_policy,omitempty"`
	ThemePrompt  string `json:"theme_prompt,omitempty"`

	Participants []ManifestParticipant `json:"participants,omitempty"`
	Characters   []ManifestCharacter   `json:"characters,omitempty"`
	Sessions     []ManifestSession     `json:"sessions,omitempty"`

	// Deprecated inline fork fields retained only for backwards compatibility;
	// use Manifest.Forks for new declarations.
	ForkFrom      string           `json:"fork_from,omitempty"`
	ForkEventSeq  uint64           `json:"fork_event_seq,omitempty"`
	ForkSessionID string           `json:"fork_session_id,omitempty"`
	Listing       *ManifestListing `json:"listing,omitempty"`
}

// ManifestParticipant defines one participant seat declaration.
type ManifestParticipant struct {
	Key        string `json:"key"`
	UserKey    string `json:"user_key,omitempty"`
	Name       string `json:"name"`
	Role       string `json:"role,omitempty"`
	Controller string `json:"controller,omitempty"`
}

// ManifestCharacter defines one character declaration.
type ManifestCharacter struct {
	Key                      string `json:"key"`
	Name                     string `json:"name"`
	Kind                     string `json:"kind,omitempty"`
	Notes                    string `json:"notes,omitempty"`
	ControllerParticipantKey string `json:"controller_participant_key,omitempty"`
}

// ManifestSession defines one session declaration.
type ManifestSession struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Status string `json:"status,omitempty"`
}

// ManifestFork defines one fork declaration.
type ManifestFork struct {
	Key               string `json:"key"`
	SourceCampaignKey string `json:"source_campaign_key"`
	OwnerUserKey      string `json:"owner_user_key"`
	NewCampaignName   string `json:"new_campaign_name,omitempty"`
	CopyParticipants  bool   `json:"copy_participants,omitempty"`
	EventSeq          uint64 `json:"event_seq,omitempty"`
	SessionID         string `json:"session_id,omitempty"`
}

// ManifestListing defines one listing declaration.
type ManifestListing struct {
	CampaignKey                string `json:"campaign_key"`
	Title                      string `json:"title"`
	Description                string `json:"description"`
	RecommendedParticipantsMin int32  `json:"recommended_participants_min"`
	RecommendedParticipantsMax int32  `json:"recommended_participants_max"`
	DifficultyTier             string `json:"difficulty_tier,omitempty"`
	ExpectedDurationLabel      string `json:"expected_duration_label"`
	System                     string `json:"system,omitempty"`
}

func defaultSystemLabel(value string) string {
	if value == "" {
		return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String()
	}
	return value
}

func defaultGmModeLabel(value string) string {
	if value == "" {
		return gamev1.GmMode_HUMAN.String()
	}
	return value
}

func defaultCampaignIntentLabel(value string) string {
	if value == "" {
		return gamev1.CampaignIntent_STANDARD.String()
	}
	return value
}

func defaultAccessPolicyLabel(value string) string {
	if value == "" {
		return gamev1.CampaignAccessPolicy_PRIVATE.String()
	}
	return value
}

func defaultParticipantRoleLabel(value string) string {
	if value == "" {
		return gamev1.ParticipantRole_PLAYER.String()
	}
	return value
}

func defaultParticipantControllerLabel(value string) string {
	if value == "" {
		return gamev1.Controller_CONTROLLER_HUMAN.String()
	}
	return value
}

func defaultCharacterKindLabel(value string) string {
	if value == "" {
		return gamev1.CharacterKind_PC.String()
	}
	return value
}

func defaultSessionStatusLabel(value string) string {
	if value == "" {
		return gamev1.SessionStatus_SESSION_ACTIVE.String()
	}
	return value
}
