package templates

import (
	"context"
	"regexp"
	"strings"
	"testing"
)

func TestTopNavMarksCurrentSectionAsActive(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		href string
	}{
		{name: "Dashboard", href: "/app/dashboard"},
		{name: "Systems", href: "/app/systems"},
		{name: "Catalog", href: "/app/catalog"},
		{name: "Icons", href: "/app/icons"},
		{name: "Scenarios", href: "/app/scenarios?prefill=1"},
		{name: "Users", href: "/app/users"},
		{name: "Campaigns", href: "/app/campaigns"},
		{name: "Status", href: "/app/status"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := renderTopNavForTest(t, tc.name)
			if count := strings.Count(got, `aria-current="page"`); count != 2 {
				t.Fatalf("aria-current count = %d, want 2", count)
			}

			for _, candidate := range cases {
				re := regexp.MustCompile(regexp.QuoteMeta(`href="`+candidate.href+`"`) + `[^>]*class="active"[^>]*aria-current="page"`)
				count := len(re.FindAllString(got, -1))
				want := 0
				if candidate.name == tc.name {
					want = 2
				}
				if count != want {
					t.Fatalf("active marker count for %q = %d, want %d", candidate.name, count, want)
				}
			}
		})
	}
}

func TestTopNavWithUnknownPageHasNoActiveSection(t *testing.T) {
	t.Parallel()

	got := renderTopNavForTest(t, "Unknown")
	if strings.Contains(got, `aria-current="page"`) {
		t.Fatalf("unexpected active nav marker in output: %q", got)
	}
}

func renderTopNavForTest(t *testing.T, activePage string) string {
	t.Helper()

	var b strings.Builder
	err := TopNav(activePage, PageContext{Lang: "en-US", Loc: langFakeLocalizer{}}, langFakeLocalizer{}).Render(context.Background(), &b)
	if err != nil {
		t.Fatalf("TopNav() render failed: %v", err)
	}
	return b.String()
}
