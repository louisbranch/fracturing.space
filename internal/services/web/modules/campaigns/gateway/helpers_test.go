package gateway

import (
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestCampaignSystemLabel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		system commonv1.GameSystem
		want   string
	}{
		{commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, "daggerheart"},
		{commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, "unspecified"},
	} {
		if got := campaignSystemLabel(tc.system); got != tc.want {
			t.Errorf("campaignSystemLabel(%v) = %q, want %q", tc.system, got, tc.want)
		}
	}
}

func TestCampaignGMModeLabel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mode statev1.GmMode
		want string
	}{
		{statev1.GmMode_HUMAN, "human"},
		{statev1.GmMode_AI, "ai"},
		{statev1.GmMode_HYBRID, "hybrid"},
		{statev1.GmMode_GM_MODE_UNSPECIFIED, "unspecified"},
	} {
		if got := campaignGMModeLabel(tc.mode); got != tc.want {
			t.Errorf("campaignGMModeLabel(%v) = %q, want %q", tc.mode, got, tc.want)
		}
	}
}

func TestCampaignStatusLabel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		status statev1.CampaignStatus
		want   string
	}{
		{statev1.CampaignStatus_DRAFT, "draft"},
		{statev1.CampaignStatus_ACTIVE, "active"},
		{statev1.CampaignStatus_COMPLETED, "completed"},
		{statev1.CampaignStatus_ARCHIVED, "archived"},
		{statev1.CampaignStatus_CAMPAIGN_STATUS_UNSPECIFIED, "unspecified"},
	} {
		if got := campaignStatusLabel(tc.status); got != tc.want {
			t.Errorf("campaignStatusLabel(%v) = %q, want %q", tc.status, got, tc.want)
		}
	}
}

func TestCampaignLocaleLabel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		locale commonv1.Locale
		want   string
	}{
		{commonv1.Locale_LOCALE_EN_US, "en_us"},
		{commonv1.Locale_LOCALE_PT_BR, "pt_br"},
		{commonv1.Locale_LOCALE_UNSPECIFIED, "unspecified"},
	} {
		if got := campaignLocaleLabel(tc.locale); got != tc.want {
			t.Errorf("campaignLocaleLabel(%v) = %q, want %q", tc.locale, got, tc.want)
		}
	}
}

func TestParticipantRoleLabel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		role statev1.ParticipantRole
		want string
	}{
		{statev1.ParticipantRole_GM, "gm"},
		{statev1.ParticipantRole_PLAYER, "player"},
		{statev1.ParticipantRole_ROLE_UNSPECIFIED, "unspecified"},
	} {
		if got := participantRoleLabel(tc.role); got != tc.want {
			t.Errorf("participantRoleLabel(%v) = %q, want %q", tc.role, got, tc.want)
		}
	}
}

func TestParticipantCampaignAccessLabel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		access statev1.CampaignAccess
		want   string
	}{
		{statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER, "member"},
		{statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER, "manager"},
		{statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER, "owner"},
		{statev1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED, "unspecified"},
	} {
		if got := participantCampaignAccessLabel(tc.access); got != tc.want {
			t.Errorf("participantCampaignAccessLabel(%v) = %q, want %q", tc.access, got, tc.want)
		}
	}
}

func TestCharacterKindLabel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		kind statev1.CharacterKind
		want string
	}{
		{statev1.CharacterKind_PC, "pc"},
		{statev1.CharacterKind_NPC, "npc"},
		{statev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED, "unspecified"},
	} {
		if got := characterKindLabel(tc.kind); got != tc.want {
			t.Errorf("characterKindLabel(%v) = %q, want %q", tc.kind, got, tc.want)
		}
	}
}

func TestSessionStatusLabel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		status statev1.SessionStatus
		want   string
	}{
		{statev1.SessionStatus_SESSION_ACTIVE, "active"},
		{statev1.SessionStatus_SESSION_ENDED, "ended"},
		{statev1.SessionStatus_SESSION_STATUS_UNSPECIFIED, "unspecified"},
	} {
		if got := sessionStatusLabel(tc.status); got != tc.want {
			t.Errorf("sessionStatusLabel(%v) = %q, want %q", tc.status, got, tc.want)
		}
	}
}

func TestTimestampString(t *testing.T) {
	t.Parallel()
	if got := timestampString(nil); got != "" {
		t.Errorf("timestampString(nil) = %q, want empty", got)
	}
	ts := timestamppb.Now()
	if got := timestampString(ts); got == "" {
		t.Error("timestampString(non-nil) returned empty string")
	}
}

