package ai

import (
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// agentAuthReferenceFromRecord reconstructs the domain-owned auth selection
// from the current persisted split-ID shape.
func agentAuthReferenceFromRecord(record storage.AgentRecord) (agent.AuthReference, error) {
	return agent.AuthReferenceFromIDs(record.CredentialID, record.ProviderGrantID, true)
}

// applyAgentAuthReference projects the typed domain auth reference back onto the
// current storage shape until persistence is made typed as well.
func applyAgentAuthReference(record *storage.AgentRecord, reference agent.AuthReference) {
	record.CredentialID = reference.CredentialID()
	record.ProviderGrantID = reference.ProviderGrantID()
}

// credentialFromRecord rebuilds one domain credential for lifecycle and
// usability checks.
func credentialFromRecord(record storage.CredentialRecord) credential.Credential {
	return credential.Credential{
		ID:          record.ID,
		OwnerUserID: record.OwnerUserID,
		Provider:    providerFromString(record.Provider),
		Label:       record.Label,
		Status:      credential.ParseStatus(record.Status),
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
		RevokedAt:   record.RevokedAt,
	}
}

// applyCredentialLifecycle writes domain-owned lifecycle fields back into the
// persisted record without reopening storage ownership of ciphertext handling.
func applyCredentialLifecycle(record *storage.CredentialRecord, value credential.Credential) {
	record.Status = string(value.Status)
	record.UpdatedAt = value.UpdatedAt
	record.RevokedAt = value.RevokedAt
}

// providerGrantFromRecord rebuilds one domain provider grant for lifecycle and
// usability checks.
func providerGrantFromRecord(record storage.ProviderGrantRecord) providergrant.ProviderGrant {
	return providergrant.ProviderGrant{
		ID:               record.ID,
		OwnerUserID:      record.OwnerUserID,
		Provider:         providerFromString(record.Provider),
		GrantedScopes:    record.GrantedScopes,
		TokenCiphertext:  record.TokenCiphertext,
		RefreshSupported: record.RefreshSupported,
		Status:           providergrant.ParseStatus(record.Status),
		LastRefreshError: record.LastRefreshError,
		CreatedAt:        record.CreatedAt,
		UpdatedAt:        record.UpdatedAt,
		RevokedAt:        record.RevokedAt,
		ExpiresAt:        record.ExpiresAt,
		RefreshedAt:      record.LastRefreshedAt,
	}
}

// applyProviderGrantLifecycle writes domain-owned lifecycle changes back into
// the persisted provider-grant record.
func applyProviderGrantLifecycle(record *storage.ProviderGrantRecord, value providergrant.ProviderGrant) {
	record.TokenCiphertext = value.TokenCiphertext
	record.Status = string(value.Status)
	record.LastRefreshError = value.LastRefreshError
	record.UpdatedAt = value.UpdatedAt
	record.RevokedAt = value.RevokedAt
	record.ExpiresAt = value.ExpiresAt
	record.LastRefreshedAt = value.RefreshedAt
}
