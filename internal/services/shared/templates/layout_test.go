package templates

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
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
	if !strings.Contains(got, `href="/app/settings/profile"`) {
		t.Fatalf("expected profile link in avatar dropdown, got %q", got)
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

func TestAppChromeLayoutRendersNotificationButtonLeftOfAvatar(t *testing.T) {
	var b strings.Builder
	err := AppChromeLayout(AppChromeLayoutOptions{
		Title:   "Campaigns",
		Lang:    "en-US",
		AppName: branding.AppName,
		Loc:     breadcrumbLocalizer{},
		ChromeOptions: ChromeLayoutOptions{
			UserName:               "Alice",
			UserAvatarURL:          "https://example.com/avatar.png",
			HasUnreadNotifications: false,
		},
	}).Render(context.Background(), &b)
	if err != nil {
		t.Fatalf("AppChromeLayout() = %v", err)
	}
	got := b.String()
	if !strings.Contains(got, `href="/app/notifications"`) {
		t.Fatalf("expected notifications button link, got %q", got)
	}
	if !strings.Contains(got, `href="#lucide-bell"`) {
		t.Fatalf("expected bell icon for read notification state, got %q", got)
	}
	if strings.Contains(got, `href="#lucide-bell-dot"`) {
		t.Fatalf("expected read notification state to avoid bell-dot icon, got %q", got)
	}
	notificationIndex := strings.Index(got, `href="/app/notifications"`)
	avatarIndex := strings.Index(got, `class="btn btn-ghost btn-circle avatar"`)
	if notificationIndex < 0 || avatarIndex < 0 {
		t.Fatalf("missing notification or avatar controls in output")
	}
	if notificationIndex > avatarIndex {
		t.Fatalf("expected notifications control before avatar control")
	}

	snippetEnd := notificationIndex + 180
	if snippetEnd > len(got) {
		snippetEnd = len(got)
	}
	snippet := got[notificationIndex:snippetEnd]
	if !strings.Contains(snippet, `data-nav-item="true"`) {
		t.Fatalf("expected notifications control to participate in nav active state, got %q", snippet)
	}
}

func TestAppChromeLayoutRendersUnreadNotificationBellDot(t *testing.T) {
	var b strings.Builder
	err := AppChromeLayout(AppChromeLayoutOptions{
		Title:   "Campaigns",
		Lang:    "en-US",
		AppName: branding.AppName,
		Loc:     breadcrumbLocalizer{},
		ChromeOptions: ChromeLayoutOptions{
			UserName:               "Alice",
			UserAvatarURL:          "https://example.com/avatar.png",
			HasUnreadNotifications: true,
		},
	}).Render(context.Background(), &b)
	if err != nil {
		t.Fatalf("AppChromeLayout() = %v", err)
	}
	got := b.String()
	if !strings.Contains(got, `href="#lucide-bell-dot"`) {
		t.Fatalf("expected bell-dot icon when unread notifications exist, got %q", got)
	}
}

func TestAppChromeLayoutDashboardNavTargetsDashboardRoot(t *testing.T) {
	var b strings.Builder
	err := AppChromeLayout(AppChromeLayoutOptions{
		Title:   "Campaigns",
		Lang:    "en-US",
		AppName: branding.AppName,
		Loc:     breadcrumbLocalizer{},
	}).Render(context.Background(), &b)
	if err != nil {
		t.Fatalf("AppChromeLayout() = %v", err)
	}

	got := b.String()
	if !strings.Contains(got, `href="/" hx-get="/"`) {
		t.Fatalf("expected dashboard nav to target root via href and hx-get, got %q", got)
	}
	if !strings.Contains(got, `>Dashboard</a>`) {
		t.Fatalf("expected dashboard nav label, got %q", got)
	}
}

func TestChromeMainClassFromStyleDefaultDoesNotCenter(t *testing.T) {
	got := chromeMainClassFromStyle("", "")
	if strings.Contains(got, "mx-auto") {
		t.Fatalf("expected default chrome main class to omit mx-auto, got %q", got)
	}
}

func TestAppChromeLayoutRendersHeadingActionOnSameLine(t *testing.T) {
	var b strings.Builder
	err := AppChromeLayout(AppChromeLayoutOptions{
		Title:   "Campaigns",
		Lang:    "en-US",
		AppName: branding.AppName,
		Loc:     breadcrumbLocalizer{},
		HeadingAction: templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
			_, err := io.WriteString(w, `<a id="heading-action-test" class="btn btn-primary btn-sm" href="/app/campaigns/create">Start</a>`)
			return err
		}),
	}).Render(context.Background(), &b)
	if err != nil {
		t.Fatalf("AppChromeLayout() = %v", err)
	}
	got := b.String()
	if !strings.Contains(got, `class="mb-5 flex items-center justify-between gap-3"`) {
		t.Fatalf("expected heading row flex container, got %q", got)
	}
	if !strings.Contains(got, `<h1 class="mb-0">Campaigns</h1>`) {
		t.Fatalf("expected heading text in same-row h1, got %q", got)
	}
	if !strings.Contains(got, `id="heading-action-test"`) {
		t.Fatalf("expected custom heading action component in output, got %q", got)
	}
}
