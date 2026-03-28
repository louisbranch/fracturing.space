package agent

import "strings"

// AuthReferenceKind identifies which auth source one agent uses.
type AuthReferenceKind string

const (
	// AuthReferenceKindCredential points at one stored BYO credential.
	AuthReferenceKindCredential AuthReferenceKind = "credential"
	// AuthReferenceKindProviderGrant points at one stored OAuth provider grant.
	AuthReferenceKindProviderGrant AuthReferenceKind = "provider_grant"
)

// AuthReference keeps agent auth selection typed so callers do not reimplement
// kind dispatch and exclusivity rules.
type AuthReference struct {
	Kind AuthReferenceKind
	ID   string
}

// CredentialAuthReference constructs one credential-backed auth reference.
func CredentialAuthReference(credentialID string) AuthReference {
	return AuthReference{Kind: AuthReferenceKindCredential, ID: credentialID}
}

// ProviderGrantAuthReference constructs one provider-grant-backed auth reference.
func ProviderGrantAuthReference(providerGrantID string) AuthReference {
	return AuthReference{Kind: AuthReferenceKindProviderGrant, ID: providerGrantID}
}

// NormalizeAuthReference trims and validates one typed auth reference.
func NormalizeAuthReference(reference AuthReference, require bool) (AuthReference, error) {
	reference.Kind = AuthReferenceKind(strings.TrimSpace(string(reference.Kind)))
	reference.ID = strings.TrimSpace(reference.ID)

	if reference.Kind == "" && reference.ID == "" {
		if require {
			return AuthReference{}, ErrMissingAuthReference
		}
		return AuthReference{}, nil
	}
	if reference.Kind == "" || reference.ID == "" {
		return AuthReference{}, ErrMissingAuthReference
	}
	switch reference.Kind {
	case AuthReferenceKindCredential, AuthReferenceKindProviderGrant:
		return reference, nil
	default:
		return AuthReference{}, ErrInvalidAuthReference
	}
}

// CredentialID returns the credential ID when this reference points at one.
func (r AuthReference) CredentialID() string {
	if r.Kind != AuthReferenceKindCredential {
		return ""
	}
	return r.ID
}

// ProviderGrantID returns the provider-grant ID when this reference points at one.
func (r AuthReference) ProviderGrantID() string {
	if r.Kind != AuthReferenceKindProviderGrant {
		return ""
	}
	return r.ID
}

// Type reports the stable auth-reference kind string for transport uses.
func (r AuthReference) Type() string {
	switch r.Kind {
	case AuthReferenceKindCredential:
		return string(AuthReferenceKindCredential)
	case AuthReferenceKindProviderGrant:
		return string(AuthReferenceKindProviderGrant)
	default:
		return ""
	}
}

// IsZero reports whether the auth reference is unset.
func (r AuthReference) IsZero() bool {
	return r.ID == "" && r.Kind == ""
}
