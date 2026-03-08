package participant

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"

// State captures replayed campaign membership and control intent.
//
// Permission checks in multiple services derive from this snapshot to avoid
// scattering identity and role logic across handlers.
type State struct {
	// Joined indicates the participant currently exists in this campaign roster.
	Joined bool
	// Left indicates a completed leave command has been processed.
	Left bool
	// ParticipantID is the campaign-scoped identity used by domain commands.
	ParticipantID ids.ParticipantID
	// UserID links participant records to external authentication identities.
	UserID ids.UserID
	// Name is shown across campaign/session UI and projection outputs.
	Name string
	// Role is the campaign role used for authorization decisions.
	Role Role
	// Controller indicates who can command actions for this participant.
	Controller Controller
	// CampaignAccess controls visibility and permission scope at campaign level.
	CampaignAccess CampaignAccess
	// AvatarSetID identifies the avatar set bound to this participant.
	AvatarSetID string
	// AvatarAssetID identifies the avatar image within AvatarSetID.
	AvatarAssetID string
	// Pronouns stores participant pronouns independent from social profile.
	Pronouns string
}
