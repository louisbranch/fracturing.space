package game

import (
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Campaign proto conversion helpers.
func campaignToProto(c campaign.Campaign) *campaignv1.Campaign {
	return &campaignv1.Campaign{
		Id:               c.ID,
		Name:             c.Name,
		Locale:           platformi18n.NormalizeLocale(c.Locale),
		System:           gameSystemToProto(c.System),
		Status:           campaignStatusToProto(c.Status),
		GmMode:           gmModeToProto(c.GmMode),
		Intent:           campaignIntentToProto(c.Intent),
		AccessPolicy:     campaignAccessPolicyToProto(c.AccessPolicy),
		ParticipantCount: int32(c.ParticipantCount),
		CharacterCount:   int32(c.CharacterCount),
		ThemePrompt:      c.ThemePrompt,
		CreatedAt:        timestamppb.New(c.CreatedAt),
		UpdatedAt:        timestamppb.New(c.UpdatedAt),
		CompletedAt:      timestampOrNil(c.CompletedAt),
		ArchivedAt:       timestampOrNil(c.ArchivedAt),
	}
}

func campaignStatusToProto(status campaign.CampaignStatus) campaignv1.CampaignStatus {
	switch status {
	case campaign.CampaignStatusDraft:
		return campaignv1.CampaignStatus_DRAFT
	case campaign.CampaignStatusActive:
		return campaignv1.CampaignStatus_ACTIVE
	case campaign.CampaignStatusCompleted:
		return campaignv1.CampaignStatus_COMPLETED
	case campaign.CampaignStatusArchived:
		return campaignv1.CampaignStatus_ARCHIVED
	default:
		return campaignv1.CampaignStatus_CAMPAIGN_STATUS_UNSPECIFIED
	}
}

func gmModeFromProto(mode campaignv1.GmMode) campaign.GmMode {
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

func gmModeToProto(mode campaign.GmMode) campaignv1.GmMode {
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

func campaignIntentFromProto(intent campaignv1.CampaignIntent) campaign.CampaignIntent {
	switch intent {
	case campaignv1.CampaignIntent_STANDARD:
		return campaign.CampaignIntentStandard
	case campaignv1.CampaignIntent_STARTER:
		return campaign.CampaignIntentStarter
	case campaignv1.CampaignIntent_SANDBOX:
		return campaign.CampaignIntentSandbox
	default:
		return campaign.CampaignIntentUnspecified
	}
}

func campaignIntentToProto(intent campaign.CampaignIntent) campaignv1.CampaignIntent {
	switch intent {
	case campaign.CampaignIntentStandard:
		return campaignv1.CampaignIntent_STANDARD
	case campaign.CampaignIntentStarter:
		return campaignv1.CampaignIntent_STARTER
	case campaign.CampaignIntentSandbox:
		return campaignv1.CampaignIntent_SANDBOX
	default:
		return campaignv1.CampaignIntent_CAMPAIGN_INTENT_UNSPECIFIED
	}
}

func campaignAccessPolicyFromProto(policy campaignv1.CampaignAccessPolicy) campaign.CampaignAccessPolicy {
	switch policy {
	case campaignv1.CampaignAccessPolicy_PRIVATE:
		return campaign.CampaignAccessPolicyPrivate
	case campaignv1.CampaignAccessPolicy_RESTRICTED:
		return campaign.CampaignAccessPolicyRestricted
	case campaignv1.CampaignAccessPolicy_PUBLIC:
		return campaign.CampaignAccessPolicyPublic
	default:
		return campaign.CampaignAccessPolicyUnspecified
	}
}

func campaignAccessPolicyToProto(policy campaign.CampaignAccessPolicy) campaignv1.CampaignAccessPolicy {
	switch policy {
	case campaign.CampaignAccessPolicyPrivate:
		return campaignv1.CampaignAccessPolicy_PRIVATE
	case campaign.CampaignAccessPolicyRestricted:
		return campaignv1.CampaignAccessPolicy_RESTRICTED
	case campaign.CampaignAccessPolicyPublic:
		return campaignv1.CampaignAccessPolicy_PUBLIC
	default:
		return campaignv1.CampaignAccessPolicy_CAMPAIGN_ACCESS_POLICY_UNSPECIFIED
	}
}

func gameSystemToProto(system commonv1.GameSystem) commonv1.GameSystem {
	return system
}

func gameSystemFromProto(system commonv1.GameSystem) commonv1.GameSystem {
	return system
}