func TestFormatDamageDice(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		dice []*daggerheartv1.DiceSpec
		want string
	}{
		{"nil", nil, ""},
		{"empty", []*daggerheartv1.DiceSpec{}, ""},
		{"single", []*daggerheartv1.DiceSpec{{Count: 2, Sides: 8}}, "2d8"},
		{"zero count defaults to 1", []*daggerheartv1.DiceSpec{{Count: 0, Sides: 6}}, "1d6"},
		{"multiple", []*daggerheartv1.DiceSpec{{Count: 1, Sides: 6}, {Count: 1, Sides: 8}}, "1d6 + 1d8"},
		{"nil entry skipped", []*daggerheartv1.DiceSpec{nil, {Count: 1, Sides: 4}}, "1d4"},
		{"zero sides skipped", []*daggerheartv1.DiceSpec{{Count: 1, Sides: 0}}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatDamageDice(tc.dice); got != tc.want {
				t.Errorf("formatDamageDice = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMapGameSystemToProto(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		system campaignapp.GameSystem
		want   commonv1.GameSystem
	}{
		{campaignapp.GameSystemDaggerheart, commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART},
		{"unknown", commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED},
	} {
		if got := mapGameSystemToProto(tc.system); got != tc.want {
			t.Errorf("mapGameSystemToProto(%q) = %v, want %v", tc.system, got, tc.want)
		}
	}
}

func TestMapGmModeToProto(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mode campaignapp.GmMode
		want statev1.GmMode
	}{
		{campaignapp.GmModeHuman, statev1.GmMode_HUMAN},
		{campaignapp.GmModeAI, statev1.GmMode_AI},
		{campaignapp.GmModeHybrid, statev1.GmMode_HYBRID},
		{"unknown", statev1.GmMode_GM_MODE_UNSPECIFIED},
	} {
		if got := mapGmModeToProto(tc.mode); got != tc.want {
			t.Errorf("mapGmModeToProto(%q) = %v, want %v", tc.mode, got, tc.want)
		}
	}
}

func TestMapCharacterKindToProto(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		kind campaignapp.CharacterKind
		want statev1.CharacterKind
	}{
		{campaignapp.CharacterKindPC, statev1.CharacterKind_PC},
		{campaignapp.CharacterKindNPC, statev1.CharacterKind_NPC},
		{"unknown", statev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED},
	} {
		if got := mapCharacterKindToProto(tc.kind); got != tc.want {
			t.Errorf("mapCharacterKindToProto(%q) = %v, want %v", tc.kind, got, tc.want)
		}
	}
}

func TestMapParticipantRoleToProto(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		value string
		want  statev1.ParticipantRole
	}{
		{"gm", statev1.ParticipantRole_GM},
		{"GM", statev1.ParticipantRole_GM},
		{"participant_role_gm", statev1.ParticipantRole_GM},
		{"role_gm", statev1.ParticipantRole_GM},
		{"player", statev1.ParticipantRole_PLAYER},
		{"Player", statev1.ParticipantRole_PLAYER},
		{"unknown", statev1.ParticipantRole_ROLE_UNSPECIFIED},
		{"", statev1.ParticipantRole_ROLE_UNSPECIFIED},
	} {
		if got := mapParticipantRoleToProto(tc.value); got != tc.want {
			t.Errorf("mapParticipantRoleToProto(%q) = %v, want %v", tc.value, got, tc.want)
		}
	}
}

func TestMapParticipantAccessToProto(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		value string
		want  statev1.CampaignAccess
	}{
		{"member", statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER},
		{"manager", statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER},
		{"owner", statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER},
		{"campaign_access_member", statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER},
		{"unknown", statev1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED},
		{"", statev1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED},
	} {
		if got := mapParticipantAccessToProto(tc.value); got != tc.want {
			t.Errorf("mapParticipantAccessToProto(%q) = %v, want %v", tc.value, got, tc.want)
		}
	}
}

func TestDaggerheartHeritageKindLabel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		kind daggerheartv1.DaggerheartHeritageKind
		want string
	}{
		{daggerheartv1.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_ANCESTRY, "ancestry"},
		{daggerheartv1.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_COMMUNITY, "community"},
		{daggerheartv1.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_UNSPECIFIED, ""},
	} {
		if got := daggerheartHeritageKindLabel(tc.kind); got != tc.want {
			t.Errorf("daggerheartHeritageKindLabel(%v) = %q, want %q", tc.kind, got, tc.want)
		}
	}
}

func TestDaggerheartDomainCardTypeLabel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		cardType daggerheartv1.DaggerheartDomainCardType
		want     string
	}{
		{daggerheartv1.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_ABILITY, "ability"},
		{daggerheartv1.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_SPELL, "spell"},
		{daggerheartv1.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_GRIMOIRE, "grimoire"},
		{daggerheartv1.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_UNSPECIFIED, ""},
	} {
		if got := daggerheartDomainCardTypeLabel(tc.cardType); got != tc.want {
			t.Errorf("daggerheartDomainCardTypeLabel(%v) = %q, want %q", tc.cardType, got, tc.want)
		}
	}
}
