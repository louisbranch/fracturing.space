package app

import "context"

// ProfileNotFoundMessage is the canonical safe message for a missing public profile.
const ProfileNotFoundMessage = "public profile not found"

// SocialProfileStatus describes how much social-owned profile data was available.
type SocialProfileStatus string

const (
	// SocialProfileStatusUnspecified indicates the gateway did not report a social detail state.
	SocialProfileStatusUnspecified SocialProfileStatus = ""

	// SocialProfileStatusLoaded indicates social-owned profile data was loaded successfully.
	SocialProfileStatusLoaded SocialProfileStatus = "loaded"

	// SocialProfileStatusMissing indicates the social service was queried but no social profile exists.
	SocialProfileStatusMissing SocialProfileStatus = "missing"

	// SocialProfileStatusUnavailable indicates the social service could not be queried successfully.
	SocialProfileStatusUnavailable SocialProfileStatus = "unavailable"

	// SocialProfileStatusUnconfigured indicates the profile module was composed without a social client.
	SocialProfileStatusUnconfigured SocialProfileStatus = "unconfigured"
)

// Profile stores one public profile page payload.
type Profile struct {
	Username            string
	UserID              string
	Name                string
	Pronouns            string
	Bio                 string
	AvatarSetID         string
	AvatarAssetID       string
	SocialProfileStatus SocialProfileStatus
}

// LookupUserProfileRequest represents a domain request to load one user profile.
type LookupUserProfileRequest struct {
	Username string
}

// LookupUserProfileResponse stores the minimal profile fields the web module needs.
type LookupUserProfileResponse struct {
	Username            string
	UserID              string
	Name                string
	Pronouns            string
	Bio                 string
	AvatarSetID         string
	AvatarAssetID       string
	SocialProfileStatus SocialProfileStatus
}

// Gateway abstracts profile lookup operations behind domain types.
type Gateway interface {
	LookupUserProfile(context.Context, LookupUserProfileRequest) (LookupUserProfileResponse, error)
}

// Service exposes profile orchestration methods used by transport handlers.
type Service interface {
	LoadProfile(context.Context, string) (Profile, error)
}
