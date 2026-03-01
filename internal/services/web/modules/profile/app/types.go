package app

import "context"

// ProfileNotFoundMessage is the canonical safe message for a missing public profile.
const ProfileNotFoundMessage = "public profile not found"

// Profile stores one public profile page payload.
type Profile struct {
	Username  string
	Name      string
	Pronouns  string
	Bio       string
	AvatarURL string
}

// LookupUserProfileRequest represents a domain request to load one user profile.
type LookupUserProfileRequest struct {
	Username string
}

// LookupUserProfileResponse stores the minimal profile fields the web module needs.
type LookupUserProfileResponse struct {
	Username      string
	UserID        string
	Name          string
	Pronouns      string
	Bio           string
	AvatarSetID   string
	AvatarAssetID string
}

// Gateway abstracts profile lookup operations behind domain types.
type Gateway interface {
	LookupUserProfile(context.Context, LookupUserProfileRequest) (LookupUserProfileResponse, error)
}

// Service exposes profile orchestration methods used by transport handlers.
type Service interface {
	LoadProfile(context.Context, string) (Profile, error)
}
