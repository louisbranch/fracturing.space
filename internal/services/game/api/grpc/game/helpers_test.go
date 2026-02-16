package game

import (
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestCampaignToProto(t *testing.T) {
	created := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	updated := created.Add(2 * time.Hour)
	completed := created.Add(24 * time.Hour)
	archived := created.Add(48 * time.Hour)

	proto := campaignToProto(storage.CampaignRecord{
		ID:               "camp-1",
		Name:             "Campaign",
		Locale:           commonv1.Locale_LOCALE_PT_BR,
		System:           commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		Status:           campaign.StatusActive,
		GmMode:           campaign.GmModeHybrid,
		Intent:           campaign.IntentStarter,
		AccessPolicy:     campaign.AccessPolicyRestricted,
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
	if campaignStatusToProto(campaign.StatusArchived) != campaignv1.CampaignStatus_ARCHIVED {
		t.Fatal("expected archived campaign status")
	}
	if campaignStatusToProto(campaign.StatusUnspecified) != campaignv1.CampaignStatus_CAMPAIGN_STATUS_UNSPECIFIED {
		t.Fatal("expected unspecified campaign status")
	}

	if gmModeFromProto(campaignv1.GmMode_AI) != campaign.GmModeAI {
		t.Fatal("expected gm mode AI")
	}
	if gmModeFromProto(campaignv1.GmMode_GM_MODE_UNSPECIFIED) != campaign.GmModeUnspecified {
		t.Fatal("expected gm mode unspecified")
	}

	if campaignIntentFromProto(campaignv1.CampaignIntent_STARTER) != campaign.IntentStarter {
		t.Fatal("expected starter intent")
	}
	if campaignIntentFromProto(campaignv1.CampaignIntent_CAMPAIGN_INTENT_UNSPECIFIED) != campaign.IntentUnspecified {
		t.Fatal("expected unspecified intent")
	}

	if campaignAccessPolicyFromProto(campaignv1.CampaignAccessPolicy_PUBLIC) != campaign.AccessPolicyPublic {
		t.Fatal("expected public access policy")
	}
	if campaignAccessPolicyFromProto(campaignv1.CampaignAccessPolicy_CAMPAIGN_ACCESS_POLICY_UNSPECIFIED) != campaign.AccessPolicyUnspecified {
		t.Fatal("expected unspecified access policy")
	}

	if participantRoleFromProto(campaignv1.ParticipantRole_GM) != participant.RoleGM {
		t.Fatal("expected GM role")
	}
	if participantRoleFromProto(campaignv1.ParticipantRole_ROLE_UNSPECIFIED) != participant.RoleUnspecified {
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

	if characterKindToProto(character.KindNPC) != campaignv1.CharacterKind_NPC {
		t.Fatal("expected NPC character kind")
	}
	if characterKindFromProto(campaignv1.CharacterKind_CHARACTER_KIND_UNSPECIFIED) != character.KindUnspecified {
		t.Fatal("expected unspecified character kind")
	}

	if sessionStatusToProto(session.StatusEnded) != campaignv1.SessionStatus_SESSION_ENDED {
		t.Fatal("expected ended session status")
	}
}

func TestCharacterToProtoParticipantID(t *testing.T) {
	created := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)

	withParticipant := characterToProto(storage.CharacterRecord{
		ID:            "char-1",
		CampaignID:    "camp-1",
		Name:          "Hero",
		Kind:          character.KindPC,
		ParticipantID: "part-1",
		CreatedAt:     created,
		UpdatedAt:     updated,
	})
	if withParticipant.GetParticipantId().GetValue() != "part-1" {
		t.Fatal("expected participant id wrapper to be set")
	}

	noParticipant := characterToProto(storage.CharacterRecord{
		ID:            "char-2",
		CampaignID:    "camp-1",
		Name:          "NPC",
		Kind:          character.KindNPC,
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

	withEnd := sessionToProto(storage.SessionRecord{
		ID:         "sess-1",
		CampaignID: "camp-1",
		Name:       "Session",
		Status:     session.StatusEnded,
		StartedAt:  started,
		UpdatedAt:  updated,
		EndedAt:    &ended,
	})
	if withEnd.GetEndedAt().AsTime().UTC() != ended {
		t.Fatal("expected ended_at to be set")
	}

	noEnd := sessionToProto(storage.SessionRecord{
		ID:         "sess-2",
		CampaignID: "camp-1",
		Name:       "Active",
		Status:     session.StatusActive,
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

func TestInviteStatusToProto(t *testing.T) {
	tests := []struct {
		name   string
		input  invite.Status
		expect campaignv1.InviteStatus
	}{
		{"pending", invite.StatusPending, campaignv1.InviteStatus_PENDING},
		{"claimed", invite.StatusClaimed, campaignv1.InviteStatus_CLAIMED},
		{"revoked", invite.StatusRevoked, campaignv1.InviteStatus_REVOKED},
		{"unspecified", invite.StatusUnspecified, campaignv1.InviteStatus_INVITE_STATUS_UNSPECIFIED},
		{"unknown", invite.Status("unknown"), campaignv1.InviteStatus_INVITE_STATUS_UNSPECIFIED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inviteStatusToProto(tt.input); got != tt.expect {
				t.Fatalf("got %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestInviteStatusFromProto(t *testing.T) {
	tests := []struct {
		name   string
		input  campaignv1.InviteStatus
		expect invite.Status
	}{
		{"pending", campaignv1.InviteStatus_PENDING, invite.StatusPending},
		{"claimed", campaignv1.InviteStatus_CLAIMED, invite.StatusClaimed},
		{"revoked", campaignv1.InviteStatus_REVOKED, invite.StatusRevoked},
		{"unspecified", campaignv1.InviteStatus_INVITE_STATUS_UNSPECIFIED, invite.StatusUnspecified},
		{"unknown", campaignv1.InviteStatus(99), invite.StatusUnspecified},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inviteStatusFromProto(tt.input); got != tt.expect {
				t.Fatalf("got %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestSessionGateStatusToProto(t *testing.T) {
	tests := []struct {
		name   string
		input  session.GateStatus
		expect campaignv1.SessionGateStatus
	}{
		{"open", session.GateStatusOpen, campaignv1.SessionGateStatus_SESSION_GATE_OPEN},
		{"resolved", session.GateStatusResolved, campaignv1.SessionGateStatus_SESSION_GATE_RESOLVED},
		{"abandoned", session.GateStatusAbandoned, campaignv1.SessionGateStatus_SESSION_GATE_ABANDONED},
		{"open_uppercase", session.GateStatus(" OPEN "), campaignv1.SessionGateStatus_SESSION_GATE_OPEN},
		{"empty", session.GateStatus(""), campaignv1.SessionGateStatus_SESSION_GATE_STATUS_UNSPECIFIED},
		{"unknown", session.GateStatus("invalid"), campaignv1.SessionGateStatus_SESSION_GATE_STATUS_UNSPECIFIED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sessionGateStatusToProto(tt.input); got != tt.expect {
				t.Fatalf("got %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestSessionSpotlightTypeFromProto(t *testing.T) {
	tests := []struct {
		name      string
		input     campaignv1.SessionSpotlightType
		expect    session.SpotlightType
		wantError bool
	}{
		{"gm", campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_GM, session.SpotlightTypeGM, false},
		{"character", campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_CHARACTER, session.SpotlightTypeCharacter, false},
		{"unspecified", campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_UNSPECIFIED, "", true},
		{"unknown", campaignv1.SessionSpotlightType(99), "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sessionSpotlightTypeFromProto(tt.input)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expect {
				t.Fatalf("got %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestSessionSpotlightTypeToProto(t *testing.T) {
	tests := []struct {
		name   string
		input  session.SpotlightType
		expect campaignv1.SessionSpotlightType
	}{
		{"gm", session.SpotlightTypeGM, campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_GM},
		{"character", session.SpotlightTypeCharacter, campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_CHARACTER},
		{"gm_uppercase", session.SpotlightType(" GM "), campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_GM},
		{"empty", session.SpotlightType(""), campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_UNSPECIFIED},
		{"unknown", session.SpotlightType("invalid"), campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_UNSPECIFIED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sessionSpotlightTypeToProto(tt.input); got != tt.expect {
				t.Fatalf("got %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestStructToMap(t *testing.T) {
	if got := structToMap(nil); got != nil {
		t.Fatalf("expected nil for nil input, got %v", got)
	}

	s, err := structpb.NewStruct(map[string]any{"key": "value", "num": float64(42)})
	if err != nil {
		t.Fatalf("failed to create struct: %v", err)
	}
	m := structToMap(s)
	if m["key"] != "value" {
		t.Fatalf("expected key=value, got %v", m["key"])
	}
	if m["num"] != float64(42) {
		t.Fatalf("expected num=42, got %v", m["num"])
	}
}

func TestValidateStructPayload(t *testing.T) {
	if err := validateStructPayload(map[string]any{"valid": "ok"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateStructPayload(map[string]any{"": "bad"}); err == nil {
		t.Fatal("expected error for empty key")
	}
	if err := validateStructPayload(map[string]any{"  ": "bad"}); err == nil {
		t.Fatal("expected error for whitespace key")
	}
	if err := validateStructPayload(nil); err != nil {
		t.Fatalf("unexpected error for nil: %v", err)
	}
	if err := validateStructPayload(map[string]any{}); err != nil {
		t.Fatalf("unexpected error for empty map: %v", err)
	}
}

func TestStructFromJSON(t *testing.T) {
	s, err := structFromJSON(nil)
	if err != nil || s != nil {
		t.Fatal("expected nil,nil for nil input")
	}

	s, err = structFromJSON([]byte{})
	if err != nil || s != nil {
		t.Fatal("expected nil,nil for empty input")
	}

	s, err = structFromJSON([]byte(`{"hello":"world"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.AsMap()["hello"] != "world" {
		t.Fatal("expected hello=world")
	}

	_, err = structFromJSON([]byte(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestSessionGateToProto(t *testing.T) {
	created := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	resolved := created.Add(time.Hour)

	gate, err := sessionGateToProto(storage.SessionGate{
		GateID:              "gate-1",
		CampaignID:          "camp-1",
		SessionID:           "sess-1",
		GateType:            "decision",
		Status:              "open",
		Reason:              "test",
		CreatedAt:           created,
		CreatedByActorType:  "user",
		CreatedByActorID:    "user-1",
		ResolvedAt:          &resolved,
		ResolvedByActorType: "user",
		ResolvedByActorID:   "user-2",
		MetadataJSON:        []byte(`{"key":"val"}`),
		ResolutionJSON:      []byte(`{"choice":"yes"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gate.GetId() != "gate-1" {
		t.Fatalf("expected gate id gate-1, got %v", gate.GetId())
	}
	if gate.GetStatus() != campaignv1.SessionGateStatus_SESSION_GATE_OPEN {
		t.Fatalf("expected open status, got %v", gate.GetStatus())
	}
	if gate.GetMetadata().AsMap()["key"] != "val" {
		t.Fatal("expected metadata key=val")
	}
	if gate.GetResolution().AsMap()["choice"] != "yes" {
		t.Fatal("expected resolution choice=yes")
	}

	// nil metadata/resolution
	gate2, err := sessionGateToProto(storage.SessionGate{
		GateID:     "gate-2",
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		GateType:   "decision",
		Status:     "resolved",
		CreatedAt:  created,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gate2.GetMetadata() != nil {
		t.Fatal("expected nil metadata")
	}

	// bad metadata JSON
	_, err = sessionGateToProto(storage.SessionGate{
		GateID:       "gate-3",
		MetadataJSON: []byte(`{invalid`),
		CreatedAt:    created,
	})
	if err == nil {
		t.Fatal("expected error for bad metadata JSON")
	}

	// bad resolution JSON
	_, err = sessionGateToProto(storage.SessionGate{
		GateID:         "gate-4",
		ResolutionJSON: []byte(`{invalid`),
		CreatedAt:      created,
	})
	if err == nil {
		t.Fatal("expected error for bad resolution JSON")
	}
}

func TestSessionSpotlightToProto(t *testing.T) {
	updated := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)

	spotlight := sessionSpotlightToProto(storage.SessionSpotlight{
		CampaignID:         "camp-1",
		SessionID:          "sess-1",
		SpotlightType:      "gm",
		CharacterID:        "",
		UpdatedAt:          updated,
		UpdatedByActorType: "user",
		UpdatedByActorID:   "user-1",
	})
	if spotlight.GetType() != campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_GM {
		t.Fatalf("expected gm spotlight type, got %v", spotlight.GetType())
	}
	if spotlight.GetCampaignId() != "camp-1" {
		t.Fatalf("expected camp-1, got %v", spotlight.GetCampaignId())
	}
}

func TestInviteToProto(t *testing.T) {
	created := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)

	inv := inviteToProto(storage.InviteRecord{
		ID:                     "inv-1",
		CampaignID:             "camp-1",
		ParticipantID:          "part-1",
		RecipientUserID:        "user-2",
		Status:                 invite.StatusClaimed,
		CreatedByParticipantID: "part-gm",
		CreatedAt:              created,
		UpdatedAt:              updated,
	})
	if inv.GetId() != "inv-1" {
		t.Fatalf("expected inv-1, got %v", inv.GetId())
	}
	if inv.GetStatus() != campaignv1.InviteStatus_CLAIMED {
		t.Fatalf("expected claimed, got %v", inv.GetStatus())
	}
	if inv.GetRecipientUserId() != "user-2" {
		t.Fatalf("expected user-2, got %v", inv.GetRecipientUserId())
	}
}

func TestParticipantToProto(t *testing.T) {
	created := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)

	p := participantToProto(storage.ParticipantRecord{
		ID:             "part-1",
		CampaignID:     "camp-1",
		UserID:         "user-1",
		DisplayName:    "Test Player",
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
	if campaignStatusToProto(campaign.StatusDraft) != campaignv1.CampaignStatus_DRAFT {
		t.Fatal("expected draft")
	}
	if campaignStatusToProto(campaign.StatusCompleted) != campaignv1.CampaignStatus_COMPLETED {
		t.Fatal("expected completed")
	}

	// gmModeFromProto remaining branches
	if gmModeFromProto(campaignv1.GmMode_HUMAN) != campaign.GmModeHuman {
		t.Fatal("expected human")
	}
	if gmModeFromProto(campaignv1.GmMode_HYBRID) != campaign.GmModeHybrid {
		t.Fatal("expected hybrid")
	}

	// gmModeToProto all branches
	if gmModeToProto(campaign.GmModeHuman) != campaignv1.GmMode_HUMAN {
		t.Fatal("expected HUMAN")
	}
	if gmModeToProto(campaign.GmModeAI) != campaignv1.GmMode_AI {
		t.Fatal("expected AI")
	}
	if gmModeToProto(campaign.GmModeHybrid) != campaignv1.GmMode_HYBRID {
		t.Fatal("expected HYBRID")
	}
	if gmModeToProto(campaign.GmModeUnspecified) != campaignv1.GmMode_GM_MODE_UNSPECIFIED {
		t.Fatal("expected GM_MODE_UNSPECIFIED")
	}

	// campaignIntentToProto all branches
	if campaignIntentToProto(campaign.IntentStandard) != campaignv1.CampaignIntent_STANDARD {
		t.Fatal("expected STANDARD")
	}
	if campaignIntentToProto(campaign.IntentStarter) != campaignv1.CampaignIntent_STARTER {
		t.Fatal("expected STARTER")
	}
	if campaignIntentToProto(campaign.IntentSandbox) != campaignv1.CampaignIntent_SANDBOX {
		t.Fatal("expected SANDBOX")
	}
	if campaignIntentToProto(campaign.IntentUnspecified) != campaignv1.CampaignIntent_CAMPAIGN_INTENT_UNSPECIFIED {
		t.Fatal("expected CAMPAIGN_INTENT_UNSPECIFIED")
	}

	// campaignAccessPolicyToProto all branches
	if campaignAccessPolicyToProto(campaign.AccessPolicyPrivate) != campaignv1.CampaignAccessPolicy_PRIVATE {
		t.Fatal("expected PRIVATE")
	}
	if campaignAccessPolicyToProto(campaign.AccessPolicyRestricted) != campaignv1.CampaignAccessPolicy_RESTRICTED {
		t.Fatal("expected RESTRICTED")
	}
	if campaignAccessPolicyToProto(campaign.AccessPolicyPublic) != campaignv1.CampaignAccessPolicy_PUBLIC {
		t.Fatal("expected PUBLIC")
	}
	if campaignAccessPolicyToProto(campaign.AccessPolicyUnspecified) != campaignv1.CampaignAccessPolicy_CAMPAIGN_ACCESS_POLICY_UNSPECIFIED {
		t.Fatal("expected CAMPAIGN_ACCESS_POLICY_UNSPECIFIED")
	}

	// participantRoleFromProto player
	if participantRoleFromProto(campaignv1.ParticipantRole_PLAYER) != participant.RolePlayer {
		t.Fatal("expected player")
	}

	// participantRoleToProto all branches
	if participantRoleToProto(participant.RoleGM) != campaignv1.ParticipantRole_GM {
		t.Fatal("expected GM")
	}
	if participantRoleToProto(participant.RolePlayer) != campaignv1.ParticipantRole_PLAYER {
		t.Fatal("expected PLAYER")
	}
	if participantRoleToProto(participant.RoleUnspecified) != campaignv1.ParticipantRole_ROLE_UNSPECIFIED {
		t.Fatal("expected ROLE_UNSPECIFIED")
	}

	// controllerFromProto human
	if controllerFromProto(campaignv1.Controller_CONTROLLER_HUMAN) != participant.ControllerHuman {
		t.Fatal("expected human")
	}

	// controllerToProto all branches
	if controllerToProto(participant.ControllerHuman) != campaignv1.Controller_CONTROLLER_HUMAN {
		t.Fatal("expected HUMAN")
	}
	if controllerToProto(participant.ControllerAI) != campaignv1.Controller_CONTROLLER_AI {
		t.Fatal("expected AI")
	}
	if controllerToProto(participant.ControllerUnspecified) != campaignv1.Controller_CONTROLLER_UNSPECIFIED {
		t.Fatal("expected UNSPECIFIED")
	}

	// campaignAccessFromProto member + manager
	if campaignAccessFromProto(campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER) != participant.CampaignAccessMember {
		t.Fatal("expected member")
	}
	if campaignAccessFromProto(campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER) != participant.CampaignAccessManager {
		t.Fatal("expected manager")
	}

	// campaignAccessToProto all branches
	if campaignAccessToProto(participant.CampaignAccessMember) != campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER {
		t.Fatal("expected MEMBER")
	}
	if campaignAccessToProto(participant.CampaignAccessManager) != campaignv1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER {
		t.Fatal("expected MANAGER")
	}
	if campaignAccessToProto(participant.CampaignAccessOwner) != campaignv1.CampaignAccess_CAMPAIGN_ACCESS_OWNER {
		t.Fatal("expected OWNER")
	}
	if campaignAccessToProto(participant.CampaignAccessUnspecified) != campaignv1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED {
		t.Fatal("expected UNSPECIFIED")
	}

	// characterKindFromProto all branches
	if characterKindFromProto(campaignv1.CharacterKind_PC) != character.KindPC {
		t.Fatal("expected PC")
	}
	if characterKindFromProto(campaignv1.CharacterKind_NPC) != character.KindNPC {
		t.Fatal("expected NPC")
	}

	// characterKindToProto all branches
	if characterKindToProto(character.KindPC) != campaignv1.CharacterKind_PC {
		t.Fatal("expected PC")
	}
	if characterKindToProto(character.KindUnspecified) != campaignv1.CharacterKind_CHARACTER_KIND_UNSPECIFIED {
		t.Fatal("expected UNSPECIFIED")
	}

	// sessionStatusToProto all branches
	if sessionStatusToProto(session.StatusActive) != campaignv1.SessionStatus_SESSION_ACTIVE {
		t.Fatal("expected ACTIVE")
	}
	if sessionStatusToProto(session.StatusUnspecified) != campaignv1.SessionStatus_SESSION_STATUS_UNSPECIFIED {
		t.Fatal("expected UNSPECIFIED")
	}

	// gameSystemToProto / gameSystemFromProto
	sys := commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART
	if gameSystemToProto(sys) != sys {
		t.Fatal("expected passthrough")
	}
	if gameSystemFromProto(sys) != sys {
		t.Fatal("expected passthrough")
	}
}
