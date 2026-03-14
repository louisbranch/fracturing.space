package participanttransport

import (
	"testing"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
)

func TestParticipantToProto(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	got := ParticipantToProto(storage.ParticipantRecord{
		ID:             "part-1",
		CampaignID:     "camp-1",
		UserID:         "user-1",
		Name:           "Nova",
		Role:           participant.RoleGM,
		CampaignAccess: participant.CampaignAccessOwner,
		Controller:     participant.ControllerHuman,
		AvatarSetID:    "set-1",
		AvatarAssetID:  "asset-1",
		Pronouns:       sharedpronouns.PronounTheyThem,
		CreatedAt:      now,
		UpdatedAt:      now,
	})

	if got.GetId() != "part-1" || got.GetCampaignId() != "camp-1" || got.GetUserId() != "user-1" {
		t.Fatalf("participant identity mismatch: %+v", got)
	}
	if got.GetRole() != campaignv1.ParticipantRole_GM {
		t.Fatalf("role = %v", got.GetRole())
	}
	if got.GetCampaignAccess() != campaignv1.CampaignAccess_CAMPAIGN_ACCESS_OWNER {
		t.Fatalf("campaign access = %v", got.GetCampaignAccess())
	}
	if got.GetController() != campaignv1.Controller_CONTROLLER_HUMAN {
		t.Fatalf("controller = %v", got.GetController())
	}
	if got.GetPronouns() == nil || got.GetCreatedAt() == nil || got.GetUpdatedAt() == nil {
		t.Fatalf("expected pronouns and timestamps")
	}
}

func TestParticipantEnumConversions(t *testing.T) {
	if RoleFromProto(campaignv1.ParticipantRole_PLAYER) != participant.RolePlayer {
		t.Fatal("role from proto mismatch")
	}
	if RoleFromProto(campaignv1.ParticipantRole_ROLE_UNSPECIFIED) != participant.RoleUnspecified {
		t.Fatal("unspecified role from proto mismatch")
	}
	if RoleToProto(participant.RoleGM) != campaignv1.ParticipantRole_GM {
		t.Fatal("role to proto mismatch")
	}
	if RoleToProto(participant.Role("")) != campaignv1.ParticipantRole_ROLE_UNSPECIFIED {
		t.Fatal("unspecified role mismatch")
	}

	if ControllerFromProto(campaignv1.Controller_CONTROLLER_AI) != participant.ControllerAI {
		t.Fatal("controller from proto mismatch")
	}
	if ControllerFromProto(campaignv1.Controller_CONTROLLER_UNSPECIFIED) != participant.ControllerUnspecified {
		t.Fatal("unspecified controller from proto mismatch")
	}
	if ControllerToProto(participant.ControllerHuman) != campaignv1.Controller_CONTROLLER_HUMAN {
		t.Fatal("controller to proto mismatch")
	}
	if ControllerToProto(participant.Controller("")) != campaignv1.Controller_CONTROLLER_UNSPECIFIED {
		t.Fatal("unspecified controller mismatch")
	}

	if CampaignAccessFromProto(campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER) != participant.CampaignAccessManager {
		t.Fatal("campaign access from proto mismatch")
	}
	if CampaignAccessFromProto(campaignv1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED) != participant.CampaignAccessUnspecified {
		t.Fatal("unspecified campaign access from proto mismatch")
	}
	if CampaignAccessToProto(participant.CampaignAccessMember) != campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER {
		t.Fatal("campaign access to proto mismatch")
	}
	if CampaignAccessToProto(participant.CampaignAccess("")) != campaignv1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED {
		t.Fatal("unspecified campaign access mismatch")
	}
}
