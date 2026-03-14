package discovery

import (
	"testing"

	discoveryapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery/app"
)

func TestMapEntriesToView(t *testing.T) {
	t.Parallel()

	views := mapEntriesToView([]discoveryapp.StarterEntry{{
		CampaignID:  "c1",
		Title:       "Starter One",
		Description: "A first step",
		Tags:        []string{"beginner"},
		Difficulty:  "Beginner",
		Duration:    "2 sessions",
		GmMode:      "AI",
		System:      "Daggerheart",
		Level:       1,
		Players:     "2-4",
	}})

	if len(views) != 1 {
		t.Fatalf("len(views) = %d, want 1", len(views))
	}
	if views[0].CampaignID != "c1" {
		t.Fatalf("CampaignID = %q, want %q", views[0].CampaignID, "c1")
	}
	if views[0].Title != "Starter One" {
		t.Fatalf("Title = %q, want %q", views[0].Title, "Starter One")
	}
}

func TestMapEntriesToViewReturnsNilForEmptyInput(t *testing.T) {
	t.Parallel()

	if got := mapEntriesToView(nil); got != nil {
		t.Fatalf("mapEntriesToView(nil) = %v, want nil", got)
	}
}
