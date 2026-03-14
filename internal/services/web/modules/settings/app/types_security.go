package app

import "encoding/json"

// SettingsPasskey stores one passkey summary row rendered on the security page.
type SettingsPasskey struct {
	Number     int
	CreatedAt  string
	LastUsedAt string
}

// PasskeyChallenge stores authenticated passkey enrollment begin state.
type PasskeyChallenge struct {
	SessionID string
	PublicKey json.RawMessage
}
