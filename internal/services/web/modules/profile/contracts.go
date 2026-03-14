package profile

import (
	profileapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/app"
	profilegateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/gateway"
)

// Profile is the transport-facing alias for profile app data.
type Profile = profileapp.Profile

// LookupUserProfileRequest is the transport-facing alias for app profile lookup input.
type LookupUserProfileRequest = profileapp.LookupUserProfileRequest

// LookupUserProfileResponse is the transport-facing alias for app profile lookup output.
type LookupUserProfileResponse = profileapp.LookupUserProfileResponse

// ProfileGateway is the transport-facing alias for profile app gateway contract.
type ProfileGateway = profileapp.Gateway

// AuthClient aliases keep module constructors and tests on narrow contracts.
type AuthClient = profilegateway.AuthClient

// SocialClient aliases keep module constructors and tests on narrow contracts.
type SocialClient = profilegateway.SocialClient

const profileNotFoundMessage = profileapp.ProfileNotFoundMessage
