package requestresolver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
)

func TestResolveRequestViewerDelegatesToConfiguredResolver(t *testing.T) {
	t.Parallel()

	base := New(nil, func(*http.Request) module.Viewer {
		return module.Viewer{DisplayName: "Ada"}
	})

	if got := base.ResolveRequestViewer(httptest.NewRequest(http.MethodGet, "/", nil)); got.DisplayName != "Ada" {
		t.Fatalf("ResolveRequestViewer() = %+v, want DisplayName=Ada", got)
	}
}

func TestResolveRequestViewerReturnsZeroWhenUnset(t *testing.T) {
	t.Parallel()

	base := New(nil, nil)

	if got := base.ResolveRequestViewer(httptest.NewRequest(http.MethodGet, "/", nil)); got != (module.Viewer{}) {
		t.Fatalf("ResolveRequestViewer() = %+v, want zero Viewer", got)
	}
}

func TestResolveRequestLanguageDelegatesToConfiguredResolver(t *testing.T) {
	t.Parallel()

	base := New(func(*http.Request) string { return "pt-BR" }, nil)

	if got := base.ResolveRequestLanguage(httptest.NewRequest(http.MethodGet, "/", nil)); got != "pt-BR" {
		t.Fatalf("ResolveRequestLanguage() = %q, want %q", got, "pt-BR")
	}
}

func TestResolveRequestLanguageReturnsEmptyWhenUnset(t *testing.T) {
	t.Parallel()

	base := New(nil, nil)

	if got := base.ResolveRequestLanguage(httptest.NewRequest(http.MethodGet, "/", nil)); got != "" {
		t.Fatalf("ResolveRequestLanguage() = %q, want empty", got)
	}
}

func TestWithViewerAndLanguageReturnUpdatedCopies(t *testing.T) {
	t.Parallel()

	base := New(nil, nil)
	updated := base.WithLanguage(func(*http.Request) string { return "en" }).WithViewer(func(*http.Request) module.Viewer {
		return module.Viewer{DisplayName: "Test"}
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := base.ResolveRequestLanguage(req); got != "" {
		t.Fatalf("base ResolveRequestLanguage() = %q, want empty", got)
	}
	if got := updated.ResolveRequestLanguage(req); got != "en" {
		t.Fatalf("updated ResolveRequestLanguage() = %q, want %q", got, "en")
	}
	if got := updated.ResolveRequestViewer(req).DisplayName; got != "Test" {
		t.Fatalf("updated ResolveRequestViewer().DisplayName = %q, want %q", got, "Test")
	}
}

func TestNewFromPageResolverCopiesConfiguredResolvers(t *testing.T) {
	t.Parallel()

	base := NewFromPageResolver(New(
		func(*http.Request) string { return "en" },
		func(*http.Request) module.Viewer { return module.Viewer{DisplayName: "Test"} },
	))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := base.ResolveRequestLanguage(req); got != "en" {
		t.Fatalf("ResolveRequestLanguage() = %q, want %q", got, "en")
	}
	if got := base.ResolveRequestViewer(req).DisplayName; got != "Test" {
		t.Fatalf("ResolveRequestViewer().DisplayName = %q, want %q", got, "Test")
	}
}

func TestNewFromPageResolverReturnsZeroBaseWhenNil(t *testing.T) {
	t.Parallel()

	base := NewFromPageResolver(nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := base.ResolveRequestLanguage(req); got != "" {
		t.Fatalf("ResolveRequestLanguage() = %q, want empty", got)
	}
	if got := base.ResolveRequestViewer(req); got != (module.Viewer{}) {
		t.Fatalf("ResolveRequestViewer() = %+v, want zero Viewer", got)
	}
}

func TestPrincipalDelegatesToConfiguredResolvers(t *testing.T) {
	t.Parallel()

	principal := NewPrincipal(
		func(*http.Request) bool { return true },
		func(*http.Request) bool { return true },
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "en" },
		func(*http.Request) module.Viewer { return module.Viewer{DisplayName: "Ada"} },
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if !principal.AuthRequired(req) {
		t.Fatalf("AuthRequired() = false, want true")
	}
	if !principal.ResolveSignedIn(req) {
		t.Fatalf("ResolveSignedIn() = false, want true")
	}
	if got := principal.ResolveUserID(req); got != "user-1" {
		t.Fatalf("ResolveUserID() = %q, want %q", got, "user-1")
	}
	if got := principal.ResolveRequestLanguage(req); got != "en" {
		t.Fatalf("ResolveRequestLanguage() = %q, want %q", got, "en")
	}
	if got := principal.ResolveRequestViewer(req).DisplayName; got != "Ada" {
		t.Fatalf("ResolveRequestViewer().DisplayName = %q, want %q", got, "Ada")
	}
}

func TestPrincipalReturnsZeroValuesWhenUnset(t *testing.T) {
	t.Parallel()

	principal := NewPrincipal(nil, nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if principal.AuthRequired(req) {
		t.Fatalf("AuthRequired() = true, want false")
	}
	if principal.ResolveSignedIn(req) {
		t.Fatalf("ResolveSignedIn() = true, want false")
	}
	if got := principal.ResolveUserID(req); got != "" {
		t.Fatalf("ResolveUserID() = %q, want empty", got)
	}
}

func TestResolveLocalizedPageUsesResolverLanguage(t *testing.T) {
	t.Parallel()

	page := ResolveLocalizedPage(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/", nil),
		New(func(*http.Request) string { return "pt-BR" }, nil),
	)

	if page.Localizer == nil {
		t.Fatalf("ResolveLocalizedPage().Localizer = nil, want non-nil")
	}
	if page.Language != "pt-BR" {
		t.Fatalf("ResolveLocalizedPage().Language = %q, want %q", page.Language, "pt-BR")
	}
}

func TestResolveViewerReturnsZeroWhenUnset(t *testing.T) {
	t.Parallel()

	if got := ResolveViewer(httptest.NewRequest(http.MethodGet, "/", nil), nil); got != (module.Viewer{}) {
		t.Fatalf("ResolveViewer() = %+v, want zero Viewer", got)
	}
}

func TestResolveViewerDelegatesToResolver(t *testing.T) {
	t.Parallel()

	viewer := ResolveViewer(
		httptest.NewRequest(http.MethodGet, "/", nil),
		New(nil, func(*http.Request) module.Viewer {
			return module.Viewer{DisplayName: "Ada"}
		}),
	)

	if viewer.DisplayName != "Ada" {
		t.Fatalf("ResolveViewer().DisplayName = %q, want %q", viewer.DisplayName, "Ada")
	}
}
