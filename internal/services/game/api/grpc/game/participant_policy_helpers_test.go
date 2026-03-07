package game

import (
	"context"
	"errors"
	"strings"
	"testing"

	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestCountCampaignOwnersNilStore(t *testing.T) {
	_, err := countCampaignOwners(context.Background(), nil, "c1")
	assertStatusCode(t, err, codes.Internal)
	if err == nil || !strings.Contains(err.Error(), "participant store is not configured") {
		t.Fatalf("error = %v, want participant store configuration message", err)
	}
}

func TestCountCampaignOwnersReturnsOwnerCount(t *testing.T) {
	participantStore := newFakeParticipantStore()
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {ID: "p1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
		"p2": {ID: "p2", CampaignID: "c1", CampaignAccess: participant.CampaignAccessManager},
		"p3": {ID: "p3", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
	}

	ownerCount, err := countCampaignOwners(context.Background(), participantStore, "c1")
	if err != nil {
		t.Fatalf("countCampaignOwners returned error: %v", err)
	}
	if ownerCount != 2 {
		t.Fatalf("owner count = %d, want 2", ownerCount)
	}
}

func TestCountCampaignOwnersListErrorReturnsInternal(t *testing.T) {
	participantStore := newFakeParticipantStore()
	participantStore.listErr = errors.New("boom")

	_, err := countCampaignOwners(context.Background(), participantStore, "c1")
	assertStatusCode(t, err, codes.Internal)
	if err == nil || !strings.Contains(err.Error(), "list participants") {
		t.Fatalf("error = %v, want list participants context", err)
	}
}

func TestParticipantPolicyDecisionErrorReasonCodeMapping(t *testing.T) {
	tests := []struct {
		name          string
		reasonCode    string
		wantCode      codes.Code
		wantMsgSubset string
	}{
		{
			name:          "manager cannot assign owner",
			reasonCode:    domainauthz.ReasonDenyManagerOwnerMutationForbidden,
			wantCode:      codes.PermissionDenied,
			wantMsgSubset: "manager cannot assign owner access",
		},
		{
			name:          "active character ownership guard",
			reasonCode:    domainauthz.ReasonDenyTargetOwnsActiveCharacters,
			wantCode:      codes.FailedPrecondition,
			wantMsgSubset: "transfer ownership first",
		},
		{
			name:          "unknown fallback",
			reasonCode:    "unknown_reason",
			wantCode:      codes.PermissionDenied,
			wantMsgSubset: "participant lacks permission",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := participantPolicyDecisionError(tc.reasonCode)
			assertStatusCode(t, err, tc.wantCode)
			if err == nil || !strings.Contains(err.Error(), tc.wantMsgSubset) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantMsgSubset)
			}
		})
	}
}
