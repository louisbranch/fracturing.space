package templates

import (
	"strings"
	"testing"
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
