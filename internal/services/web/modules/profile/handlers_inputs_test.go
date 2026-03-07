package profile

import (
	"net/http/httptest"
	"testing"
)

func TestParseProfileRouteUsername(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/u/louis", nil)
	req.SetPathValue("username", "  louis  ")

	if got := parseProfileRouteUsername(req); got != "louis" {
		t.Fatalf("parseProfileRouteUsername() = %q, want %q", got, "louis")
	}
}

func TestMapPublicProfileTemplateView(t *testing.T) {
	t.Parallel()

	view := mapPublicProfileTemplateView(Profile{
		Username:  "louis",
		Name:      "Louis",
		Pronouns:  "they/them",
		Bio:       "Builder",
		AvatarURL: "https://cdn.example/avatar.png",
	}, true)

	if view.Username != "louis" {
		t.Fatalf("Username = %q, want %q", view.Username, "louis")
	}
	if view.Name != "Louis" {
		t.Fatalf("Name = %q, want %q", view.Name, "Louis")
	}
	if view.Pronouns != "they/them" {
		t.Fatalf("Pronouns = %q, want %q", view.Pronouns, "they/them")
	}
	if view.Bio != "Builder" {
		t.Fatalf("Bio = %q, want %q", view.Bio, "Builder")
	}
	if view.AvatarURL != "https://cdn.example/avatar.png" {
		t.Fatalf("AvatarURL = %q, want %q", view.AvatarURL, "https://cdn.example/avatar.png")
	}
	if !view.ViewerSignedIn {
		t.Fatalf("ViewerSignedIn = false, want true")
	}
}
