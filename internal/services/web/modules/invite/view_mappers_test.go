package invite

import (
	"fmt"
	"testing"

	inviteapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/invite/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"golang.org/x/text/message"
)

type testLocalizer map[string]string

func (l testLocalizer) Sprintf(key message.Reference, args ...any) string {
	keyString := fmt.Sprint(key)
	format, ok := l[keyString]
	if !ok {
		format = keyString
	}
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}

func englishInviteLocalizer() testLocalizer {
	return testLocalizer{
		"web.invite.action.accept":           "Accept invitation",
		"web.invite.action.dashboard":        "Back to dashboard",
		"web.invite.action.decline":          "Decline invitation",
		"web.invite.action.login":            "Sign in or create account",
		"web.invite.state.anonymous.body":    "Sign in or create an account to view and respond to this invitation.",
		"web.invite.state.anonymous.heading": "Campaign invitation",
		"web.invite.state.claimable.body":    "This invitation is unassigned. You can claim it with your current account.",
		"web.invite.state.claimable.heading": "Claim this seat",
		"web.invite.state.claimed.body":      "This seat has already been claimed.",
		"web.invite.state.claimed.heading":   "Invitation claimed",
		"web.invite.state.declined.body":     "This invitation has already been declined.",
		"web.invite.state.declined.heading":  "Invitation declined",
		"web.invite.state.default.body":      "Review this invitation.",
		"web.invite.state.default.heading":   "Campaign invitation",
		"web.invite.state.mismatch.body":     "This invitation is reserved for a different account.",
		"web.invite.state.mismatch.heading":  "Invitation reserved",
		"web.invite.state.revoked.body":      "This invitation is no longer available.",
		"web.invite.state.revoked.heading":   "Invitation revoked",
		"web.invite.state.targeted.body":     "This invitation is addressed to you. You can accept or decline it now.",
		"web.invite.state.targeted.heading":  "Invitation ready",
		"web.invite.status.claimed":          "Claimed",
		"web.invite.status.declined":         "Declined",
		"web.invite.status.pending":          "Pending",
		"web.invite.status.revoked":          "Revoked",
		"web.invite.status.unspecified":      "Unspecified",
	}
}

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
	}, englishInviteLocalizer())

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
			}, englishInviteLocalizer())
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

func TestMapPublicInviteViewLocalizesStatusAndActions(t *testing.T) {
	t.Parallel()

	view := mapPublicInviteView(inviteapp.InvitePage{
		State: inviteapp.InvitePageStateTargeted,
		Invite: inviteapp.PublicInvite{
			InviteID: "inv-1",
			Status:   inviteapp.InviteStatusClaimed,
		},
	}, testLocalizer{
		"web.invite.action.accept":          "Aceitar convite",
		"web.invite.action.decline":         "Recusar convite",
		"web.invite.state.targeted.heading": "Convite pronto",
		"web.invite.state.targeted.body":    "Este convite foi enderecado a voce.",
		"web.invite.status.claimed":         "Reivindicado",
	})

	if view.Heading != "Convite pronto" {
		t.Fatalf("Heading = %q, want %q", view.Heading, "Convite pronto")
	}
	if view.AcceptLabel != "Aceitar convite" {
		t.Fatalf("AcceptLabel = %q, want %q", view.AcceptLabel, "Aceitar convite")
	}
	if view.DeclineLabel != "Recusar convite" {
		t.Fatalf("DeclineLabel = %q, want %q", view.DeclineLabel, "Recusar convite")
	}
	if view.StatusLabel != "Reivindicado" {
		t.Fatalf("StatusLabel = %q, want %q", view.StatusLabel, "Reivindicado")
	}
}
