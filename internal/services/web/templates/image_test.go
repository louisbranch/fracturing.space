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
		WidthPX:    16,
		HeightPX:   9,
		FrameClass: "w-full",
		ImageClass: "h-full w-full object-cover",
	}).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render AppImage: %v", err)
	}
	got := buf.String()
	for _, marker := range []string{
		`class="relative overflow-hidden w-full"`,
		`class="skeleton absolute inset-0 z-0 pointer-events-none"`,
		`class="relative z-1 h-full w-full object-cover"`,
		`style="aspect-ratio: 16 / 9;"`,
		`width="16"`,
		`height="9"`,
		`loading="lazy"`,
		`decoding="async"`,
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
		WidthPX:    1,
		HeightPX:   1,
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
		WidthPX:       2,
		HeightPX:      3,
		FrameClass:    "w-10 rounded-full",
		SkeletonClass: "rounded-full",
	}).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render AppImage without src: %v", err)
	}
	got := buf.String()
	for _, marker := range []string{
		`class="skeleton absolute inset-0 z-0 pointer-events-none rounded-full"`,
		`style="aspect-ratio: 2 / 3;"`,
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

func TestAppImageAlwaysPrefixesImageClassWithForegroundLayer(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := AppImage(AppImageView{
		Src:        "https://cdn.example.com/avatars/001.png",
		Alt:        "avatar",
		WidthPX:    1,
		HeightPX:   1,
		FrameClass: "w-10 rounded-full",
		ImageClass: "h-full w-full object-cover rounded-full",
	}).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render AppImage with custom image class: %v", err)
	}
	got := buf.String()
	// Invariant: skeleton must always render below loaded images regardless of custom class input.
	if !strings.Contains(got, `class="relative z-1 h-full w-full object-cover rounded-full"`) {
		t.Fatalf("AppImage output missing foreground image layer contract: %q", got)
	}
}

func TestAppImageOmitsFetchPriorityWhenUnset(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := AppImage(AppImageView{
		Src:      "https://cdn.example.com/avatars/001.png",
		Alt:      "avatar",
		WidthPX:  1,
		HeightPX: 1,
	}).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render AppImage without fetch priority: %v", err)
	}
	got := buf.String()
	if strings.Contains(got, `fetchpriority=`) {
		t.Fatalf("AppImage output unexpectedly emitted fetchpriority without configuration: %q", got)
	}
}

func TestAppImageRendersExplicitFetchPriority(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := AppImage(AppImageView{
		Src:           "https://cdn.example.com/avatars/001.png",
		Alt:           "avatar",
		WidthPX:       1,
		HeightPX:      1,
		Loading:       "eager",
		FetchPriority: "high",
	}).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render AppImage with fetch priority: %v", err)
	}
	got := buf.String()
	for _, marker := range []string{
		`loading="eager"`,
		`fetchpriority="high"`,
	} {
		if !strings.Contains(got, marker) {
			t.Fatalf("AppImage output missing marker %q: %q", marker, got)
		}
	}
}
