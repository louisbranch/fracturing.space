package app

import (
	"context"
	"errors"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

func TestLoadInviteStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		viewerUserID string
		invite       PublicInvite
		wantState    InvitePageState
		wantAccept   bool
		wantDecline  bool
	}{
		{
			name:         "anonymous targeted invite",
			viewerUserID: "",
			invite:       PublicInvite{InviteID: "inv-1", RecipientUserID: "user-2", Status: InviteStatusPending},
			wantState:    InvitePageStateAnonymous,
		},
		{
			name:         "claimable unassigned invite",
			viewerUserID: "user-1",
			invite:       PublicInvite{InviteID: "inv-1", Status: InviteStatusPending},
			wantState:    InvitePageStateClaimable,
			wantAccept:   true,
		},
		{
			name:         "targeted invite for viewer",
			viewerUserID: "user-1",
			invite:       PublicInvite{InviteID: "inv-1", RecipientUserID: "user-1", Status: InviteStatusPending},
			wantState:    InvitePageStateTargeted,
			wantAccept:   true,
			wantDecline:  true,
		},
		{
			name:         "mismatch invite",
			viewerUserID: "user-1",
			invite:       PublicInvite{InviteID: "inv-1", RecipientUserID: "user-2", Status: InviteStatusPending},
			wantState:    InvitePageStateMismatch,
		},
		{
			name:         "claimed invite",
			viewerUserID: "user-1",
			invite:       PublicInvite{InviteID: "inv-1", Status: InviteStatusClaimed},
			wantState:    InvitePageStateClaimed,
		},
		{
			name:         "declined invite",
			viewerUserID: "user-1",
			invite:       PublicInvite{InviteID: "inv-1", Status: InviteStatusDeclined},
			wantState:    InvitePageStateDeclined,
		},
		{
			name:         "revoked invite",
			viewerUserID: "user-1",
			invite:       PublicInvite{InviteID: "inv-1", Status: InviteStatusRevoked},
			wantState:    InvitePageStateRevoked,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := NewService(&inviteGatewayStub{invite: tc.invite})
			page, err := svc.LoadInvite(context.Background(), tc.viewerUserID, "inv-1")
			if err != nil {
				t.Fatalf("LoadInvite() error = %v", err)
			}
			if page.State != tc.wantState {
				t.Fatalf("page.State = %q, want %q", page.State, tc.wantState)
			}
			if page.CanAccept != tc.wantAccept {
				t.Fatalf("page.CanAccept = %v, want %v", page.CanAccept, tc.wantAccept)
			}
			if page.CanDecline != tc.wantDecline {
				t.Fatalf("page.CanDecline = %v, want %v", page.CanDecline, tc.wantDecline)
			}
		})
	}
}

func TestLoadInviteRejectsBlankInviteID(t *testing.T) {
	t.Parallel()

	_, err := NewService(&inviteGatewayStub{}).LoadInvite(context.Background(), "user-1", "   ")
	if got := apperrors.HTTPStatus(err); got != 404 {
		t.Fatalf("HTTPStatus(err) = %d, want 404", got)
	}
}

func TestLoadInvitePropagatesGatewayError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("boom")
	_, err := NewService(&inviteGatewayStub{getErr: wantErr}).LoadInvite(context.Background(), "user-1", "inv-1")
	if !errors.Is(err, wantErr) {
		t.Fatalf("LoadInvite() error = %v, want %v", err, wantErr)
	}
}

