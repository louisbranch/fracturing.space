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

func TestPageHeadingFromTitleStripsBrandAndAdminSuffix(t *testing.T) {
	got := pageHeadingFromTitle("Campaigns - Admin | "+branding.AppName, branding.AppName)
	if got != "Campaigns" {
		t.Fatalf("pageHeadingFromTitle = %q, want %q", got, "Campaigns")
	}
}

func TestPageHeadingFromTitleUsesBaseTitleWhenAlreadyRaw(t *testing.T) {
	got := pageHeadingFromTitle("Campaigns", branding.AppName)
	if got != "Campaigns" {
		t.Fatalf("pageHeadingFromTitle = %q, want %q", got, "Campaigns")
	}
}

func TestAppChromeLayoutSupportsCustomBreadcrumbs(t *testing.T) {
	breadcrumbs := []BreadcrumbItem{
		{Label: "Dashboard", URL: "/"},
		{Label: "Custom"},
	}
	var b strings.Builder
	err := AppChromeLayout(AppChromeLayoutOptions{
		Title:         "Campaigns",
		Lang:          "en-US",
		AppName:       branding.AppName,
		Loc:           breadcrumbLocalizer{},
		Breadcrumbs:   breadcrumbs,
		ChromeOptions: ChromeLayoutOptions{},
	}).Render(context.Background(), &b)
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

func TestAppChromeLayoutRendersAvatarDropdownWhenAvatarURLProvided(t *testing.T) {
	breadcrumbs := []BreadcrumbItem{
		{Label: "Dashboard", URL: "/"},
	}
	var b strings.Builder
	err := AppChromeLayout(AppChromeLayoutOptions{
		Title:       "Campaigns",
		Lang:        "en-US",
		AppName:     branding.AppName,
		Loc:         breadcrumbLocalizer{},
		Breadcrumbs: breadcrumbs,
		ChromeOptions: ChromeLayoutOptions{
			UserName:      "Alice",
			UserAvatarURL: "https://example.com/avatar.png",
		},
	}).Render(context.Background(), &b)
	if err != nil {
		t.Fatalf("AppChromeLayout() = %v", err)
	}
	got := b.String()
	if !strings.Contains(got, `class="dropdown dropdown-end"`) {
		t.Fatalf("expected avatar dropdown wrapper, got %q", got)
	}
	if !strings.Contains(got, `src="https://example.com/avatar.png"`) {
		t.Fatalf("expected avatar URL in dropdown, got %q", got)
	}
	if !strings.Contains(got, `alt="Alice"`) {
		t.Fatalf("expected user name alt text in avatar, got %q", got)
	}
}

func TestAppChromeLayoutFallsBackToSignOutButtonWithoutAvatar(t *testing.T) {
	breadcrumbs := []BreadcrumbItem{
		{Label: "Dashboard", URL: "/"},
	}
	var b strings.Builder
	err := AppChromeLayout(AppChromeLayoutOptions{
		Title:       "Campaigns",
		Lang:        "en-US",
		AppName:     branding.AppName,
		Loc:         breadcrumbLocalizer{},
		Breadcrumbs: breadcrumbs,
		ChromeOptions: ChromeLayoutOptions{
			UserName:      "Alice",
			UserAvatarURL: "",
		},
	}).Render(context.Background(), &b)
	if err != nil {
		t.Fatalf("AppChromeLayout() = %v", err)
	}
	got := b.String()
	if strings.Contains(got, `class="dropdown dropdown-end"`) {
		t.Fatalf("expected no dropdown wrapper when avatar is missing, got %q", got)
	}
	if !strings.Contains(got, `form method="POST" action="/auth/logout"`) {
		t.Fatalf("expected sign out fallback form when avatar is missing, got %q", got)
	}
}
