package campaign

import "strings"

// GmMode identifies how the GM role is handled.
type GmMode string

const (
	GmModeUnspecified GmMode = ""
	GmModeHuman       GmMode = "human"
	GmModeAI          GmMode = "ai"
	GmModeHybrid      GmMode = "hybrid"
)

// Intent identifies the campaign intent label.
type Intent string

const (
	IntentUnspecified Intent = ""
	IntentStandard    Intent = "standard"
	IntentStarter     Intent = "starter"
	IntentSandbox     Intent = "sandbox"
)

// AccessPolicy identifies campaign discovery policy.
type AccessPolicy string

const (
	AccessPolicyUnspecified AccessPolicy = ""
	AccessPolicyPrivate     AccessPolicy = "private"
	AccessPolicyRestricted  AccessPolicy = "restricted"
	AccessPolicyPublic      AccessPolicy = "public"
)

// NormalizeStatus parses a status label into a canonical value.
func NormalizeStatus(value string) (Status, bool) {
	return normalizeStatusLabel(value)
}

// NormalizeGmMode parses a gm mode label into a canonical value.
func NormalizeGmMode(value string) (GmMode, bool) {
	if normalized, ok := normalizeGmModeLabel(value); ok {
		return GmMode(normalized), true
	}
	return GmModeUnspecified, false
}

// NormalizeIntent parses an intent label into a canonical value.
func NormalizeIntent(value string) Intent {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return IntentStandard
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "STANDARD", "CAMPAIGN_INTENT_STANDARD":
		return IntentStandard
	case "STARTER", "CAMPAIGN_INTENT_STARTER":
		return IntentStarter
	case "SANDBOX", "CAMPAIGN_INTENT_SANDBOX":
		return IntentSandbox
	default:
		return IntentStandard
	}
}

// NormalizeAccessPolicy parses an access policy label into a canonical value.
func NormalizeAccessPolicy(value string) AccessPolicy {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return AccessPolicyPrivate
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "PRIVATE", "CAMPAIGN_ACCESS_POLICY_PRIVATE":
		return AccessPolicyPrivate
	case "RESTRICTED", "CAMPAIGN_ACCESS_POLICY_RESTRICTED":
		return AccessPolicyRestricted
	case "PUBLIC", "CAMPAIGN_ACCESS_POLICY_PUBLIC":
		return AccessPolicyPublic
	default:
		return AccessPolicyPrivate
	}
}
