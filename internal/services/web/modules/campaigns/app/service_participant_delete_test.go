package app

import (
	"context"
	"net/http"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

func TestParticipantDeleteAuthorizationTargetBuildsRemoveScope(t *testing.T) {
	t.Parallel()

	target := participantDeleteAuthorizationTarget(CampaignParticipant{
		ID:             " p-1 ",
		CampaignAccess: "Owner",
	})
	if target == nil {
		t.Fatalf("participantDeleteAuthorizationTarget() = nil")
	}
	if target.ResourceID != "p-1" {
		t.Fatalf("target.ResourceID = %q, want %q", target.ResourceID, "p-1")
	}
	if target.TargetParticipantID != "p-1" {
		t.Fatalf("target.TargetParticipantID = %q, want %q", target.TargetParticipantID, "p-1")
	}
	if target.TargetCampaignAccess != "owner" {
		t.Fatalf("target.TargetCampaignAccess = %q, want %q", target.TargetCampaignAccess, "owner")
	}
	if target.ParticipantOperation != ParticipantGovernanceOperationRemove {
		t.Fatalf("target.ParticipantOperation = %q, want %q", target.ParticipantOperation, ParticipantGovernanceOperationRemove)
	}
}

func TestParticipantDeleteStateFromDecision(t *testing.T) {
	t.Parallel()

	participant := CampaignParticipant{UserID: "user-1"}
	tests := []struct {
		name     string
		decision AuthorizationDecision
		want     CampaignParticipantDeleteState
	}{
		{
			name:     "allowed",
			decision: AuthorizationDecision{Evaluated: true, Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"},
			want: CampaignParticipantDeleteState{
				Visible:           true,
				Enabled:           true,
				HasAssociatedUser: true,
			},
		},
		{
			name:     "ai hidden",
			decision: AuthorizationDecision{Evaluated: true, Allowed: false, ReasonCode: participantDeleteReasonTargetIsAI},
			want: CampaignParticipantDeleteState{
				HasAssociatedUser: true,
			},
		},
		{
			name:     "owned blocker",
			decision: AuthorizationDecision{Evaluated: true, Allowed: false, ReasonCode: participantDeleteReasonTargetOwnsActiveCharacters},
			want: CampaignParticipantDeleteState{
				Visible:                  true,
				Enabled:                  false,
				HasAssociatedUser:        true,
				BlockedByOwnedCharacters: true,
			},
		},
		{
			name:     "controlled blocker",
			decision: AuthorizationDecision{Evaluated: true, Allowed: false, ReasonCode: participantDeleteReasonTargetControlsCharacters},
			want: CampaignParticipantDeleteState{
				Visible:                       true,
				Enabled:                       false,
				HasAssociatedUser:             true,
				BlockedByControlledCharacters: true,
			},
		},
		{
			name:     "unevaluated hidden",
			decision: AuthorizationDecision{},
			want: CampaignParticipantDeleteState{
				HasAssociatedUser: true,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := participantDeleteStateFromDecision(participant, tc.decision)
			if got != tc.want {
				t.Fatalf("participantDeleteStateFromDecision() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestParticipantDeleteErrorMapsReasonCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		reasonCode string
		wantKey    string
		wantStatus int
	}{
		{
			name:       "ai participant",
			reasonCode: participantDeleteReasonTargetIsAI,
			wantKey:    "error.web.message.ai_participants_cannot_be_deleted",
			wantStatus: http.StatusConflict,
		},
		{
			name:       "owned blocker",
			reasonCode: participantDeleteReasonTargetOwnsActiveCharacters,
			wantKey:    "error.web.message.participant_owns_active_characters",
			wantStatus: http.StatusConflict,
		},
		{
			name:       "controlled blocker",
			reasonCode: participantDeleteReasonTargetControlsCharacters,
			wantKey:    "error.web.message.participant_controls_active_characters",
			wantStatus: http.StatusConflict,
		},
		{
			name:       "generic forbidden",
			reasonCode: "AUTHZ_DENY_ACCESS_LEVEL_REQUIRED",
			wantKey:    policyManageParticipant.denyKey,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := participantDeleteError(AuthorizationDecision{ReasonCode: tc.reasonCode})
			if err == nil {
				t.Fatalf("participantDeleteError() = nil")
			}
			if got := apperrors.LocalizationKey(err); got != tc.wantKey {
				t.Fatalf("LocalizationKey(err) = %q, want %q", got, tc.wantKey)
			}
			if got := apperrors.HTTPStatus(err); got != tc.wantStatus {
				t.Fatalf("HTTPStatus(err) = %d, want %d", got, tc.wantStatus)
			}
		})
	}
}

func TestParticipantDeleteDecisionGuardsBlankCampaignAndNilGateway(t *testing.T) {
	t.Parallel()

	participant := CampaignParticipant{ID: "p-1", CampaignAccess: "member"}

	decision, err := participantDeleteDecision(context.Background(), nil, "c-1", participant)
	if err != nil {
		t.Fatalf("participantDeleteDecision(nil auth) error = %v", err)
	}
	if decision != (AuthorizationDecision{}) {
		t.Fatalf("participantDeleteDecision(nil auth) = %#v, want zero value", decision)
	}

	decision, err = participantDeleteDecision(context.Background(), &campaignGatewayStub{}, "   ", participant)
	if err != nil {
		t.Fatalf("participantDeleteDecision(blank campaign) error = %v", err)
	}
	if decision != (AuthorizationDecision{}) {
		t.Fatalf("participantDeleteDecision(blank campaign) = %#v, want zero value", decision)
	}
}
