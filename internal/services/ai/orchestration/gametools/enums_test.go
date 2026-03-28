package gametools

import (
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestScenePhaseStatusToString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input statev1.ScenePhaseStatus
		want  string
	}{
		{statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM, "GM"},
		{statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_PLAYERS, "PLAYERS"},
		{statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM_REVIEW, "GM_REVIEW"},
		{statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_UNSPECIFIED, "UNSPECIFIED"},
	}
	for _, tc := range tests {
		if got := scenePhaseStatusToString(tc.input); got != tc.want {
			t.Errorf("scenePhaseStatusToString(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestScenePlayerSlotReviewStatusToString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input statev1.ScenePlayerSlotReviewStatus
		want  string
	}{
		{statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_OPEN, "OPEN"},
		{statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_UNDER_REVIEW, "UNDER_REVIEW"},
		{statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_ACCEPTED, "ACCEPTED"},
		{statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_CHANGES_REQUESTED, "CHANGES_REQUESTED"},
		{statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_UNSPECIFIED, "UNSPECIFIED"},
	}
	for _, tc := range tests {
		if got := scenePlayerSlotReviewStatusToString(tc.input); got != tc.want {
			t.Errorf("scenePlayerSlotReviewStatusToString(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestParticipantRoleToString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input statev1.ParticipantRole
		want  string
	}{
		{statev1.ParticipantRole_GM, "GM"},
		{statev1.ParticipantRole_PLAYER, "PLAYER"},
		{0, "UNSPECIFIED"},
	}
	for _, tc := range tests {
		if got := participantRoleToString(tc.input); got != tc.want {
			t.Errorf("participantRoleToString(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestControllerToString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input statev1.Controller
		want  string
	}{
		{statev1.Controller_CONTROLLER_HUMAN, "HUMAN"},
		{statev1.Controller_CONTROLLER_AI, "AI"},
		{statev1.Controller_CONTROLLER_UNSPECIFIED, "UNSPECIFIED"},
	}
	for _, tc := range tests {
		if got := controllerToString(tc.input); got != tc.want {
			t.Errorf("controllerToString(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestCharacterKindToString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input statev1.CharacterKind
		want  string
	}{
		{statev1.CharacterKind_PC, "PC"},
		{statev1.CharacterKind_NPC, "NPC"},
		{statev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED, "UNSPECIFIED"},
	}
	for _, tc := range tests {
		if got := characterKindToString(tc.input); got != tc.want {
			t.Errorf("characterKindToString(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestCampaignStatusToString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input statev1.CampaignStatus
		want  string
	}{
		{statev1.CampaignStatus_DRAFT, "DRAFT"},
		{statev1.CampaignStatus_ACTIVE, "ACTIVE"},
		{statev1.CampaignStatus_COMPLETED, "COMPLETED"},
		{statev1.CampaignStatus_ARCHIVED, "ARCHIVED"},
		{statev1.CampaignStatus_CAMPAIGN_STATUS_UNSPECIFIED, "UNSPECIFIED"},
	}
	for _, tc := range tests {
		if got := campaignStatusToString(tc.input); got != tc.want {
			t.Errorf("campaignStatusToString(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestGmModeToString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input statev1.GmMode
		want  string
	}{
		{statev1.GmMode_HUMAN, "HUMAN"},
		{statev1.GmMode_AI, "AI"},
		{statev1.GmMode_HYBRID, "HYBRID"},
		{statev1.GmMode_GM_MODE_UNSPECIFIED, "UNSPECIFIED"},
	}
	for _, tc := range tests {
		if got := gmModeToString(tc.input); got != tc.want {
			t.Errorf("gmModeToString(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestCampaignIntentToString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input statev1.CampaignIntent
		want  string
	}{
		{statev1.CampaignIntent_STANDARD, "STANDARD"},
		{statev1.CampaignIntent_STARTER, "STARTER"},
		{statev1.CampaignIntent_SANDBOX, "SANDBOX"},
		{statev1.CampaignIntent_CAMPAIGN_INTENT_UNSPECIFIED, "UNSPECIFIED"},
	}
	for _, tc := range tests {
		if got := campaignIntentToString(tc.input); got != tc.want {
			t.Errorf("campaignIntentToString(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestCampaignAccessPolicyToString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input statev1.CampaignAccessPolicy
		want  string
	}{
		{statev1.CampaignAccessPolicy_PRIVATE, "PRIVATE"},
		{statev1.CampaignAccessPolicy_RESTRICTED, "RESTRICTED"},
		{statev1.CampaignAccessPolicy_PUBLIC, "PUBLIC"},
		{statev1.CampaignAccessPolicy_CAMPAIGN_ACCESS_POLICY_UNSPECIFIED, "UNSPECIFIED"},
	}
	for _, tc := range tests {
		if got := campaignAccessPolicyToString(tc.input); got != tc.want {
			t.Errorf("campaignAccessPolicyToString(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestSessionStatusToString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input statev1.SessionStatus
		want  string
	}{
		{statev1.SessionStatus_SESSION_ACTIVE, "ACTIVE"},
		{statev1.SessionStatus_SESSION_ENDED, "ENDED"},
		{statev1.SessionStatus_SESSION_STATUS_UNSPECIFIED, "UNSPECIFIED"},
	}
	for _, tc := range tests {
		if got := sessionStatusToString(tc.input); got != tc.want {
			t.Errorf("sessionStatusToString(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFormatTimestamp(t *testing.T) {
	t.Parallel()
	ts := timestamppb.New(time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC))
	if got := formatTimestamp(ts); got != "2026-03-01T12:00:00Z" {
		t.Errorf("formatTimestamp = %q, want %q", got, "2026-03-01T12:00:00Z")
	}
	if got := formatTimestamp(nil); got != "" {
		t.Errorf("formatTimestamp(nil) = %q, want empty", got)
	}
}
