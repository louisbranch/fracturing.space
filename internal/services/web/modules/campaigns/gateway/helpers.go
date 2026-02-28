package gateway

import (
	"strconv"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func campaignSystemLabel(system commonv1.GameSystem) string {
	switch system {
	case commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART:
		return "Daggerheart"
	default:
		return "Unspecified"
	}
}

func campaignGMModeLabel(mode statev1.GmMode) string {
	switch mode {
	case statev1.GmMode_HUMAN:
		return "Human"
	case statev1.GmMode_AI:
		return "AI"
	case statev1.GmMode_HYBRID:
		return "Hybrid"
	default:
		return "Unspecified"
	}
}

func campaignStatusLabel(status statev1.CampaignStatus) string {
	switch status {
	case statev1.CampaignStatus_DRAFT:
		return "Draft"
	case statev1.CampaignStatus_ACTIVE:
		return "Active"
	case statev1.CampaignStatus_COMPLETED:
		return "Completed"
	case statev1.CampaignStatus_ARCHIVED:
		return "Archived"
	default:
		return "Unspecified"
	}
}

func campaignLocaleLabel(locale commonv1.Locale) string {
	switch locale {
	case commonv1.Locale_LOCALE_EN_US:
		return "English (US)"
	case commonv1.Locale_LOCALE_PT_BR:
		return "Portuguese (Brazil)"
	default:
		return "Unspecified"
	}
}

func campaignIntentLabel(intent statev1.CampaignIntent) string {
	switch intent {
	case statev1.CampaignIntent_STANDARD:
		return "Standard"
	case statev1.CampaignIntent_STARTER:
		return "Starter"
	case statev1.CampaignIntent_SANDBOX:
		return "Sandbox"
	default:
		return "Unspecified"
	}
}

func campaignAccessPolicyLabel(policy statev1.CampaignAccessPolicy) string {
	switch policy {
	case statev1.CampaignAccessPolicy_PRIVATE:
		return "Private"
	case statev1.CampaignAccessPolicy_RESTRICTED:
		return "Restricted"
	case statev1.CampaignAccessPolicy_PUBLIC:
		return "Public"
	default:
		return "Unspecified"
	}
}

func participantDisplayName(participant *statev1.Participant) string {
	if participant == nil {
		return "Unknown participant"
	}
	if name := strings.TrimSpace(participant.GetName()); name != "" {
		return name
	}
	if userID := strings.TrimSpace(participant.GetUserId()); userID != "" {
		return userID
	}
	if participantID := strings.TrimSpace(participant.GetId()); participantID != "" {
		return participantID
	}
	return "Unknown participant"
}

func participantRoleLabel(role statev1.ParticipantRole) string {
	switch role {
	case statev1.ParticipantRole_GM:
		return "GM"
	case statev1.ParticipantRole_PLAYER:
		return "Player"
	default:
		return "Unspecified"
	}
}

func participantCampaignAccessLabel(access statev1.CampaignAccess) string {
	switch access {
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER:
		return "Member"
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER:
		return "Manager"
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER:
		return "Owner"
	default:
		return "Unspecified"
	}
}

func participantControllerLabel(controller statev1.Controller) string {
	switch controller {
	case statev1.Controller_CONTROLLER_HUMAN:
		return "Human"
	case statev1.Controller_CONTROLLER_AI:
		return "AI"
	default:
		return "Unspecified"
	}
}

func characterDisplayName(character *statev1.Character) string {
	if character == nil {
		return "Unknown character"
	}
	if name := strings.TrimSpace(character.GetName()); name != "" {
		return name
	}
	if characterID := strings.TrimSpace(character.GetId()); characterID != "" {
		return characterID
	}
	return "Unknown character"
}

func characterKindLabel(kind statev1.CharacterKind) string {
	switch kind {
	case statev1.CharacterKind_PC:
		return "PC"
	case statev1.CharacterKind_NPC:
		return "NPC"
	default:
		return "Unspecified"
	}
}

func sessionStatusLabel(status statev1.SessionStatus) string {
	switch status {
	case statev1.SessionStatus_SESSION_ACTIVE:
		return "Active"
	case statev1.SessionStatus_SESSION_ENDED:
		return "Ended"
	default:
		return "Unspecified"
	}
}

func inviteStatusLabel(status statev1.InviteStatus) string {
	switch status {
	case statev1.InviteStatus_PENDING:
		return "Pending"
	case statev1.InviteStatus_CLAIMED:
		return "Claimed"
	case statev1.InviteStatus_REVOKED:
		return "Revoked"
	default:
		return "Unspecified"
	}
}

func timestampString(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return strings.TrimSpace(ts.AsTime().UTC().Format("2006-01-02 15:04 UTC"))
}

func int32ValueString(value *wrapperspb.Int32Value) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(int64(value.GetValue()), 10)
}

func daggerheartHeritageKindLabel(kind daggerheartv1.DaggerheartHeritageKind) string {
	switch kind {
	case daggerheartv1.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_ANCESTRY:
		return "ancestry"
	case daggerheartv1.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_COMMUNITY:
		return "community"
	default:
		return ""
	}
}

func daggerheartWeaponCategoryLabel(category daggerheartv1.DaggerheartWeaponCategory) string {
	switch category {
	case daggerheartv1.DaggerheartWeaponCategory_DAGGERHEART_WEAPON_CATEGORY_PRIMARY:
		return "primary"
	case daggerheartv1.DaggerheartWeaponCategory_DAGGERHEART_WEAPON_CATEGORY_SECONDARY:
		return "secondary"
	default:
		return ""
	}
}

// --- Domain enum → proto mapping ---

func mapGameSystemToProto(s campaignapp.GameSystem) commonv1.GameSystem {
	switch s {
	case campaignapp.GameSystemDaggerheart:
		return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART
	default:
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED
	}
}

func mapGmModeToProto(m campaignapp.GmMode) statev1.GmMode {
	switch m {
	case campaignapp.GmModeHuman:
		return statev1.GmMode_HUMAN
	case campaignapp.GmModeAI:
		return statev1.GmMode_AI
	case campaignapp.GmModeHybrid:
		return statev1.GmMode_HYBRID
	default:
		return statev1.GmMode_GM_MODE_UNSPECIFIED
	}
}

func mapCharacterKindToProto(k campaignapp.CharacterKind) statev1.CharacterKind {
	switch k {
	case campaignapp.CharacterKindPC:
		return statev1.CharacterKind_PC
	case campaignapp.CharacterKindNPC:
		return statev1.CharacterKind_NPC
	default:
		return statev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED
	}
}
