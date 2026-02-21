package templates

import (
	"context"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/platform/branding"
)

func TestComposePageTitleAddsBrandNameSuffix(t *testing.T) {
	got := ComposePageTitle("Campaigns")
	want := "Campaigns | " + branding.AppName
	if got != want {
		t.Fatalf("composePageTitle = %q, want %q", got, want)
	}
}

func TestComposePageTitleSkipsWhenAlreadyUsingPipeBrandSuffix(t *testing.T) {
	got := ComposePageTitle("Campaigns | " + branding.AppName)
	want := "Campaigns | " + branding.AppName
	if got != want {
		t.Fatalf("composePageTitle = %q, want %q", got, want)
	}
}

func TestComposePageTitleNormalizesHyphenBrandSuffix(t *testing.T) {
	got := ComposePageTitle("Campaigns - " + branding.AppName)
	want := "Campaigns | " + branding.AppName
	if got != want {
		t.Fatalf("composePageTitle = %q, want %q", got, want)
	}
}

func TestComposePageTitleSupportsAdminSuffix(t *testing.T) {
	got := ComposePageTitle("Campaigns - Admin")
	want := "Campaigns - Admin | " + branding.AppName
	if got != want {
		t.Fatalf("composePageTitle = %q, want %q", got, want)
	}
}

func TestAppChromeLayoutSupportsCustomBreadcrumbs(t *testing.T) {
	breadcrumbs := []BreadcrumbItem{
		{Label: "Dashboard", URL: "/"},
		{Label: "Custom"},
	}
	var b strings.Builder
	err := AppChromeLayout("Campaigns", "en-US", branding.AppName, breadcrumbLocalizer{}, breadcrumbs).Render(context.Background(), &b)
	if err != nil {
		t.Fatalf("AppChromeLayout() = %v", err)
	}
	got := b.String()
	if !strings.Contains(got, `href="/">Dashboard</a>`) {
		t.Fatalf("expected custom breadcrumb root in chrome layout, got %q", got)
	}
	if !strings.Contains(got, `<li>Custom</li>`) {
		t.Fatalf("expected custom breadcrumb tail in chrome layout, got %q", got)
	}
}
