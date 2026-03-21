package app

import "context"

// Gateway abstracts profile lookup operations behind domain types.
type Gateway interface {
	LookupUserProfile(context.Context, LookupUserProfileRequest) (LookupUserProfileResponse, error)
}

// Service exposes profile orchestration methods used by transport handlers.
type Service interface {
	LoadProfile(context.Context, string) (Profile, error)
}
