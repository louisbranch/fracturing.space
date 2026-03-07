package profile

import (
	"net/http"
	"strings"
)

// parseProfileRouteUsername normalizes the username route parameter used by the
// profile transport handler.
func parseProfileRouteUsername(r *http.Request) string {
	return strings.TrimSpace(r.PathValue("username"))
}
