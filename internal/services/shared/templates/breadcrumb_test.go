package templates

import (
	"reflect"
	"testing"

	"golang.org/x/text/message"
)

type breadcrumbLocalizer struct{}

func (breadcrumbLocalizer) Sprintf(key message.Reference, _ ...any) string {
	if s, ok := key.(string); ok {
		switch s {
		case "dashboard.title":
			return "Dashboard"
		case "game.campaigns.title":
			return "Campaigns"
		case "game.create.title":
			return "Create Campaign"
		case "game.sessions.title":
			return "Sessions"
		case "game.participants.title":
			return "Participants"
		case "game.characters.title":
			return "Characters"
		case "game.campaign_invites.title":
			return "Campaign Invites"
		case "game.my_invites.title":
			return "My Invites"
		}
		return s
	}
	return ""
}

func TestBuildPathBreadcrumbs(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []BreadcrumbItem
	}{
		{
			name: "campaigns list",
			path: "/campaigns",
			expected: []BreadcrumbItem{
				{Label: "Dashboard", URL: "/"},
				{Label: "Campaigns", URL: "/campaigns"},
			},
		},
		{
			name: "campaign sessions",
			path: "/campaigns/camp-1/sessions",
			expected: []BreadcrumbItem{
				{Label: "Dashboard", URL: "/"},
				{Label: "Campaigns", URL: "/campaigns"},
				{Label: "camp-1", URL: "/campaigns/camp-1"},
				{Label: "Sessions"},
			},
		},
		{
			name: "campaign session detail",
			path: "/campaigns/camp-1/sessions/sess-1",
			expected: []BreadcrumbItem{
				{Label: "Dashboard", URL: "/"},
				{Label: "Campaigns", URL: "/campaigns"},
				{Label: "camp-1", URL: "/campaigns/camp-1"},
				{Label: "Sessions", URL: "/campaigns/camp-1/sessions"},
				{Label: "sess-1"},
			},
		},
		{
			name: "campaign create",
			path: "/campaigns/create",
			expected: []BreadcrumbItem{
				{Label: "Dashboard", URL: "/"},
				{Label: "Campaigns", URL: "/campaigns"},
				{Label: "Create Campaign"},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := BuildPathBreadcrumbs(tc.path, breadcrumbLocalizer{})
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("BuildPathBreadcrumbs(%q) = %#v, expected %#v", tc.path, got, tc.expected)
			}
		})
	}
}

func TestBuildPathBreadcrumbsForWeb(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		hasBreadcrumbs bool
	}{
		{
			name:           "dashboard has no breadcrumbs",
			path:           "/dashboard",
			hasBreadcrumbs: false,
		},
		{
			name:           "campaigns has breadcrumbs",
			path:           "/campaigns",
			hasBreadcrumbs: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := BuildPathBreadcrumbsForWeb(tc.path, breadcrumbLocalizer{})
			if hasBreadcrumbs := len(got) > 0; hasBreadcrumbs != tc.hasBreadcrumbs {
				t.Fatalf("BuildPathBreadcrumbsForWeb(%q) len=%d, expected hasBreadcrumbs=%t", tc.path, len(got), tc.hasBreadcrumbs)
			}
		})
	}
}

func TestBuildPathBreadcrumbsForWebCampaignLabels(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		campaignNames map[string]string
		expected      []BreadcrumbItem
	}{
		{
			name: "campaign sessions uses campaign name",
			path: "/campaigns/camp-1/sessions",
			campaignNames: map[string]string{
				"camp-1": "The Guildhouse",
			},
			expected: []BreadcrumbItem{
				{Label: "Campaigns", URL: "/campaigns"},
				{Label: "The Guildhouse", URL: "/campaigns/camp-1"},
				{Label: "Sessions"},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := BuildPathBreadcrumbsForWeb(tc.path, breadcrumbLocalizer{}, tc.campaignNames)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("BuildPathBreadcrumbsForWeb(%q) = %#v, expected %#v", tc.path, got, tc.expected)
			}
		})
	}
}

func TestBuildPathBreadcrumbsForWebOmitsDashboardRoot(t *testing.T) {
	got := BuildPathBreadcrumbsForWeb("/campaigns/camp-1/sessions", breadcrumbLocalizer{})
	if len(got) == 0 {
		t.Fatalf("BuildPathBreadcrumbsForWeb(%q) returned empty trail", "/campaigns/camp-1/sessions")
	}
	if got[0].Label == "Dashboard" {
		t.Fatalf("BuildPathBreadcrumbsForWeb(%q) should not start with Dashboard, got %#v", "/campaigns/camp-1/sessions", got)
	}
}
