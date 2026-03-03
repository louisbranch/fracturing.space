package templates

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestAppImageRendersSkeletonAndImage(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := AppImage(AppImageView{
		Src:        "https://cdn.example.com/covers/shipyard.png",
		Alt:        "Shipyard cover",
		FrameClass: "aspect-[16/9] w-full",
		FrameStyle: "aspect-ratio: 16 / 9;",
		ImageClass: "h-full w-full object-cover",
	}).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render AppImage: %v", err)
	}
	got := buf.String()
	for _, marker := range []string{
		`data-image-frame="true"`,
		`data-image-skeleton="true"`,
		`data-image-el="true"`,
		`class="relative overflow-hidden aspect-[16/9] w-full"`,
		`style="aspect-ratio: 16 / 9;"`,
		`src="https://cdn.example.com/covers/shipyard.png"`,
		`alt="Shipyard cover"`,
	} {
		if !strings.Contains(got, marker) {
			t.Fatalf("AppImage output missing marker %q: %q", marker, got)
		}
	}
}

func TestAppImageOmitsEmptyStyleAttribute(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := AppImage(AppImageView{
		Src:        "https://cdn.example.com/avatars/001.png",
		Alt:        "avatar",
		FrameClass: "w-10 rounded-full",
	}).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render AppImage without frame style: %v", err)
	}
	got := buf.String()
	// Invariant: style attribute should only be emitted when explicitly configured.
	if strings.Contains(got, `style=""`) {
		t.Fatalf("AppImage output unexpectedly emitted empty style attribute: %q", got)
	}
}

func TestAppImageRendersSkeletonWhenSourceMissing(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := AppImage(AppImageView{
		Src:           "   ",
		Alt:           "unused alt",
		FrameClass:    "w-10 rounded-full",
		SkeletonClass: "rounded-full",
	}).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render AppImage without src: %v", err)
	}
	got := buf.String()
	for _, marker := range []string{
		`data-image-frame="true"`,
		`data-image-skeleton="true"`,
		`class="skeleton absolute inset-0 rounded-full"`,
	} {
		if !strings.Contains(got, marker) {
			t.Fatalf("AppImage output missing marker %q: %q", marker, got)
		}
	}
	// Invariant: missing source keeps skeleton-only rendering and avoids a broken image request.
	if strings.Contains(got, "<img") {
		t.Fatalf("AppImage output unexpectedly rendered img element without source: %q", got)
	}
}
