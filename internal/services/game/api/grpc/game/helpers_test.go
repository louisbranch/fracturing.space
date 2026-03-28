package game

import (
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/campaigntransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/charactertransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/participanttransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestCampaignToProto(t *testing.T) {
	created := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	updated := created.Add(2 * time.Hour)
	completed := created.Add(24 * time.Hour)
	archived := created.Add(48 * time.Hour)

	proto := campaigntransport.CampaignToProto(storage.CampaignRecord{
		ID:               "camp-1",
		Name:             "Campaign",
		Locale:           "pt-BR",
		System:           bridge.SystemIDDaggerheart,
		Status:           campaign.StatusActive,
		GmMode:           campaign.GmModeHybrid,
		Intent:           campaign.IntentStarter,
		AccessPolicy:     campaign.AccessPolicyRestricted,
		ParticipantCount: 2,
		CharacterCount:   3,
		ThemePrompt:      "storm",
		CoverAssetID:     "camp-cover-03",
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
	if proto.GetIntent() != campaignv1.CampaignIntent_STARTER {
		t.Fatalf("expected starter intent, got %v", proto.GetIntent())
	}
	if proto.GetAccessPolicy() != campaignv1.CampaignAccessPolicy_RESTRICTED {
		t.Fatalf("expected restricted access policy, got %v", proto.GetAccessPolicy())
	}
	if proto.GetLocale() != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("expected locale %v, got %v", commonv1.Locale_LOCALE_PT_BR, proto.GetLocale())
	}
	if proto.GetParticipantCount() != 2 || proto.GetCharacterCount() != 3 {
		t.Fatal("expected participant/character counts to map")
	}
	if proto.GetCoverAssetId() != "camp-cover-03" {
		t.Fatalf("expected cover asset id %q, got %q", "camp-cover-03", proto.GetCoverAssetId())
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
	if campaignv1.GmMode_GM_MODE_UNSPECIFIED != 0 {
		t.Fatal("expected gm mode unspecified to be 0")
	}
	if campaignv1.GmMode_AI != 1 {
		t.Fatal("expected gm mode AI to be 1")
	}
	if campaignv1.GmMode_HUMAN != 2 {
		t.Fatal("expected gm mode HUMAN to be 2")
	}
	if campaignv1.GmMode_HYBRID != 3 {
		t.Fatal("expected gm mode HYBRID to be 3")
	}

	if campaigntransport.CampaignStatusToProto(campaign.StatusArchived) != campaignv1.CampaignStatus_ARCHIVED {
		t.Fatal("expected archived campaign status")
	}
	if campaigntransport.CampaignStatusToProto(campaign.StatusUnspecified) != campaignv1.CampaignStatus_CAMPAIGN_STATUS_UNSPECIFIED {
		t.Fatal("expected unspecified campaign status")
	}

	if campaigntransport.GMModeFromProto(campaignv1.GmMode_AI) != campaign.GmModeAI {
		t.Fatal("expected gm mode AI")
	}
	if campaigntransport.GMModeFromProto(campaignv1.GmMode_GM_MODE_UNSPECIFIED) != campaign.GmModeUnspecified {
		t.Fatal("expected gm mode unspecified")
	}

	if campaigntransport.CampaignIntentFromProto(campaignv1.CampaignIntent_STARTER) != campaign.IntentStarter {
		t.Fatal("expected starter intent")
	}
	if campaigntransport.CampaignIntentFromProto(campaignv1.CampaignIntent_CAMPAIGN_INTENT_UNSPECIFIED) != campaign.IntentUnspecified {
		t.Fatal("expected unspecified intent")
	}

	if campaigntransport.CampaignAccessPolicyFromProto(campaignv1.CampaignAccessPolicy_PUBLIC) != campaign.AccessPolicyPublic {
		t.Fatal("expected public access policy")
	}
	if campaigntransport.CampaignAccessPolicyFromProto(campaignv1.CampaignAccessPolicy_CAMPAIGN_ACCESS_POLICY_UNSPECIFIED) != campaign.AccessPolicyUnspecified {
		t.Fatal("expected unspecified access policy")
	}

	if participanttransport.RoleFromProto(campaignv1.ParticipantRole_GM) != participant.RoleGM {
		t.Fatal("expected GM role")
	}
	if participanttransport.RoleFromProto(campaignv1.ParticipantRole_ROLE_UNSPECIFIED) != participant.RoleUnspecified {
		t.Fatal("expected unspecified role")
	}

	if participanttransport.ControllerFromProto(campaignv1.Controller_CONTROLLER_AI) != participant.ControllerAI {
		t.Fatal("expected AI controller")
	}
	if participanttransport.ControllerFromProto(campaignv1.Controller_CONTROLLER_UNSPECIFIED) != participant.ControllerUnspecified {
		t.Fatal("expected unspecified controller")
	}

	if participanttransport.CampaignAccessFromProto(campaignv1.CampaignAccess_CAMPAIGN_ACCESS_OWNER) != participant.CampaignAccessOwner {
		t.Fatal("expected owner campaign access")
	}
	if participanttransport.CampaignAccessFromProto(campaignv1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED) != participant.CampaignAccessUnspecified {
		t.Fatal("expected unspecified campaign access")
	}

	if charactertransport.KindToProto(character.KindNPC) != campaignv1.CharacterKind_NPC {
		t.Fatal("expected NPC character kind")
	}
	if charactertransport.KindFromProto(campaignv1.CharacterKind_CHARACTER_KIND_UNSPECIFIED) != character.KindUnspecified {
		t.Fatal("expected unspecified character kind")
	}

}

func TestCharacterToProtoOwnerParticipantID(t *testing.T) {
	created := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)

	withParticipant := charactertransport.CharacterToProto(storage.CharacterRecord{
		ID:                 "char-1",
		CampaignID:         "camp-1",
		Name:               "Hero",
		Kind:               character.KindPC,
		OwnerParticipantID: "part-1",
		CreatedAt:          created,
		UpdatedAt:          updated,
	})
	if withParticipant.GetOwnerParticipantId().GetValue() != "part-1" {
		t.Fatal("expected owner participant id wrapper to be set")
	}

	noParticipant := charactertransport.CharacterToProto(storage.CharacterRecord{
		ID:                 "char-2",
		CampaignID:         "camp-1",
		Name:               "NPC",
		Kind:               character.KindNPC,
		OwnerParticipantID: "  ",
		CreatedAt:          created,
		UpdatedAt:          updated,
	})
	if noParticipant.GetOwnerParticipantId() != nil {
		t.Fatal("expected owner participant id wrapper to be nil")
	}
}

func TestStructToMap(t *testing.T) {
	if got := handler.StructToMap(nil); got != nil {
		t.Fatalf("expected nil for nil input, got %v", got)
	}

	s, err := structpb.NewStruct(map[string]any{"key": "value", "num": float64(42)})
	if err != nil {
		t.Fatalf("failed to create struct: %v", err)
	}
	m := handler.StructToMap(s)
	if m["key"] != "value" {
		t.Fatalf("expected key=value, got %v", m["key"])
	}
	if m["num"] != float64(42) {
		t.Fatalf("expected num=42, got %v", m["num"])
	}
}

func TestValidateStructPayload(t *testing.T) {
	if err := handler.ValidateStructPayload(map[string]any{"valid": "ok"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := handler.ValidateStructPayload(map[string]any{"": "bad"}); err == nil {
		t.Fatal("expected error for empty key")
	}
	if err := handler.ValidateStructPayload(map[string]any{"  ": "bad"}); err == nil {
		t.Fatal("expected error for whitespace key")
	}
	if err := handler.ValidateStructPayload(nil); err != nil {
		t.Fatalf("unexpected error for nil: %v", err)
	}
	if err := handler.ValidateStructPayload(map[string]any{}); err != nil {
		t.Fatalf("unexpected error for empty map: %v", err)
	}
}

func TestParticipantToProto(t *testing.T) {
	created := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)

	p := participanttransport.ParticipantToProto(storage.ParticipantRecord{
		ID:             "part-1",
		CampaignID:     "camp-1",
		UserID:         "user-1",
		Name:           "Test Player",
		Role:           participant.RolePlayer,
		CampaignAccess: participant.CampaignAccessMember,
		Controller:     participant.ControllerHuman,
		CreatedAt:      created,
		UpdatedAt:      updated,
	})
	if p.GetId() != "part-1" {
		t.Fatalf("expected part-1, got %v", p.GetId())
	}
	if p.GetRole() != campaignv1.ParticipantRole_PLAYER {
		t.Fatalf("expected player role, got %v", p.GetRole())
	}
	if p.GetCampaignAccess() != campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER {
		t.Fatalf("expected member access, got %v", p.GetCampaignAccess())
	}
	if p.GetController() != campaignv1.Controller_CONTROLLER_HUMAN {
		t.Fatalf("expected human controller, got %v", p.GetController())
	}
}

func TestEnumConversionsExtended(t *testing.T) {
	// campaignStatusToProto remaining branches
	if campaigntransport.CampaignStatusToProto(campaign.StatusDraft) != campaignv1.CampaignStatus_DRAFT {
		t.Fatal("expected draft")
	}
	if campaigntransport.CampaignStatusToProto(campaign.StatusCompleted) != campaignv1.CampaignStatus_COMPLETED {
		t.Fatal("expected completed")
	}

	// gmModeFromProto remaining branches
	if campaigntransport.GMModeFromProto(campaignv1.GmMode_HUMAN) != campaign.GmModeHuman {
		t.Fatal("expected human")
	}
	if campaigntransport.GMModeFromProto(campaignv1.GmMode_HYBRID) != campaign.GmModeHybrid {
		t.Fatal("expected hybrid")
	}

	// gmModeToProto all branches
	if campaigntransport.GMModeToProto(campaign.GmModeHuman) != campaignv1.GmMode_HUMAN {
		t.Fatal("expected HUMAN")
	}
	if campaigntransport.GMModeToProto(campaign.GmModeAI) != campaignv1.GmMode_AI {
		t.Fatal("expected AI")
	}
	if campaigntransport.GMModeToProto(campaign.GmModeHybrid) != campaignv1.GmMode_HYBRID {
		t.Fatal("expected HYBRID")
	}
	if campaigntransport.GMModeToProto(campaign.GmModeUnspecified) != campaignv1.GmMode_GM_MODE_UNSPECIFIED {
		t.Fatal("expected GM_MODE_UNSPECIFIED")
	}

	// campaignIntentToProto all branches
	if campaigntransport.CampaignIntentToProto(campaign.IntentStandard) != campaignv1.CampaignIntent_STANDARD {
		t.Fatal("expected STANDARD")
	}
	if campaigntransport.CampaignIntentToProto(campaign.IntentStarter) != campaignv1.CampaignIntent_STARTER {
		t.Fatal("expected STARTER")
	}
	if campaigntransport.CampaignIntentToProto(campaign.IntentSandbox) != campaignv1.CampaignIntent_SANDBOX {
		t.Fatal("expected SANDBOX")
	}
	if campaigntransport.CampaignIntentToProto(campaign.IntentUnspecified) != campaignv1.CampaignIntent_CAMPAIGN_INTENT_UNSPECIFIED {
		t.Fatal("expected CAMPAIGN_INTENT_UNSPECIFIED")
	}

	// campaignAccessPolicyToProto all branches
	if campaigntransport.CampaignAccessPolicyToProto(campaign.AccessPolicyPrivate) != campaignv1.CampaignAccessPolicy_PRIVATE {
		t.Fatal("expected PRIVATE")
	}
	if campaigntransport.CampaignAccessPolicyToProto(campaign.AccessPolicyRestricted) != campaignv1.CampaignAccessPolicy_RESTRICTED {
		t.Fatal("expected RESTRICTED")
	}
	if campaigntransport.CampaignAccessPolicyToProto(campaign.AccessPolicyPublic) != campaignv1.CampaignAccessPolicy_PUBLIC {
		t.Fatal("expected PUBLIC")
	}
	if campaigntransport.CampaignAccessPolicyToProto(campaign.AccessPolicyUnspecified) != campaignv1.CampaignAccessPolicy_CAMPAIGN_ACCESS_POLICY_UNSPECIFIED {
		t.Fatal("expected CAMPAIGN_ACCESS_POLICY_UNSPECIFIED")
	}

	// participanttransport.RoleFromProto player
	if participanttransport.RoleFromProto(campaignv1.ParticipantRole_PLAYER) != participant.RolePlayer {
		t.Fatal("expected player")
	}

	// participanttransport.RoleToProto all branches
	if participanttransport.RoleToProto(participant.RoleGM) != campaignv1.ParticipantRole_GM {
		t.Fatal("expected GM")
	}
	if participanttransport.RoleToProto(participant.RolePlayer) != campaignv1.ParticipantRole_PLAYER {
		t.Fatal("expected PLAYER")
	}
	if participanttransport.RoleToProto(participant.RoleUnspecified) != campaignv1.ParticipantRole_ROLE_UNSPECIFIED {
		t.Fatal("expected ROLE_UNSPECIFIED")
	}

	// participanttransport.ControllerFromProto human
	if participanttransport.ControllerFromProto(campaignv1.Controller_CONTROLLER_HUMAN) != participant.ControllerHuman {
		t.Fatal("expected human")
	}

	// participanttransport.ControllerToProto all branches
	if participanttransport.ControllerToProto(participant.ControllerHuman) != campaignv1.Controller_CONTROLLER_HUMAN {
		t.Fatal("expected HUMAN")
	}
	if participanttransport.ControllerToProto(participant.ControllerAI) != campaignv1.Controller_CONTROLLER_AI {
		t.Fatal("expected AI")
	}
	if participanttransport.ControllerToProto(participant.ControllerUnspecified) != campaignv1.Controller_CONTROLLER_UNSPECIFIED {
		t.Fatal("expected UNSPECIFIED")
	}

	// participanttransport.CampaignAccessFromProto member + manager
	if participanttransport.CampaignAccessFromProto(campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER) != participant.CampaignAccessMember {
		t.Fatal("expected member")
	}
	if participanttransport.CampaignAccessFromProto(campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER) != participant.CampaignAccessManager {
		t.Fatal("expected manager")
	}

	// participanttransport.CampaignAccessToProto all branches
	if participanttransport.CampaignAccessToProto(participant.CampaignAccessMember) != campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER {
		t.Fatal("expected MEMBER")
	}
	if participanttransport.CampaignAccessToProto(participant.CampaignAccessManager) != campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER {
		t.Fatal("expected MANAGER")
	}
	if participanttransport.CampaignAccessToProto(participant.CampaignAccessOwner) != campaignv1.CampaignAccess_CAMPAIGN_ACCESS_OWNER {
		t.Fatal("expected OWNER")
	}
	if participanttransport.CampaignAccessToProto(participant.CampaignAccessUnspecified) != campaignv1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED {
		t.Fatal("expected UNSPECIFIED")
	}

	// charactertransport.KindFromProto all branches
	if charactertransport.KindFromProto(campaignv1.CharacterKind_PC) != character.KindPC {
		t.Fatal("expected PC")
	}
	if charactertransport.KindFromProto(campaignv1.CharacterKind_NPC) != character.KindNPC {
		t.Fatal("expected NPC")
	}

	// charactertransport.KindToProto all branches
	if charactertransport.KindToProto(character.KindPC) != campaignv1.CharacterKind_PC {
		t.Fatal("expected PC")
	}
	if charactertransport.KindToProto(character.KindUnspecified) != campaignv1.CharacterKind_CHARACTER_KIND_UNSPECIFIED {
		t.Fatal("expected UNSPECIFIED")
	}

	// gameSystemToProto / gameSystemFromProto
	sys := bridge.SystemIDDaggerheart
	if campaigntransport.GameSystemToProto(sys) != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatal("expected daggerheart mapping")
	}
	if campaigntransport.GameSystemFromProto(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART) != sys {
		t.Fatal("expected daggerheart reverse mapping")
	}
}
