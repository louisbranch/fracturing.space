package sharedpath

import (
	"reflect"
	"testing"
)

func TestSplitPathParts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "empty path",
			path: "",
			want: []string{},
		},
		{
			name: "single segment",
			path: "campaign-1",
			want: []string{"campaign-1"},
		},
		{
			name: "multiple segments",
			path: "campaign-1/sessions/session-1",
			want: []string{"campaign-1", "sessions", "session-1"},
		},
		{
			name: "ignores repeated slashes and surrounding spaces",
			path: " /campaign-1//sessions/ session-1 / ",
			want: []string{"campaign-1", "sessions", "session-1"},
		},
		{
			name: "trailing slash",
			path: "campaign-1/",
			want: []string{"campaign-1"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := SplitPathParts(tc.path)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("SplitPathParts(%q) = %#v, want %#v", tc.path, got, tc.want)
			}
		})
	}
}
