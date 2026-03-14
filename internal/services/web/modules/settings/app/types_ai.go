package app

import (
	"regexp"
	"strings"
)

// SettingsAIKey stores a credential row displayed in the AI keys page.
type SettingsAIKey struct {
	ID        string
	Label     string
	Provider  string
	Status    string
	CreatedAt string
	RevokedAt string
	CanRevoke bool
}

// SettingsAICredentialOption stores an active credential option for agent creation.
type SettingsAICredentialOption struct {
	ID       string
	Label    string
	Provider string
}

// SettingsAIModelOption stores one provider-backed model option for agent creation.
type SettingsAIModelOption struct {
	ID      string
	OwnedBy string
}

// SettingsAIAgent stores an agent row displayed in the AI agents page.
type SettingsAIAgent struct {
	ID                  string
	Label               string
	Provider            string
	Model               string
	AuthState           string
	CanDelete           bool
	ActiveCampaignCount int32
	CreatedAt           string
	Instructions        string
}

// CreateAIAgentInput stores validated agent creation input.
type CreateAIAgentInput struct {
	Label        string
	CredentialID string
	Model        string
	Instructions string
}

var aiAgentLabelPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{2,31}$`)

// IsSafePathID reports whether a route-bound identifier is safe to reuse in
// settings transport and service flows.
func IsSafePathID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	return !strings.Contains(value, "/") && !strings.Contains(value, "\\")
}
