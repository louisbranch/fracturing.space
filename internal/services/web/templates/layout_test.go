package templates

import (
	"bytes"
	"context"
	"strings"
	"testing"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	"github.com/louisbranch/fracturing.space/internal/services/web/module"
)

func TestAppSideMenuItemSubItemsFiltersInvalidSubItems(t *testing.T) {
	t.Parallel()

	item := AppSideMenuItem{
		SubItems: []AppSideMenuSubItem{
			{Label: "Valid", URL: "/app/campaigns/c1/sessions/s1"},
			{Label: "Missing URL", URL: ""},
			{Label: "", URL: "/app/campaigns/c1/sessions/s2"},
		},
	}

	got := appSideMenuItemSubItems(item)
	if len(got) != 1 {
		t.Fatalf("len(appSideMenuItemSubItems(...)) = %d, want 1", len(got))
	}
	if got[0].Label != "Valid" || got[0].URL != "/app/campaigns/c1/sessions/s1" {
		t.Fatalf("got[0] = %+v, want valid subitem", got[0])
	}
}

func TestAppSideMenuGroupsFiltersInvalidGroupsAndItems(t *testing.T) {
	t.Parallel()

	menu := &AppSideMenu{
		Groups: []AppSideMenuGroup{
			{
				Title: "AI",
				Items: []AppSideMenuItem{
					{Label: "API Keys", URL: "/app/settings/ai-keys"},
					{Label: "", URL: "/app/settings/ai-agents"},
				},
			},
			{
				Title: "",
				Items: []AppSideMenuItem{{Label: "Hidden", URL: "/app/settings/hidden"}},
			},
		},
	}

	got := appSideMenuGroups(menu)
	if len(got) != 1 {
		t.Fatalf("len(appSideMenuGroups(...)) = %d, want 1", len(got))
	}
	if got[0].Title != "AI" {
		t.Fatalf("group title = %q, want %q", got[0].Title, "AI")
	}
	if len(got[0].Items) != 1 || got[0].Items[0].Label != "API Keys" {
		t.Fatalf("group items = %+v, want one valid item", got[0].Items)
	}
}

func TestAppSideMenuSubItemClassHighlightsActiveSessionRows(t *testing.T) {
	t.Parallel()

	menu := &AppSideMenu{CurrentPath: "/app/campaigns/c1/sessions/s1"}
	active := AppSideMenuSubItem{
		Label:         "Session One",
		URL:           "/app/campaigns/c1/sessions/s1",
		ActiveSession: true,
	}
	activeClass := appSideMenuSubItemClass(menu, active)
	for _, want := range []string{"border-success", "bg-base-200", "menu-active"} {
		if !strings.Contains(activeClass, want) {
			t.Fatalf("activeClass = %q, want to contain %q", activeClass, want)
		}
	}

	inactive := AppSideMenuSubItem{
		Label:         "Session Two",
		URL:           "/app/campaigns/c1/sessions/s2",
		ActiveSession: false,
	}
	inactiveClass := appSideMenuSubItemClass(menu, inactive)
	for _, want := range []string{"border-base-300", "bg-base-100"} {
		if !strings.Contains(inactiveClass, want) {
			t.Fatalf("inactiveClass = %q, want to contain %q", inactiveClass, want)
		}
	}
	if strings.Contains(inactiveClass, "menu-active") {
		t.Fatalf("inactiveClass = %q, want no menu-active", inactiveClass)
	}
}

