package redirectpath

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestResolveSafeRejectsUnsafeTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "external url", input: "https://evil.example/app/campaigns", want: ""},
		{name: "app root", input: routepath.AppPrefix, want: ""},
		{name: "invite root", input: strings.TrimRight(routepath.InvitePrefix, "/"), want: ""},
		{name: "encoded slash", input: "/app/campaigns/%2fadmin", want: ""},
		{name: "dot segment", input: "/app/../settings", want: ""},
		{name: "valid app path", input: "/app/campaigns/camp-1?tab=people", want: "/app/campaigns/camp-1?tab=people"},
		{name: "valid invite path", input: "/invite/inv-1", want: "/invite/inv-1"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := ResolveSafe(tc.input); got != tc.want {
				t.Fatalf("ResolveSafe(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
