package oauth

import "time"

// OAuthUser represents a local credentialed OAuth user.
type OAuthUser struct {
	UserID       string
	Username     string
	PasswordHash string
	DisplayName  string
}

// AuthorizationRequest captures inbound /authorize parameters.
type AuthorizationRequest struct {
	ResponseType        string
	ClientID            string
	RedirectURI         string
	Scope               string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
}

// AuthorizationCode represents a stored authorization code.
type AuthorizationCode struct {
	Code                string
	ClientID            string
	UserID              string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	Scope               string
	State               string
	ExpiresAt           time.Time
	Used                bool
}

// AccessToken represents a bearer access token.
type AccessToken struct {
	Token     string
	ClientID  string
	UserID    string
	Scope     string
	ExpiresAt time.Time
}

// PendingAuthorization tracks an in-progress authorization flow.
type PendingAuthorization struct {
	ID        string
	Request   AuthorizationRequest
	UserID    string
	ExpiresAt time.Time
}

// ProviderState tracks an external OAuth login flow.
type ProviderState struct {
	State        string
	Provider     string
	RedirectURI  string
	CodeVerifier string
	ExpiresAt    time.Time
}

// ExternalIdentity represents a linked external provider identity.
type ExternalIdentity struct {
	ID             string
	Provider       string
	ProviderUserID string
	UserID         string
	AccessToken    string
	RefreshToken   string
	Scope          string
	ExpiresAt      time.Time
	IDToken        string
	UpdatedAt      time.Time
}
