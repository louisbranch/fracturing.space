package render

import campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"

// CampaignCharacterCreationStepView carries one step status row for the
// character-creation workflow.
type CampaignCharacterCreationStepView = campaignworkflow.CharacterCreationStepView

// CampaignCreationClassFeatureView carries one feature paragraph for class,
// heritage, and subclass cards.
type CampaignCreationClassFeatureView = campaignworkflow.CreationClassFeatureView

// CampaignCreationDomainWatermarkView carries class-domain icon metadata for
// selectable cards.
type CampaignCreationDomainWatermarkView = campaignworkflow.CreationDomainWatermarkView

// CampaignCreationClassView carries one class option card.
type CampaignCreationClassView = campaignworkflow.CreationClassView

// CampaignCreationSubclassView carries one subclass option card.
type CampaignCreationSubclassView = campaignworkflow.CreationSubclassView

// CampaignCreationHeritageView carries one ancestry or community option card.
type CampaignCreationHeritageView = campaignworkflow.CreationHeritageView

// CampaignCreationWeaponView carries one weapon choice.
type CampaignCreationWeaponView = campaignworkflow.CreationWeaponView

// CampaignCreationArmorView carries one armor choice.
type CampaignCreationArmorView = campaignworkflow.CreationArmorView

// CampaignCreationItemView carries one item choice.
type CampaignCreationItemView = campaignworkflow.CreationItemView

// CampaignCreationExperienceView carries one freeform experience row.
type CampaignCreationExperienceView = campaignworkflow.CreationExperienceView

// CampaignCreationDomainCardView carries one domain-card choice.
type CampaignCreationDomainCardView = campaignworkflow.CreationDomainCardView

// CampaignCharacterCreationView carries the full render contract for one
// character-creation workflow state.
type CampaignCharacterCreationView = campaignworkflow.CharacterCreationView
