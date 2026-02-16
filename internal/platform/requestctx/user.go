package requestctx

import "context"

// userIDContextKey is the context key for authenticated user identity.
type userIDContextKey struct{}

// WithUserID stores a user identifier in context.
func WithUserID(ctx context.Context, userID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, userIDContextKey{}, userID)
}

// UserIDFromContext returns the user identifier stored in context.
func UserIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(userIDContextKey{}).(string)
	return value
}
