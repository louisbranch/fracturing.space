package campaigntransport

import (
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CampaignToProto converts a campaign projection record to its protobuf read
// model.
func CampaignToProto(c storage.CampaignRecord) *campaignv1.Campaign {
	return &campaignv1.Campaign{
		Id:               c.ID,
		Name:             c.Name,
		Locale:           LocaleStringToProto(c.Locale),
		System:           GameSystemToProto(c.System),
		Status:           CampaignStatusToProto(c.Status),
		GmMode:           GMModeToProto(c.GmMode),
		Intent:           CampaignIntentToProto(c.Intent),
		AccessPolicy:     CampaignAccessPolicyToProto(c.AccessPolicy),
		ParticipantCount: int32(c.ParticipantCount),
		CharacterCount:   int32(c.CharacterCount),
		ThemePrompt:      c.ThemePrompt,
		CoverAssetId:     c.CoverAssetID,
		CoverSetId:       c.CoverSetID,
		AiAgentId:        c.AIAgentID,
		CreatedAt:        timestamppb.New(c.CreatedAt),
		UpdatedAt:        timestamppb.New(c.UpdatedAt),
		CompletedAt:      timestampOrNil(c.CompletedAt),
		ArchivedAt:       timestampOrNil(c.ArchivedAt),
	}
}

// CampaignStatusToProto converts a campaign status to its protobuf enum.
func CampaignStatusToProto(status campaign.Status) campaignv1.CampaignStatus {
	switch status {
	case campaign.StatusDraft:
		return campaignv1.CampaignStatus_DRAFT
	case campaign.StatusActive:
		return campaignv1.CampaignStatus_ACTIVE
	case campaign.StatusCompleted:
		return campaignv1.CampaignStatus_COMPLETED
	case campaign.StatusArchived:
		return campaignv1.CampaignStatus_ARCHIVED
	default:
		return campaignv1.CampaignStatus_CAMPAIGN_STATUS_UNSPECIFIED
	}
}

// GMModeFromProto converts the protobuf GM mode to the domain value.
func GMModeFromProto(mode campaignv1.GmMode) campaign.GmMode {
	switch mode {
	case campaignv1.GmMode_HUMAN:
		return campaign.GmModeHuman
	case campaignv1.GmMode_AI:
		return campaign.GmModeAI
	case campaignv1.GmMode_HYBRID:
		return campaign.GmModeHybrid
	default:
		return campaign.GmModeUnspecified
	}
}

// GMModeToProto converts the domain GM mode to the protobuf enum.
func GMModeToProto(mode campaign.GmMode) campaignv1.GmMode {
	switch mode {
	case campaign.GmModeHuman:
		return campaignv1.GmMode_HUMAN
	case campaign.GmModeAI:
		return campaignv1.GmMode_AI
	case campaign.GmModeHybrid:
		return campaignv1.GmMode_HYBRID
	default:
		return campaignv1.GmMode_GM_MODE_UNSPECIFIED
	}
}

// CampaignIntentFromProto converts the protobuf intent to the domain value.
func CampaignIntentFromProto(intent campaignv1.CampaignIntent) campaign.Intent {
	switch intent {
	case campaignv1.CampaignIntent_STANDARD:
		return campaign.IntentStandard
	case campaignv1.CampaignIntent_STARTER:
		return campaign.IntentStarter
	case campaignv1.CampaignIntent_SANDBOX:
		return campaign.IntentSandbox
	default:
		return campaign.IntentUnspecified
	}
}

// CampaignIntentToProto converts the domain intent to the protobuf enum.
func CampaignIntentToProto(intent campaign.Intent) campaignv1.CampaignIntent {
	switch intent {
	case campaign.IntentStandard:
		return campaignv1.CampaignIntent_STANDARD
	case campaign.IntentStarter:
		return campaignv1.CampaignIntent_STARTER
	case campaign.IntentSandbox:
		return campaignv1.CampaignIntent_SANDBOX
	default:
		return campaignv1.CampaignIntent_CAMPAIGN_INTENT_UNSPECIFIED
	}
}

// CampaignAccessPolicyFromProto converts the protobuf access policy to the
// domain value.
func CampaignAccessPolicyFromProto(policy campaignv1.CampaignAccessPolicy) campaign.AccessPolicy {
	switch policy {
	case campaignv1.CampaignAccessPolicy_PRIVATE:
		return campaign.AccessPolicyPrivate
	case campaignv1.CampaignAccessPolicy_RESTRICTED:
		return campaign.AccessPolicyRestricted
	case campaignv1.CampaignAccessPolicy_PUBLIC:
		return campaign.AccessPolicyPublic
	default:
		return campaign.AccessPolicyUnspecified
	}
}

// CampaignAccessPolicyToProto converts the domain access policy to the protobuf
// enum.
func CampaignAccessPolicyToProto(policy campaign.AccessPolicy) campaignv1.CampaignAccessPolicy {
	switch policy {
	case campaign.AccessPolicyPrivate:
		return campaignv1.CampaignAccessPolicy_PRIVATE
	case campaign.AccessPolicyRestricted:
		return campaignv1.CampaignAccessPolicy_RESTRICTED
	case campaign.AccessPolicyPublic:
		return campaignv1.CampaignAccessPolicy_PUBLIC
	default:
		return campaignv1.CampaignAccessPolicy_CAMPAIGN_ACCESS_POLICY_UNSPECIFIED
	}
}

// GameSystemToProto converts the bridge system id to the common protobuf enum.
func GameSystemToProto(system bridge.SystemID) commonv1.GameSystem {
	switch system {
	case bridge.SystemIDDaggerheart:
		return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART
	default:
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED
	}
}

// GameSystemFromProto converts the common protobuf enum to the bridge system
// id.
func GameSystemFromProto(system commonv1.GameSystem) bridge.SystemID {
	switch system {
	case commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART:
		return bridge.SystemIDDaggerheart
	default:
		return ""
	}
}

// LocaleStringToProto converts a BCP-47 locale string to the proto enum,
// falling back to the default locale for unrecognized values.
func LocaleStringToProto(locale string) commonv1.Locale {
	parsed, _ := platformi18n.ParseLocale(locale)
	return platformi18n.NormalizeLocale(parsed)
}

func timestampOrNil(value *time.Time) *timestamppb.Timestamp {
	if value == nil {
		return nil
	}
	return timestamppb.New(*value)
}
