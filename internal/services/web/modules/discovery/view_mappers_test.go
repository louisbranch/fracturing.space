package discovery

import (
	"testing"

	discoveryapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery/app"
)

func TestMapEntriesToView(t *testing.T) {
	t.Parallel()

	views := mapEntriesToView([]discoveryapp.StarterEntry{{
		EntryID:     "starter:one",
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
	if views[0].EntryID != "starter:one" {
		t.Fatalf("EntryID = %q, want %q", views[0].EntryID, "starter:one")
	}
	if views[0].Title != "Starter One" {
		t.Fatalf("Title = %q, want %q", views[0].Title, "Starter One")
	}
}

func TestMapPageToView(t *testing.T) {
	t.Parallel()

	view := mapPageToView(discoveryapp.Page{
		Status: discoveryapp.PageStatusUnavailable,
	})
	if !view.Unavailable {
		t.Fatal("Unavailable = false, want true")
	}
	if view.Status != "unavailable" {
		t.Fatalf("Status = %q, want %q", view.Status, "unavailable")
	}
}

func TestMapEntriesToViewReturnsNilForEmptyInput(t *testing.T) {
	t.Parallel()

	if got := mapEntriesToView(nil); got != nil {
		t.Fatalf("mapEntriesToView(nil) = %v, want nil", got)
	}
}
