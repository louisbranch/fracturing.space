package modulehandler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
)

func TestNewBaseExtractsResolvers(t *testing.T) {
	t.Parallel()

	resolveUserID := func(*http.Request) string { return "user-1" }
	resolveLanguage := func(*http.Request) string { return "en" }
	resolveViewer := func(*http.Request) module.Viewer { return module.Viewer{DisplayName: "Test"} }

	base := NewBase(resolveUserID, resolveLanguage, resolveViewer)
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	if got := base.RequestUserID(r); got != "user-1" {
		t.Fatalf("RequestUserID() = %q, want %q", got, "user-1")
	}
	if got := base.ResolveRequestLanguage(r); got != "en" {
		t.Fatalf("ResolveRequestLanguage() = %q, want %q", got, "en")
	}
	if got := base.ResolveRequestViewer(r); got.DisplayName != "Test" {
		t.Fatalf("ResolveRequestViewer() = %+v, want DisplayName=Test", got)
	}
}

func TestResolveRequestViewerDelegatesToResolver(t *testing.T) {
	t.Parallel()

	want := module.Viewer{DisplayName: "Test"}
	base := NewBase(nil, nil, func(*http.Request) module.Viewer { return want })

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := base.ResolveRequestViewer(r); got != want {
		t.Fatalf("ResolveRequestViewer() = %+v, want %+v", got, want)
	}
}

func TestResolveRequestViewerReturnsZeroWhenNil(t *testing.T) {
	t.Parallel()

	base := NewBase(nil, nil, nil)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := base.ResolveRequestViewer(r); got != (module.Viewer{}) {
		t.Fatalf("ResolveRequestViewer() = %+v, want zero Viewer", got)
	}
}

func TestResolveRequestLanguageDelegatesToResolver(t *testing.T) {
	t.Parallel()

	base := NewBase(nil, func(*http.Request) string { return "en" }, nil)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := base.ResolveRequestLanguage(r); got != "en" {
		t.Fatalf("ResolveRequestLanguage() = %q, want %q", got, "en")
	}
}

func TestResolveRequestLanguageReturnsEmptyWhenNil(t *testing.T) {
	t.Parallel()

	base := NewBase(nil, nil, nil)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := base.ResolveRequestLanguage(r); got != "" {
		t.Fatalf("ResolveRequestLanguage() = %q, want empty", got)
	}
}

func TestRequestUserIDReturnsEmpty(t *testing.T) {
	t.Parallel()

	t.Run("nil request", func(t *testing.T) {
		base := NewBase(func(*http.Request) string { return "user-1" }, nil, nil)
		if got := base.RequestUserID(nil); got != "" {
			t.Fatalf("expected empty, got %q", got)
		}
	})

	t.Run("nil resolver", func(t *testing.T) {
		base := NewBase(nil, nil, nil)
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		if got := base.RequestUserID(r); got != "" {
			t.Fatalf("expected empty, got %q", got)
		}
	})
}

func TestRequestUserIDTrimsWhitespace(t *testing.T) {
	t.Parallel()

	base := NewBase(func(*http.Request) string { return "  user-1  " }, nil, nil)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := base.RequestUserID(r); got != "user-1" {
		t.Fatalf("expected %q, got %q", "user-1", got)
	}
}

func TestRequestContextAndUserIDReturnsContext(t *testing.T) {
	t.Parallel()

	base := NewBase(func(*http.Request) string { return "user-1" }, nil, nil)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx, userID := base.RequestContextAndUserID(r)
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	if userID != "user-1" {
		t.Fatalf("expected %q, got %q", "user-1", userID)
	}
}
