package oauth

import "time"

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