func TestAcceptInviteClaimsInviteAndReturnsAffectedUsers(t *testing.T) {
	t.Parallel()

	gateway := inviteGatewayStub{
		invite: PublicInvite{
			InviteID:        "inv-1",
			CampaignID:      "camp-1",
			ParticipantID:   "part-1",
			RecipientUserID: "user-1",
			CreatedByUserID: "creator-1",
			Status:          InviteStatusPending,
		},
	}

	result, err := NewService(&gateway).AcceptInvite(context.Background(), " user-1 ", "inv-1")
	if err != nil {
		t.Fatalf("AcceptInvite() error = %v", err)
	}
	if gateway.acceptViewer != "user-1" {
		t.Fatalf("accept viewer = %q, want %q", gateway.acceptViewer, "user-1")
	}
	if gateway.acceptInvite.InviteID != "inv-1" {
		t.Fatalf("accept invite = %+v, want invite id inv-1", gateway.acceptInvite)
	}
	if result.CampaignID != "camp-1" {
		t.Fatalf("CampaignID = %q, want %q", result.CampaignID, "camp-1")
	}
	if len(result.UserIDs) != 2 || result.UserIDs[0] != "user-1" || result.UserIDs[1] != "creator-1" {
		t.Fatalf("UserIDs = %v, want [user-1 creator-1]", result.UserIDs)
	}
}

func TestAcceptInviteRejectsForbiddenState(t *testing.T) {
	t.Parallel()

	_, err := NewService(&inviteGatewayStub{
		invite: PublicInvite{InviteID: "inv-1", RecipientUserID: "user-2", Status: InviteStatusPending},
	}).AcceptInvite(context.Background(), "user-1", "inv-1")
	if got := apperrors.HTTPStatus(err); got != 403 {
		t.Fatalf("HTTPStatus(err) = %d, want 403", got)
	}
}

func TestDeclineInviteDeclinesTargetedInvite(t *testing.T) {
	t.Parallel()

	gateway := inviteGatewayStub{
		invite: PublicInvite{
			InviteID:        "inv-1",
			CampaignID:      "camp-1",
			RecipientUserID: "user-1",
			CreatedByUserID: "creator-1",
			Status:          InviteStatusPending,
		},
	}

	result, err := NewService(&gateway).DeclineInvite(context.Background(), "user-1", "inv-1")
	if err != nil {
		t.Fatalf("DeclineInvite() error = %v", err)
	}
	if gateway.declineViewer != "user-1" || gateway.declineInviteID != "inv-1" {
		t.Fatalf("decline call = (%q, %q), want (%q, %q)", gateway.declineViewer, gateway.declineInviteID, "user-1", "inv-1")
	}
	if len(result.UserIDs) != 2 || result.UserIDs[0] != "user-1" || result.UserIDs[1] != "creator-1" {
		t.Fatalf("UserIDs = %v, want [user-1 creator-1]", result.UserIDs)
	}
}

func TestDeclineInviteRejectsForbiddenState(t *testing.T) {
	t.Parallel()

	_, err := NewService(&inviteGatewayStub{
		invite: PublicInvite{InviteID: "inv-1", Status: InviteStatusPending},
	}).DeclineInvite(context.Background(), "user-1", "inv-1")
	if got := apperrors.HTTPStatus(err); got != 403 {
		t.Fatalf("HTTPStatus(err) = %d, want 403", got)
	}
}

func TestNewServiceUsesUnavailableGatewayWhenNil(t *testing.T) {
	t.Parallel()

	_, err := NewService(nil).LoadInvite(context.Background(), "user-1", "inv-1")
	if got := apperrors.HTTPStatus(err); got != 503 {
		t.Fatalf("HTTPStatus(err) = %d, want 503", got)
	}
}

type inviteGatewayStub struct {
	invite          PublicInvite
	getErr          error
	acceptErr       error
	declineErr      error
	acceptViewer    string
	acceptInvite    PublicInvite
	declineViewer   string
	declineInviteID string
}

func (s inviteGatewayStub) GetPublicInvite(context.Context, string) (PublicInvite, error) {
	if s.getErr != nil {
		return PublicInvite{}, s.getErr
	}
	return s.invite, nil
}

func (s *inviteGatewayStub) AcceptInvite(_ context.Context, viewerUserID string, invite PublicInvite) error {
	s.acceptViewer = viewerUserID
	s.acceptInvite = invite
	return s.acceptErr
}

func (s *inviteGatewayStub) DeclineInvite(_ context.Context, viewerUserID string, inviteID string) error {
	s.declineViewer = viewerUserID
	s.declineInviteID = inviteID
	return s.declineErr
}
