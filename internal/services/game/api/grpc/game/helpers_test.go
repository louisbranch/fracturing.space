package game

import (
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
)

func TestCampaignToProto(t *testing.T) {
	created := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	updated := created.Add(2 * time.Hour)
	completed := created.Add(24 * time.Hour)
	archived := created.Add(48 * time.Hour)

	proto := campaignToProto(campaign.Campaign{
		ID:               "camp-1",
		Name:             "Campaign",
		System:           commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		Status:           campaign.CampaignStatusActive,
		GmMode:           campaign.GmModeHybrid,
		ParticipantCount: 2,
		CharacterCount:   3,
		ThemePrompt:      "storm",
		CreatedAt:        created,
		UpdatedAt:        updated,
		CompletedAt:      &completed,
		ArchivedAt:       &archived,
	})

	if proto.GetId() != "camp-1" || proto.GetName() != "Campaign" {
		t.Fatalf("unexpected campaign proto values: %v", proto)
	}
	if proto.GetStatus() != campaignv1.CampaignStatus_ACTIVE {
		t.Fatalf("expected active status, got %v", proto.GetStatus())
	}
	if proto.GetGmMode() != campaignv1.GmMode_HYBRID {
		t.Fatalf("expected hybrid gm mode, got %v", proto.GetGmMode())
	}
	if proto.GetParticipantCount() != 2 || proto.GetCharacterCount() != 3 {
		t.Fatal("expected participant/character counts to map")
	}
	if proto.GetCreatedAt().AsTime().UTC() != created {
		t.Fatal("expected created timestamp to match")
	}
	if proto.GetUpdatedAt().AsTime().UTC() != updated {
		t.Fatal("expected updated timestamp to match")
	}
	if proto.GetCompletedAt().AsTime().UTC() != completed {
		t.Fatal("expected completed timestamp to match")
	}
	if proto.GetArchivedAt().AsTime().UTC() != archived {
		t.Fatal("expected archived timestamp to match")
	}
}

func TestEnumConversions(t *testing.T) {
	if campaignStatusToProto(campaign.CampaignStatusArchived) != campaignv1.CampaignStatus_ARCHIVED {
		t.Fatal("expected archived campaign status")
	}
	if campaignStatusToProto(campaign.CampaignStatusUnspecified) != campaignv1.CampaignStatus_CAMPAIGN_STATUS_UNSPECIFIED {
		t.Fatal("expected unspecified campaign status")
	}

	if gmModeFromProto(campaignv1.GmMode_AI) != campaign.GmModeAI {
		t.Fatal("expected gm mode AI")
	}
	if gmModeFromProto(campaignv1.GmMode_GM_MODE_UNSPECIFIED) != campaign.GmModeUnspecified {
		t.Fatal("expected gm mode unspecified")
	}

	if participantRoleFromProto(campaignv1.ParticipantRole_GM) != participant.ParticipantRoleGM {
		t.Fatal("expected GM role")
	}
	if participantRoleFromProto(campaignv1.ParticipantRole_ROLE_UNSPECIFIED) != participant.ParticipantRoleUnspecified {
		t.Fatal("expected unspecified role")
	}

	if controllerFromProto(campaignv1.Controller_CONTROLLER_AI) != participant.ControllerAI {
		t.Fatal("expected AI controller")
	}
	if controllerFromProto(campaignv1.Controller_CONTROLLER_UNSPECIFIED) != participant.ControllerUnspecified {
		t.Fatal("expected unspecified controller")
	}

	if campaignAccessFromProto(campaignv1.CampaignAccess_CAMPAIGN_ACCESS_OWNER) != participant.CampaignAccessOwner {
		t.Fatal("expected owner campaign access")
	}
	if campaignAccessFromProto(campaignv1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED) != participant.CampaignAccessUnspecified {
		t.Fatal("expected unspecified campaign access")
	}

	if inviteStatusToProto(invite.StatusPending) != campaignv1.InviteStatus_PENDING {
		t.Fatal("expected pending invite status")
	}
	if inviteStatusFromProto(campaignv1.InviteStatus_INVITE_STATUS_UNSPECIFIED) != invite.StatusUnspecified {
		t.Fatal("expected unspecified invite status")
	}

	if characterKindToProto(character.CharacterKindNPC) != campaignv1.CharacterKind_NPC {
		t.Fatal("expected NPC character kind")
	}
	if characterKindFromProto(campaignv1.CharacterKind_CHARACTER_KIND_UNSPECIFIED) != character.CharacterKindUnspecified {
		t.Fatal("expected unspecified character kind")
	}

	if sessionStatusToProto(session.SessionStatusEnded) != campaignv1.SessionStatus_SESSION_ENDED {
		t.Fatal("expected ended session status")
	}
}

func TestCharacterToProtoParticipantID(t *testing.T) {
	created := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)

	withParticipant := characterToProto(character.Character{
		ID:            "char-1",
		CampaignID:    "camp-1",
		Name:          "Hero",
		Kind:          character.CharacterKindPC,
		ParticipantID: "part-1",
		CreatedAt:     created,
		UpdatedAt:     updated,
	})
	if withParticipant.GetParticipantId().GetValue() != "part-1" {
		t.Fatal("expected participant id wrapper to be set")
	}

	noParticipant := characterToProto(character.Character{
		ID:            "char-2",
		CampaignID:    "camp-1",
		Name:          "NPC",
		Kind:          character.CharacterKindNPC,
		ParticipantID: "  ",
		CreatedAt:     created,
		UpdatedAt:     updated,
	})
	if noParticipant.GetParticipantId() != nil {
		t.Fatal("expected participant id wrapper to be nil")
	}
}

func TestSessionToProtoEndedAt(t *testing.T) {
	started := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	updated := started.Add(time.Hour)
	ended := started.Add(2 * time.Hour)

	withEnd := sessionToProto(session.Session{
		ID:         "sess-1",
		CampaignID: "camp-1",
		Name:       "Session",
		Status:     session.SessionStatusEnded,
		StartedAt:  started,
		UpdatedAt:  updated,
		EndedAt:    &ended,
	})
	if withEnd.GetEndedAt().AsTime().UTC() != ended {
		t.Fatal("expected ended_at to be set")
	}

	noEnd := sessionToProto(session.Session{
		ID:         "sess-2",
		CampaignID: "camp-1",
		Name:       "Active",
		Status:     session.SessionStatusActive,
		StartedAt:  started,
		UpdatedAt:  updated,
	})
	if noEnd.GetEndedAt() != nil {
		t.Fatal("expected ended_at to be nil")
	}
}

func TestTimestampOrNil(t *testing.T) {
	if timestampOrNil(nil) != nil {
		t.Fatal("expected nil timestamp for nil time")
	}
	value := time.Date(2026, 2, 1, 10, 0, 0, 0, time.FixedZone("offset", 3600))
	stamp := timestampOrNil(&value)
	if stamp.AsTime().UTC() != value.UTC() {
		t.Fatal("expected timestamp to be UTC")
	}
}
