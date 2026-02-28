package campaigns

import (
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"
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
	CampaignCharacter                          = campaignapp.CampaignCharacter
	CampaignSession                            = campaignapp.CampaignSession
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
	CreateCampaignResult                       = campaignapp.CreateCampaignResult
	CreateCharacterInput                       = campaignapp.CreateCharacterInput
	CreateCharacterResult                      = campaignapp.CreateCharacterResult
	CharacterCreationWorkflow                  = campaignapp.CharacterCreationWorkflow
	CampaignGateway                            = campaignapp.CampaignGateway

	campaignAuthorizationDecision = campaignapp.AuthorizationDecision
	campaignAuthorizationTarget   = campaignapp.AuthorizationTarget
	campaignAuthorizationCheck    = campaignapp.AuthorizationCheck
	campaignAuthorizationAction   = campaignapp.AuthorizationAction
	campaignAuthorizationResource = campaignapp.AuthorizationResource

	CampaignClient           = campaigngateway.CampaignClient
	ParticipantClient        = campaigngateway.ParticipantClient
	CharacterClient          = campaigngateway.CharacterClient
	DaggerheartContentClient = campaigngateway.DaggerheartContentClient
	SessionClient            = campaigngateway.SessionClient
	InviteClient             = campaigngateway.InviteClient
	AuthorizationClient      = campaigngateway.AuthorizationClient
	GRPCGatewayDeps          = campaigngateway.GRPCGatewayDeps
	grpcGateway              = campaigngateway.GRPCGateway
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

	campaignAuthzActionManage = campaignapp.AuthorizationActionManage
	campaignAuthzActionMutate = campaignapp.AuthorizationActionMutate

	campaignAuthzResourceSession     = campaignapp.AuthorizationResourceSession
	campaignAuthzResourceParticipant = campaignapp.AuthorizationResourceParticipant
	campaignAuthzResourceCharacter   = campaignapp.AuthorizationResourceCharacter
	campaignAuthzResourceInvite      = campaignapp.AuthorizationResourceInvite
)

// NewGRPCGateway returns the production campaigns gateway.
func NewGRPCGateway(deps GRPCGatewayDeps) CampaignGateway {
	return campaigngateway.NewGRPCGateway(deps)
}

func mapCampaignCharacterCreationStepToProto(step *CampaignCharacterCreationStepInput) (*daggerheartv1.DaggerheartCreationStepInput, error) {
	return campaigngateway.MapCampaignCharacterCreationStepToProto(step)
}
