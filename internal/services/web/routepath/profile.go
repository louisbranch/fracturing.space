package routepath

const (
	UserProfilePrefix                   = "/u/"
	UserProfilePattern                  = UserProfilePrefix + "{username}"
	UserProfilePatternWithTrailingSlash = UserProfilePrefix + "{username}/"
	UserProfileRestPattern              = UserProfilePrefix + "{username}/{rest...}"
)

// UserProfile returns the public user profile route.
func UserProfile(username string) string {
	return UserProfilePrefix + escapeSegment(username)
}
