package campaigntransport

import (
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestCampaignToProto(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	completedAt := now.Add(time.Hour)
	archivedAt := now.Add(2 * time.Hour)

	record := storage.CampaignRecord{
		ID:               "camp-1",
		Name:             "Alpha",
		Locale:           "pt-BR",
		System:           bridge.SystemIDDaggerheart,
		Status:           campaign.StatusActive,
		GmMode:           campaign.GmModeHybrid,
		Intent:           campaign.IntentStarter,
		AccessPolicy:     campaign.AccessPolicyRestricted,
		ParticipantCount: 4,
		CharacterCount:   6,
		ThemePrompt:      "grim",
		CoverAssetID:     "asset-1",
		CoverSetID:       "set-1",
		AIAgentID:        "agent-1",
		CreatedAt:        now,
		UpdatedAt:        now,
		CompletedAt:      &completedAt,
		ArchivedAt:       &archivedAt,
	}

	got := CampaignToProto(record)
	if got.GetId() != record.ID || got.GetName() != record.Name {
		t.Fatalf("campaign proto identity mismatch: %+v", got)
	}
	if got.GetLocale() != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("locale = %v, want %v", got.GetLocale(), commonv1.Locale_LOCALE_PT_BR)
	}
	if got.GetSystem() != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatalf("system = %v", got.GetSystem())
	}
	if got.GetStatus() != campaignv1.CampaignStatus_ACTIVE {
		t.Fatalf("status = %v", got.GetStatus())
	}
	if got.GetGmMode() != campaignv1.GmMode_HYBRID {
		t.Fatalf("gm mode = %v", got.GetGmMode())
	}
	if got.GetIntent() != campaignv1.CampaignIntent_STARTER {
		t.Fatalf("intent = %v", got.GetIntent())
	}
	if got.GetAccessPolicy() != campaignv1.CampaignAccessPolicy_RESTRICTED {
		t.Fatalf("access policy = %v", got.GetAccessPolicy())
	}
	if got.GetParticipantCount() != 4 || got.GetCharacterCount() != 6 {
		t.Fatalf("counts = %d/%d", got.GetParticipantCount(), got.GetCharacterCount())
	}
	if got.GetCreatedAt() == nil || got.GetUpdatedAt() == nil || got.GetCompletedAt() == nil || got.GetArchivedAt() == nil {
		t.Fatalf("expected timestamps to be populated")
	}
}

func TestEnumConversions(t *testing.T) {
	if CampaignStatusToProto(campaign.StatusDraft) != campaignv1.CampaignStatus_DRAFT {
		t.Fatal("draft status mismatch")
	}
	if CampaignStatusToProto(campaign.StatusArchived) != campaignv1.CampaignStatus_ARCHIVED {
		t.Fatal("archived status mismatch")
	}
	if CampaignStatusToProto(campaign.Status("")) != campaignv1.CampaignStatus_CAMPAIGN_STATUS_UNSPECIFIED {
		t.Fatal("unspecified campaign status mismatch")
	}

	if GMModeFromProto(campaignv1.GmMode_AI) != campaign.GmModeAI {
		t.Fatal("gm mode from proto mismatch")
	}
	if GMModeToProto(campaign.GmModeHuman) != campaignv1.GmMode_HUMAN {
		t.Fatal("gm mode to proto mismatch")
	}
	if GMModeFromProto(campaignv1.GmMode_GM_MODE_UNSPECIFIED) != campaign.GmModeUnspecified {
		t.Fatal("unspecified gm mode from proto mismatch")
	}
	if GMModeToProto(campaign.GmMode("")) != campaignv1.GmMode_GM_MODE_UNSPECIFIED {
		t.Fatal("unspecified gm mode to proto mismatch")
	}

	if CampaignIntentFromProto(campaignv1.CampaignIntent_SANDBOX) != campaign.IntentSandbox {
		t.Fatal("intent from proto mismatch")
	}
	if CampaignIntentToProto(campaign.IntentStandard) != campaignv1.CampaignIntent_STANDARD {
		t.Fatal("intent to proto mismatch")
	}
	if CampaignIntentFromProto(campaignv1.CampaignIntent_CAMPAIGN_INTENT_UNSPECIFIED) != campaign.IntentUnspecified {
		t.Fatal("unspecified intent from proto mismatch")
	}
	if CampaignIntentToProto(campaign.Intent("")) != campaignv1.CampaignIntent_CAMPAIGN_INTENT_UNSPECIFIED {
		t.Fatal("unspecified intent to proto mismatch")
	}

	if CampaignAccessPolicyFromProto(campaignv1.CampaignAccessPolicy_PUBLIC) != campaign.AccessPolicyPublic {
		t.Fatal("access policy from proto mismatch")
	}
	if CampaignAccessPolicyToProto(campaign.AccessPolicyPrivate) != campaignv1.CampaignAccessPolicy_PRIVATE {
		t.Fatal("access policy to proto mismatch")
	}
	if CampaignAccessPolicyFromProto(campaignv1.CampaignAccessPolicy_CAMPAIGN_ACCESS_POLICY_UNSPECIFIED) != campaign.AccessPolicyUnspecified {
		t.Fatal("unspecified access policy from proto mismatch")
	}
	if CampaignAccessPolicyToProto(campaign.AccessPolicy("")) != campaignv1.CampaignAccessPolicy_CAMPAIGN_ACCESS_POLICY_UNSPECIFIED {
		t.Fatal("unspecified access policy to proto mismatch")
	}

	if GameSystemFromProto(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART) != bridge.SystemIDDaggerheart {
		t.Fatal("game system from proto mismatch")
	}
	if GameSystemToProto(bridge.SystemIDDaggerheart) != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatal("game system to proto mismatch")
	}
	if GameSystemFromProto(commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED) != "" {
		t.Fatal("unspecified game system from proto mismatch")
	}
	if GameSystemToProto("unknown") != commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
		t.Fatal("unknown game system mismatch")
	}
	if timestampOrNil(nil) != nil {
		t.Fatal("nil timestamp should stay nil")
	}
}
