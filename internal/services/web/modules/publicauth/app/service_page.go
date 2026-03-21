package app

// pageService owns page-only public auth behavior.
type pageService struct {
	authBaseURL string
}

// NewPageService wires page-only public auth flows behind input validation.
func NewPageService(authBaseURL string) PageService {
	return pageService{authBaseURL: normalizeAuthBaseURL(authBaseURL)}
}

// HealthBody returns the plain-text health response expected by the endpoint.
func (pageService) HealthBody() string {
	return "ok"
}

// ResolvePostAuthRedirect returns the auth consent URL, validated continuation,
// or dashboard.
func (s pageService) ResolvePostAuthRedirect(pendingID string, nextPath string) string {
	return resolvePostAuthRedirect(s.authBaseURL, pendingID, nextPath)
}
