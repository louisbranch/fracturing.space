package templates

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestLazyLoadUsesSharedLoadingTemplate(t *testing.T) {
	var buf bytes.Buffer
	if err := LazyLoad("/dashboard/content", "Loading dashboard...").Render(context.Background(), &buf); err != nil {
		t.Fatalf("render LazyLoad: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, `hx-get="/dashboard/content"`) {
		t.Fatalf("LazyLoad output missing hx-get URL: %q", got)
	}
	if !strings.Contains(got, `class="loading loading-ring loading-md"`) {
		t.Fatalf("LazyLoad output missing loading ring: %q", got)
	}
	if !strings.Contains(got, `<span class="sr-only">Loading dashboard...</span>`) {
		t.Fatalf("LazyLoad output should include sr-only message: %q", got)
	}
}

func TestLoadingSpinnerUsesSharedLoadingTemplate(t *testing.T) {
	var buf bytes.Buffer
	if err := LoadingSpinner().Render(context.Background(), &buf); err != nil {
		t.Fatalf("render LoadingSpinner: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, `class="loading loading-ring loading-md"`) {
		t.Fatalf("LoadingSpinner output missing loading ring: %q", got)
	}
	if strings.Contains(got, "<p>Loading...</p>") {
		t.Fatalf("LoadingSpinner output should not include message: %q", got)
	}
}
