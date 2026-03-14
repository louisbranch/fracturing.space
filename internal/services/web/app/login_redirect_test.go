package app

import (
	"net/http/httptest"
	"testing"
)

func TestLoginRedirectPathFallbacks(t *testing.T) {
	t.Parallel()

	if got := loginRedirectPath(nil); got != defaultLoginPath {
		t.Fatalf("loginRedirectPath(nil) = %q, want %q", got, defaultLoginPath)
	}

	req := httptest.NewRequest("GET", "/app/dashboard/", nil)
	req.URL = nil
	if got := loginRedirectPath(req); got != defaultLoginPath {
		t.Fatalf("loginRedirectPath(req with nil URL) = %q, want %q", got, defaultLoginPath)
	}
}

func TestLoginRedirectPathEncodesRequestURI(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/invite/inv-1?from=mail", nil)
	if got := loginRedirectPath(req); got != "/login?next=%2Finvite%2Finv-1%3Ffrom%3Dmail" {
		t.Fatalf("loginRedirectPath() = %q, want %q", got, "/login?next=%2Finvite%2Finv-1%3Ffrom%3Dmail")
	}
}