func TestAppSideMenuComponentRendersTitledGroups(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := AppSideMenuComponent(&AppSideMenu{
		CurrentPath: "/app/settings/ai-agents",
		Items: []AppSideMenuItem{
			{Label: "Profile", URL: "/app/settings/profile"},
		},
		Groups: []AppSideMenuGroup{
			{
				Title: "AI",
				Items: []AppSideMenuItem{
					{Label: "API Keys", URL: "/app/settings/ai-keys"},
					{Label: "Agents", URL: "/app/settings/ai-agents"},
				},
			},
		},
	}).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render AppSideMenuComponent: %v", err)
	}

	got := buf.String()
	for _, marker := range []string{
		`data-app-side-menu-group="AI"`,
		`<h2 class="menu-title">AI</h2>`,
		`href="/app/settings/profile"`,
		`href="/app/settings/ai-keys"`,
		`href="/app/settings/ai-agents"`,
		`data-app-side-menu-item="/app/settings/ai-agents"`,
		`class="menu-active"`,
	} {
		if !strings.Contains(got, marker) {
			t.Fatalf("AppSideMenuComponent output missing marker %q: %q", marker, got)
		}
	}
}

func TestAppToastComponentIncludesTopOffsetClass(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := AppToastComponent(&AppToast{
		Kind:    "info",
		Message: "Profile updated",
	}).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render AppToastComponent: %v", err)
	}

	got := buf.String()
	for _, marker := range []string{
		`id="app-toast-stack"`,
		`toast-top`,
		`toast-end`,
		`top-20`,
		`data-app-toast="true"`,
		`data-app-toast-hide-after-ms="4500"`,
	} {
		if !strings.Contains(got, marker) {
			t.Fatalf("AppToastComponent output missing marker %q: %q", marker, got)
		}
	}
}

func TestAppMainContentWithLayoutAddsCoverHeaderClassForBackgroundImages(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := AppMainContentWithLayout(&AppMainHeader{
		Title: "The Guildhouse",
		Breadcrumbs: []sharedtemplates.BreadcrumbItem{
			{Label: "Campaigns", URL: "/app/campaigns"},
			{Label: "The Guildhouse"},
		},
	}, AppMainLayoutOptions{
		MainBackground: &AppBackgroundImage{
			PreviewURL: "/static/campaign-covers/guildhouse-preview.png",
			FullURL:    "/static/campaign-covers/guildhouse.png",
		},
	}).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render AppMainContentWithLayout: %v", err)
	}

	got := buf.String()
	for _, marker := range []string{
		`campaign-cover-header`,
		`<h1 class="mb-0 text-3xl">The Guildhouse</h1>`,
		`data-app-main-background-preview="/static/campaign-covers/guildhouse-preview.png"`,
		`data-app-main-background-full="/static/campaign-covers/guildhouse.png"`,
	} {
		if !strings.Contains(got, marker) {
			t.Fatalf("AppMainContentWithLayout output missing marker %q: %q", marker, got)
		}
	}
}

func TestAppMainContentWithLayoutLeavesHeaderClassOffWithoutBackgroundImages(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := AppMainContentWithLayout(&AppMainHeader{
		Title: "The Guildhouse",
		Breadcrumbs: []sharedtemplates.BreadcrumbItem{
			{Label: "Campaigns", URL: "/app/campaigns"},
			{Label: "The Guildhouse"},
		},
	}, AppMainLayoutOptions{}).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render AppMainContentWithLayout: %v", err)
	}

	got := buf.String()
	for _, marker := range []string{
		`campaign-cover-header`,
	} {
		if strings.Contains(got, marker) {
			t.Fatalf("AppMainContentWithLayout output unexpectedly contains marker %q: %q", marker, got)
		}
	}
	if !strings.Contains(got, `class="mb-0 text-3xl"`) {
		t.Fatalf("AppMainContentWithLayout output missing default h1 class: %q", got)
	}
}

func TestAppLayoutMarksFixedNavbarForSharedScrollOffset(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := AppLayout("Dashboard", module.Viewer{}, "en-US", nil).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render AppLayout: %v", err)
	}

	got := buf.String()
	for _, marker := range []string{
		`data-app-navbar="true"`,
		`src="/static/app-shell.js"`,
		`href="/static/theme.css"`,
	} {
		if !strings.Contains(got, marker) {
			t.Fatalf("AppLayout output missing marker %q: %q", marker, got)
		}
	}
}
