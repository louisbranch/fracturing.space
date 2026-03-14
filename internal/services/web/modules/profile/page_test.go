package profile

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestPublicProfileRendersPortraitAvatarSlot(t *testing.T) {
	t.Parallel()

	view := PublicProfileView{
		Username:  "louis",
		Name:      "Louis Branch",
		AvatarURL: "https://cdn.example.com/avatars/louis.png",
	}

	var buf bytes.Buffer
	if err := PublicProfilePage(view, nil).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render PublicProfilePage: %v", err)
	}

	got := buf.String()
	for _, marker := range []string{
		`style="aspect-ratio: 2 / 3;"`,
		`width="2"`,
		`height="3"`,
		`loading="eager"`,
		`src="https://cdn.example.com/avatars/louis.png"`,
	} {
		if !strings.Contains(got, marker) {
			t.Fatalf("PublicProfile output missing marker %q: %q", marker, got)
		}
	}
	if strings.Contains(got, `aspect-ratio: 1 / 1;`) {
		t.Fatalf("PublicProfile output unexpectedly kept square avatar framing: %q", got)
	}
}
