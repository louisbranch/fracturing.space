package campaigns

import (
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
)

// Domain type aliases keep the root campaigns package as the HTTP/module
// surface while app contracts live in campaigns/app.
type (
	GameSystem                                 = campaignapp.GameSystem
	GmMode                                     = campaignapp.GmMode
	CharacterKind                              = campaignapp.CharacterKind
	CampaignSummary                            = campaignapp.CampaignSummary
	CampaignWorkspace                          = campaignapp.CampaignWorkspace
	CampaignParticipant                        = campaignapp.CampaignParticipant
	CampaignParticipantAccessOption            = campaignapp.CampaignParticipantAccessOption
	CampaignParticipantEditor                  = campaignapp.CampaignParticipantEditor
	CampaignCharacter                          = campaignapp.CampaignCharacter
	CampaignSession                            = campaignapp.CampaignSession
	CampaignSessionReadiness                   = campaignapp.CampaignSessionReadiness
	CampaignSessionReadinessBlocker            = campaignapp.CampaignSessionReadinessBlocker
	CampaignInvite                             = campaignapp.CampaignInvite
	CampaignCharacterCreationStep              = campaignapp.CampaignCharacterCreationStep
	CampaignCharacterCreationProgress          = campaignapp.CampaignCharacterCreationProgress
	CatalogClass                               = campaignapp.CatalogClass
	CatalogSubclass                            = campaignapp.CatalogSubclass
	CatalogHeritage                            = campaignapp.CatalogHeritage
	CatalogWeapon                              = campaignapp.CatalogWeapon
	CatalogArmor                               = campaignapp.CatalogArmor
	CatalogItem                                = campaignapp.CatalogItem
	CatalogDomainCard                          = campaignapp.CatalogDomainCard
	CampaignCharacterCreationCatalog           = campaignapp.CampaignCharacterCreationCatalog
	CampaignCharacterCreationProfile           = campaignapp.CampaignCharacterCreationProfile
	CampaignCharacterCreationStepInput         = campaignapp.CampaignCharacterCreationStepInput
	CampaignCharacterCreationStepClassSubclass = campaignapp.CampaignCharacterCreationStepClassSubclass
	CampaignCharacterCreationStepHeritage      = campaignapp.CampaignCharacterCreationStepHeritage
	CampaignCharacterCreationStepTraits        = campaignapp.CampaignCharacterCreationStepTraits
	CampaignCharacterCreationStepDetails       = campaignapp.CampaignCharacterCreationStepDetails
	CampaignCharacterCreationStepEquipment     = campaignapp.CampaignCharacterCreationStepEquipment
	CampaignCharacterCreationStepBackground    = campaignapp.CampaignCharacterCreationStepBackground
	CampaignCharacterCreationStepExperience    = campaignapp.CampaignCharacterCreationStepExperience
	CampaignCharacterCreationStepExperiences   = campaignapp.CampaignCharacterCreationStepExperiences
	CampaignCharacterCreationStepDomainCards   = campaignapp.CampaignCharacterCreationStepDomainCards
	CampaignCharacterCreationStepConnections   = campaignapp.CampaignCharacterCreationStepConnections
	CampaignCharacterCreation                  = campaignapp.CampaignCharacterCreation
	CreateCampaignInput                        = campaignapp.CreateCampaignInput
	UpdateCampaignInput                        = campaignapp.UpdateCampaignInput
	CreateCampaignResult                       = campaignapp.CreateCampaignResult
	StartSessionInput                          = campaignapp.StartSessionInput
	EndSessionInput                            = campaignapp.EndSessionInput
	CreateInviteInput                          = campaignapp.CreateInviteInput
	RevokeInviteInput                          = campaignapp.RevokeInviteInput
	CreateCharacterInput                       = campaignapp.CreateCharacterInput
	UpdateCharacterInput                       = campaignapp.UpdateCharacterInput
	CreateCharacterResult                      = campaignapp.CreateCharacterResult
	UpdateParticipantInput                     = campaignapp.UpdateParticipantInput
	CampaignGateway                            = campaignapp.CampaignGateway
)

const (
	GameSystemUnspecified = campaignapp.GameSystemUnspecified
	GameSystemDaggerheart = campaignapp.GameSystemDaggerheart

	GmModeUnspecified = campaignapp.GmModeUnspecified
	GmModeHuman       = campaignapp.GmModeHuman
	GmModeAI          = campaignapp.GmModeAI
	GmModeHybrid      = campaignapp.GmModeHybrid

	CharacterKindUnspecified = campaignapp.CharacterKindUnspecified
	CharacterKindPC          = campaignapp.CharacterKindPC
	CharacterKindNPC         = campaignapp.CharacterKindNPC
)
