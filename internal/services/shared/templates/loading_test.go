package templates

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestLoadingRendersRingOnly(t *testing.T) {
	var buf bytes.Buffer
	if err := Loading().Render(context.Background(), &buf); err != nil {
		t.Fatalf("render Loading: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, `class="loading loading-ring loading-md"`) {
		t.Fatalf("Loading output missing loading ring classes: %q", got)
	}
	if strings.Contains(got, "<p>Loading...</p>") {
		t.Fatalf("Loading output should not include message: %q", got)
	}
}
