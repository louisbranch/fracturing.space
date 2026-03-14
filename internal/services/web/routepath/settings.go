package routepath

const (
	AppSettings                       = "/app/settings"
	SettingsPrefix                    = "/app/settings/"
	AppSettingsProfile                = "/app/settings/profile"
	AppSettingsLocale                 = "/app/settings/locale"
	AppSettingsSecurity               = "/app/settings/security"
	AppSettingsSecurityPasskeysStart  = "/app/settings/security/passkeys/start"
	AppSettingsSecurityPasskeysFinish = "/app/settings/security/passkeys/finish"
	AppSettingsAIKeys                 = "/app/settings/ai-keys"
	AppSettingsAIAgents               = "/app/settings/ai-agents"
	AppSettingsAIKeyRevokePattern     = SettingsPrefix + "ai-keys/{credentialID}/revoke"
	AppSettingsRestPattern            = SettingsPrefix + "{rest...}"
)

// AppSettingsAIKeyRevoke returns the AI key revoke route.
func AppSettingsAIKeyRevoke(credentialID string) string {
	return AppSettingsAIKeys + "/" + escapeSegment(credentialID) + "/revoke"
}
