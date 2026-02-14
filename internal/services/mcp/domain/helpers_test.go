package domain

import (
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestCampaignStatusToString(t *testing.T) {
	tests := []struct {
		status statev1.CampaignStatus
		want   string
	}{
		{statev1.CampaignStatus_DRAFT, "DRAFT"},
		{statev1.CampaignStatus_ACTIVE, "ACTIVE"},
		{statev1.CampaignStatus_COMPLETED, "COMPLETED"},
		{statev1.CampaignStatus_ARCHIVED, "ARCHIVED"},
		{statev1.CampaignStatus_CAMPAIGN_STATUS_UNSPECIFIED, "UNSPECIFIED"},
		{statev1.CampaignStatus(99), "UNSPECIFIED"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := campaignStatusToString(tt.status); got != tt.want {
				t.Errorf("campaignStatusToString(%v) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestSessionStatusToString(t *testing.T) {
	tests := []struct {
		status statev1.SessionStatus
		want   string
	}{
		{statev1.SessionStatus_SESSION_ACTIVE, "ACTIVE"},
		{statev1.SessionStatus_SESSION_ENDED, "ENDED"},
		{statev1.SessionStatus_SESSION_STATUS_UNSPECIFIED, "UNSPECIFIED"},
		{statev1.SessionStatus(99), "UNSPECIFIED"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := sessionStatusToString(tt.status); got != tt.want {
				t.Errorf("sessionStatusToString(%v) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestGmModeToString(t *testing.T) {
	tests := []struct {
		mode statev1.GmMode
		want string
	}{
		{statev1.GmMode_HUMAN, "HUMAN"},
		{statev1.GmMode_AI, "AI"},
		{statev1.GmMode_HYBRID, "HYBRID"},
		{statev1.GmMode_GM_MODE_UNSPECIFIED, "UNSPECIFIED"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := gmModeToString(tt.mode); got != tt.want {
				t.Errorf("gmModeToString(%v) = %q, want %q", tt.mode, got, tt.want)
			}
		})
	}
}

func TestGmModeFromString(t *testing.T) {
	tests := []struct {
		input string
		want  statev1.GmMode
	}{
		{"HUMAN", statev1.GmMode_HUMAN},
		{"human", statev1.GmMode_HUMAN},
		{"AI", statev1.GmMode_AI},
		{"HYBRID", statev1.GmMode_HYBRID},
		{"", statev1.GmMode_GM_MODE_UNSPECIFIED},
		{"invalid", statev1.GmMode_GM_MODE_UNSPECIFIED},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := gmModeFromString(tt.input); got != tt.want {
				t.Errorf("gmModeFromString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestCampaignIntentConversions(t *testing.T) {
	t.Run("from string", func(t *testing.T) {
		tests := []struct {
			input string
			want  statev1.CampaignIntent
		}{
			{"STANDARD", statev1.CampaignIntent_STANDARD},
			{"starter", statev1.CampaignIntent_STARTER},
			{"campaign_intent_sandbox", statev1.CampaignIntent_SANDBOX},
			{"", statev1.CampaignIntent_CAMPAIGN_INTENT_UNSPECIFIED},
			{"invalid", statev1.CampaignIntent_CAMPAIGN_INTENT_UNSPECIFIED},
		}
		for _, tt := range tests {
			if got := campaignIntentFromString(tt.input); got != tt.want {
				t.Errorf("campaignIntentFromString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		}
	})

	t.Run("to string", func(t *testing.T) {
		tests := []struct {
			intent statev1.CampaignIntent
			want   string
		}{
			{statev1.CampaignIntent_STANDARD, "STANDARD"},
			{statev1.CampaignIntent_STARTER, "STARTER"},
			{statev1.CampaignIntent_SANDBOX, "SANDBOX"},
			{statev1.CampaignIntent_CAMPAIGN_INTENT_UNSPECIFIED, "UNSPECIFIED"},
		}
		for _, tt := range tests {
			if got := campaignIntentToString(tt.intent); got != tt.want {
				t.Errorf("campaignIntentToString(%v) = %q, want %q", tt.intent, got, tt.want)
			}
		}
	})
}

func TestCampaignAccessPolicyConversions(t *testing.T) {
	t.Run("from string", func(t *testing.T) {
		tests := []struct {
			input string
			want  statev1.CampaignAccessPolicy
		}{
			{"PRIVATE", statev1.CampaignAccessPolicy_PRIVATE},
			{"restricted", statev1.CampaignAccessPolicy_RESTRICTED},
			{"campaign_access_policy_public", statev1.CampaignAccessPolicy_PUBLIC},
			{"", statev1.CampaignAccessPolicy_CAMPAIGN_ACCESS_POLICY_UNSPECIFIED},
			{"invalid", statev1.CampaignAccessPolicy_CAMPAIGN_ACCESS_POLICY_UNSPECIFIED},
		}
		for _, tt := range tests {
			if got := campaignAccessPolicyFromString(tt.input); got != tt.want {
				t.Errorf("campaignAccessPolicyFromString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		}
	})

	t.Run("to string", func(t *testing.T) {
		tests := []struct {
			policy statev1.CampaignAccessPolicy
			want   string
		}{
			{statev1.CampaignAccessPolicy_PRIVATE, "PRIVATE"},
			{statev1.CampaignAccessPolicy_RESTRICTED, "RESTRICTED"},
			{statev1.CampaignAccessPolicy_PUBLIC, "PUBLIC"},
			{statev1.CampaignAccessPolicy_CAMPAIGN_ACCESS_POLICY_UNSPECIFIED, "UNSPECIFIED"},
		}
		for _, tt := range tests {
			if got := campaignAccessPolicyToString(tt.policy); got != tt.want {
				t.Errorf("campaignAccessPolicyToString(%v) = %q, want %q", tt.policy, got, tt.want)
			}
		}
	})
}

func TestGameSystemFromString(t *testing.T) {
	tests := []struct {
		input string
		want  commonv1.GameSystem
	}{
		{"DAGGERHEART", commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART},
		{"daggerheart", commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART},
		{"GAME_SYSTEM_DAGGERHEART", commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART},
		{"", commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED},
		{"invalid", commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := gameSystemFromString(tt.input); got != tt.want {
				t.Errorf("gameSystemFromString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParticipantRoleConversion(t *testing.T) {
	t.Run("from string", func(t *testing.T) {
		tests := []struct {
			input string
			want  statev1.ParticipantRole
		}{
			{"GM", statev1.ParticipantRole_GM},
			{"gm", statev1.ParticipantRole_GM},
			{"PLAYER", statev1.ParticipantRole_PLAYER},
			{"", statev1.ParticipantRole_ROLE_UNSPECIFIED},
		}
		for _, tt := range tests {
			if got := participantRoleFromString(tt.input); got != tt.want {
				t.Errorf("participantRoleFromString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		}
	})

	t.Run("to string", func(t *testing.T) {
		tests := []struct {
			role statev1.ParticipantRole
			want string
		}{
			{statev1.ParticipantRole_GM, "GM"},
			{statev1.ParticipantRole_PLAYER, "PLAYER"},
			{statev1.ParticipantRole_ROLE_UNSPECIFIED, "UNSPECIFIED"},
		}
		for _, tt := range tests {
			if got := participantRoleToString(tt.role); got != tt.want {
				t.Errorf("participantRoleToString(%v) = %q, want %q", tt.role, got, tt.want)
			}
		}
	})
}

func TestControllerConversion(t *testing.T) {
	t.Run("from string", func(t *testing.T) {
		tests := []struct {
			input string
			want  statev1.Controller
		}{
			{"HUMAN", statev1.Controller_CONTROLLER_HUMAN},
			{"human", statev1.Controller_CONTROLLER_HUMAN},
			{"AI", statev1.Controller_CONTROLLER_AI},
			{"", statev1.Controller_CONTROLLER_UNSPECIFIED},
		}
		for _, tt := range tests {
			if got := controllerFromString(tt.input); got != tt.want {
				t.Errorf("controllerFromString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		}
	})

	t.Run("to string", func(t *testing.T) {
		tests := []struct {
			ctrl statev1.Controller
			want string
		}{
			{statev1.Controller_CONTROLLER_HUMAN, "HUMAN"},
			{statev1.Controller_CONTROLLER_AI, "AI"},
			{statev1.Controller_CONTROLLER_UNSPECIFIED, "UNSPECIFIED"},
		}
		for _, tt := range tests {
			if got := controllerToString(tt.ctrl); got != tt.want {
				t.Errorf("controllerToString(%v) = %q, want %q", tt.ctrl, got, tt.want)
			}
		}
	})
}

func TestCharacterKindConversion(t *testing.T) {
	t.Run("from string", func(t *testing.T) {
		tests := []struct {
			input string
			want  statev1.CharacterKind
		}{
			{"PC", statev1.CharacterKind_PC},
			{"pc", statev1.CharacterKind_PC},
			{"NPC", statev1.CharacterKind_NPC},
			{"", statev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED},
		}
		for _, tt := range tests {
			if got := characterKindFromString(tt.input); got != tt.want {
				t.Errorf("characterKindFromString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		}
	})

	t.Run("to string", func(t *testing.T) {
		tests := []struct {
			kind statev1.CharacterKind
			want string
		}{
			{statev1.CharacterKind_PC, "PC"},
			{statev1.CharacterKind_NPC, "NPC"},
			{statev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED, "UNSPECIFIED"},
		}
		for _, tt := range tests {
			if got := characterKindToString(tt.kind); got != tt.want {
				t.Errorf("characterKindToString(%v) = %q, want %q", tt.kind, got, tt.want)
			}
		}
	})
}

func TestFormatTimestamp(t *testing.T) {
	t.Run("nil returns empty", func(t *testing.T) {
		if got := formatTimestamp(nil); got != "" {
			t.Errorf("formatTimestamp(nil) = %q, want empty", got)
		}
	})

	t.Run("valid timestamp", func(t *testing.T) {
		ts := timestamppb.Now()
		got := formatTimestamp(ts)
		if got == "" {
			t.Error("expected non-empty timestamp")
		}
	})
}

func TestRollModeConversion(t *testing.T) {
	t.Run("to proto", func(t *testing.T) {
		tests := []struct {
			input string
			want  commonv1.RollMode
		}{
			{"REPLAY", commonv1.RollMode_REPLAY},
			{"replay", commonv1.RollMode_REPLAY},
			{"LIVE", commonv1.RollMode_LIVE},
			{"", commonv1.RollMode_ROLL_MODE_UNSPECIFIED},
			{"unknown", commonv1.RollMode_ROLL_MODE_UNSPECIFIED},
		}
		for _, tt := range tests {
			if got := rollModeToProto(tt.input); got != tt.want {
				t.Errorf("rollModeToProto(%q) = %v, want %v", tt.input, got, tt.want)
			}
		}
	})

	t.Run("label", func(t *testing.T) {
		tests := []struct {
			mode commonv1.RollMode
			want string
		}{
			{commonv1.RollMode_REPLAY, "REPLAY"},
			{commonv1.RollMode_LIVE, "LIVE"},
			{commonv1.RollMode_ROLL_MODE_UNSPECIFIED, ""},
		}
		for _, tt := range tests {
			if got := rollModeLabel(tt.mode); got != tt.want {
				t.Errorf("rollModeLabel(%v) = %q, want %q", tt.mode, got, tt.want)
			}
		}
	})
}

func TestCharacterProfileResultFromProto(t *testing.T) {
	t.Run("nil profile", func(t *testing.T) {
		result := characterProfileResultFromProto(&statev1.CharacterProfile{})
		if result.HpMax != 0 {
			t.Errorf("expected zero hp_max, got %d", result.HpMax)
		}
	})

	t.Run("with daggerheart fields", func(t *testing.T) {
		profile := &statev1.CharacterProfile{
			CharacterId: "char-1",
			SystemProfile: &statev1.CharacterProfile_Daggerheart{
				Daggerheart: &daggerheartv1.DaggerheartProfile{
					HpMax:           20,
					StressMax:       wrapperspb.Int32(6),
					Evasion:         wrapperspb.Int32(12),
					MajorThreshold:  wrapperspb.Int32(7),
					SevereThreshold: wrapperspb.Int32(14),
					Agility:         wrapperspb.Int32(2),
					Strength:        wrapperspb.Int32(1),
					Finesse:         wrapperspb.Int32(3),
					Instinct:        wrapperspb.Int32(0),
					Presence:        wrapperspb.Int32(-1),
					Knowledge:       wrapperspb.Int32(4),
				},
			},
		}

		result := characterProfileResultFromProto(profile)
		if result.CharacterID != "char-1" {
			t.Errorf("expected character ID %q, got %q", "char-1", result.CharacterID)
		}
		if result.HpMax != 20 {
			t.Errorf("expected hp_max 20, got %d", result.HpMax)
		}
		if result.StressMax != 6 {
			t.Errorf("expected stress_max 6, got %d", result.StressMax)
		}
		if result.Evasion != 12 {
			t.Errorf("expected evasion 12, got %d", result.Evasion)
		}
		if result.Agility != 2 {
			t.Errorf("expected agility 2, got %d", result.Agility)
		}
		if result.Knowledge != 4 {
			t.Errorf("expected knowledge 4, got %d", result.Knowledge)
		}
	})
}

func TestCharacterStateResultFromProto(t *testing.T) {
	t.Run("nil state", func(t *testing.T) {
		result := characterStateResultFromProto(&statev1.CharacterState{})
		if result.Hp != 0 {
			t.Errorf("expected zero hp, got %d", result.Hp)
		}
	})

	t.Run("with daggerheart fields", func(t *testing.T) {
		state := &statev1.CharacterState{
			CharacterId: "char-1",
			SystemState: &statev1.CharacterState_Daggerheart{
				Daggerheart: &daggerheartv1.DaggerheartCharacterState{
					Hp:     15,
					Hope:   3,
					Stress: 2,
				},
			},
		}

		result := characterStateResultFromProto(state)
		if result.CharacterID != "char-1" {
			t.Errorf("expected character ID %q, got %q", "char-1", result.CharacterID)
		}
		if result.Hp != 15 {
			t.Errorf("expected hp 15, got %d", result.Hp)
		}
		if result.Hope != 3 {
			t.Errorf("expected hope 3, got %d", result.Hope)
		}
		if result.Stress != 2 {
			t.Errorf("expected stress 2, got %d", result.Stress)
		}
	})
}

func TestIntSlice(t *testing.T) {
	got := intSlice([]int32{1, 2, 3})
	if len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Errorf("intSlice([1,2,3]) = %v", got)
	}

	got = intSlice(nil)
	if len(got) != 0 {
		t.Errorf("intSlice(nil) should be empty, got %v", got)
	}
}
