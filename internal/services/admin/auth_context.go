package admin

import "context"

// authUserKey is the context key for the authenticated user ID.
type authUserKey struct{}

// contextWithAuthUser returns a context carrying the authenticated user ID.
func contextWithAuthUser(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, authUserKey{}, userID)
}

// authUserFromContext extracts the authenticated user ID from the context.
// Returns an empty string if absent or the context is nil.
func authUserFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(authUserKey{}).(string)
	return v
}
