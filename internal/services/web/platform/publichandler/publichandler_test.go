package publichandler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
)

func TestIsViewerSignedInUsesSignedInResolver(t *testing.T) {
	t.Parallel()

	base := NewBase(
		WithResolveViewer(func(*http.Request) module.Viewer { return module.Viewer{DisplayName: "ignored"} }),
		WithResolveViewerSignedIn(func(*http.Request) bool { return true }),
	)
	if got := base.IsViewerSignedIn(httptest.NewRequest(http.MethodGet, "/", nil)); !got {
		t.Fatalf("IsViewerSignedIn() = %v, want true", got)
	}
}

func TestIsViewerSignedInReturnsFalseWithoutSignedInResolver(t *testing.T) {
	t.Parallel()

	base := NewBase(WithResolveViewer(func(*http.Request) module.Viewer { return module.Viewer{DisplayName: "signed in"} }))
	if got := base.IsViewerSignedIn(httptest.NewRequest(http.MethodGet, "/", nil)); got {
		t.Fatalf("IsViewerSignedIn() = %v, want false", got)
	}
}

func TestIsViewerSignedInNotBackedByDisplayName(t *testing.T) {
	t.Parallel()

	base := NewBase(
		WithResolveViewer(func(*http.Request) module.Viewer { return module.Viewer{DisplayName: ""} }),
		WithResolveViewerSignedIn(func(*http.Request) bool { return true }),
	)
	if !base.IsViewerSignedIn(httptest.NewRequest(http.MethodGet, "/", nil)) {
		t.Fatalf("IsViewerSignedIn() = false, want true")
	}
	if got := base.ResolveRequestViewer(httptest.NewRequest(http.MethodGet, "/", nil)); got.DisplayName != "" {
		t.Fatalf("ResolveRequestViewer() = %+v, want DisplayName=empty", got)
	}
}
