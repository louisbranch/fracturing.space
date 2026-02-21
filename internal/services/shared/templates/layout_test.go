package templates

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/platform/branding"
)

func TestComposePageTitleAddsBrandNameSuffix(t *testing.T) {
	got := ComposePageTitle("Campaigns")
	want := "Campaigns | " + branding.AppName
	if got != want {
		t.Fatalf("composePageTitle = %q, want %q", got, want)
	}
}

func TestComposePageTitleSkipsWhenAlreadyUsingPipeBrandSuffix(t *testing.T) {
	got := ComposePageTitle("Campaigns | " + branding.AppName)
	want := "Campaigns | " + branding.AppName
	if got != want {
		t.Fatalf("composePageTitle = %q, want %q", got, want)
	}
}

func TestComposePageTitleNormalizesHyphenBrandSuffix(t *testing.T) {
	got := ComposePageTitle("Campaigns - " + branding.AppName)
	want := "Campaigns | " + branding.AppName
	if got != want {
		t.Fatalf("composePageTitle = %q, want %q", got, want)
	}
}
