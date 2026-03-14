package invite

import (
	"testing"

	inviteapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/invite/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestMapPublicInviteViewAnonymousInviteIncludesInviterProfileAndLoginRedirect(t *testing.T) {
	t.Parallel()

	view := mapPublicInviteView(inviteapp.InvitePage{
		State: inviteapp.InvitePageStateAnonymous,
		Invite: inviteapp.PublicInvite{
			InviteID:        "inv-1",
			CampaignName:    " Skyfall ",
			ParticipantName: " Scout ",
			InviterUsername: " gm ",
			Status:          inviteapp.InviteStatusPending,
		},
	})

	if view.CampaignName != "Skyfall" {
		t.Fatalf("CampaignName = %q, want %q", view.CampaignName, "Skyfall")
	}
	if view.ParticipantName != "Scout" {
		t.Fatalf("ParticipantName = %q, want %q", view.ParticipantName, "Scout")
	}
	if view.StatusLabel != "Pending" {
		t.Fatalf("StatusLabel = %q, want %q", view.StatusLabel, "Pending")
	}
	if view.InviterUsername != "gm" {
		t.Fatalf("InviterUsername = %q, want %q", view.InviterUsername, "gm")
	}
	if view.InviterProfileURL != routepath.UserProfile("gm") {
		t.Fatalf("InviterProfileURL = %q, want %q", view.InviterProfileURL, routepath.UserProfile("gm"))
	}
	if view.LoginURL != routepath.Login+"?next=%2Finvite%2Finv-1" {
		t.Fatalf("LoginURL = %q, want login redirect", view.LoginURL)
	}
}

func TestMapPublicInviteViewMapsActionableAndTerminalStates(t *testing.T) {
	t.Parallel()

	type assertionsView struct {
		Heading        string
		Body           string
		AcceptLabel    string
		AcceptURL      string
		DeclineLabel   string
		DeclineURL     string
		DashboardLabel string
		DashboardURL   string
	}

	tests := []struct {
		name     string
		state    inviteapp.InvitePageState
		wantBody string
		assert   func(t *testing.T, view assertionsView)
	}{
		{
			name:     "claimable",
			state:    inviteapp.InvitePageStateClaimable,
			wantBody: "This invitation is unassigned. You can claim it with your current account.",
			assert: func(t *testing.T, view assertionsView) {
				t.Helper()
				if view.Heading != "Claim this seat" {
					t.Fatalf("Heading = %q, want %q", view.Heading, "Claim this seat")
				}
				if view.AcceptLabel != "Accept invitation" || view.AcceptURL != routepath.PublicInviteAccept("inv-1") {
					t.Fatalf("accept = (%q, %q), want actionable accept button", view.AcceptLabel, view.AcceptURL)
				}
			},
		},
		{
			name:     "targeted",
			state:    inviteapp.InvitePageStateTargeted,
			wantBody: "This invitation is addressed to you. You can accept or decline it now.",
			assert: func(t *testing.T, view assertionsView) {
				t.Helper()
				if view.Heading != "Invitation ready" {
					t.Fatalf("Heading = %q, want %q", view.Heading, "Invitation ready")
				}
				if view.AcceptURL != routepath.PublicInviteAccept("inv-1") || view.DeclineURL != routepath.PublicInviteDecline("inv-1") {
					t.Fatalf("actions = (%q, %q), want invite actions", view.AcceptURL, view.DeclineURL)
				}
			},
		},
		{
			name:     "mismatch",
			state:    inviteapp.InvitePageStateMismatch,
			wantBody: "This invitation is reserved for a different account.",
			assert: func(t *testing.T, view assertionsView) {
				t.Helper()
				if view.DashboardURL != routepath.AppDashboard {
					t.Fatalf("DashboardURL = %q, want %q", view.DashboardURL, routepath.AppDashboard)
				}
			},
		},
		{
			name:     "claimed",
			state:    inviteapp.InvitePageStateClaimed,
			wantBody: "This seat has already been claimed.",
			assert: func(t *testing.T, view assertionsView) {
				t.Helper()
			},
		},
		{
			name:     "declined",
			state:    inviteapp.InvitePageStateDeclined,
			wantBody: "This invitation has already been declined.",
			assert: func(t *testing.T, view assertionsView) {
				t.Helper()
			},
		},
		{
			name:     "revoked",
			state:    inviteapp.InvitePageStateRevoked,
			wantBody: "This invitation is no longer available.",
			assert: func(t *testing.T, view assertionsView) {
				t.Helper()
			},
		},
		{
			name:     "default",
			state:    inviteapp.InvitePageState("unknown"),
			wantBody: "Review this invitation.",
			assert: func(t *testing.T, view assertionsView) {
				t.Helper()
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mapped := mapPublicInviteView(inviteapp.InvitePage{
				State: tc.state,
				Invite: inviteapp.PublicInvite{
					InviteID: "inv-1",
					Status:   inviteapp.InviteStatusPending,
				},
			})
			view := assertionsView{
				Heading:        mapped.Heading,
				Body:           mapped.Body,
				AcceptLabel:    mapped.AcceptLabel,
				AcceptURL:      mapped.AcceptURL,
				DeclineLabel:   mapped.DeclineLabel,
				DeclineURL:     mapped.DeclineURL,
				DashboardLabel: mapped.DashboardLabel,
				DashboardURL:   mapped.DashboardURL,
			}

			if view.Body != tc.wantBody {
				t.Fatalf("Body = %q, want %q", view.Body, tc.wantBody)
			}
			tc.assert(t, view)
		})
	}
}
