package participant

// Role identifies the participant role label.
type Role string

const (
	RoleUnspecified Role = ""
	RoleGM          Role = "gm"
	RolePlayer      Role = "player"
)

// Controller identifies the participant controller label.
type Controller string

const (
	ControllerUnspecified Controller = ""
	ControllerHuman       Controller = "human"
	ControllerAI          Controller = "ai"
)

// CampaignAccess identifies the participant campaign access label.
type CampaignAccess string

const (
	CampaignAccessUnspecified CampaignAccess = ""
	CampaignAccessMember      CampaignAccess = "member"
	CampaignAccessManager     CampaignAccess = "manager"
	CampaignAccessOwner       CampaignAccess = "owner"
)

// NormalizeRole parses a role label into a canonical value.
func NormalizeRole(value string) (Role, bool) {
	if normalized, ok := normalizeRoleLabel(value); ok {
		return Role(normalized), true
	}
	return RoleUnspecified, false
}

// NormalizeController parses a controller label into a canonical value.
func NormalizeController(value string) (Controller, bool) {
	if normalized, ok := normalizeControllerLabel(value); ok {
		return Controller(normalized), true
	}
	return ControllerUnspecified, false
}

// NormalizeCampaignAccess parses an access label into a canonical value.
func NormalizeCampaignAccess(value string) (CampaignAccess, bool) {
	if normalized, ok := normalizeCampaignAccessLabel(value); ok {
		return CampaignAccess(normalized), true
	}
	return CampaignAccessUnspecified, false
}
