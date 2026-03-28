package routepath

const (
	AppCampaignCharactersPattern             = CampaignsPrefix + "{campaignID}/characters"
	AppCampaignCharacterPattern              = CampaignsPrefix + "{campaignID}/characters/{characterID}"
	AppCampaignCharacterEditPattern          = CampaignsPrefix + "{campaignID}/characters/{characterID}/edit"
	AppCampaignCharacterOwnerPattern         = CampaignsPrefix + "{campaignID}/characters/{characterID}/owner"
	AppCampaignCharacterDeletePattern        = CampaignsPrefix + "{campaignID}/characters/{characterID}/delete"
	AppCampaignCharacterCreatePattern        = CampaignsPrefix + "{campaignID}/characters/create"
	AppCampaignCharacterCreationPattern      = CampaignsPrefix + "{campaignID}/characters/{characterID}/creation"
	AppCampaignCharacterCreationStepPattern  = CampaignsPrefix + "{campaignID}/characters/{characterID}/creation/step"
	AppCampaignCharacterCreationResetPattern = CampaignsPrefix + "{campaignID}/characters/{characterID}/creation/reset"
)

// AppCampaignCharacters returns the campaign characters route.
func AppCampaignCharacters(campaignID string) string {
	return AppCampaign(campaignID) + "/characters"
}

// AppCampaignCharacter returns the campaign character-detail route.
func AppCampaignCharacter(campaignID string, characterID string) string {
	return AppCampaignCharacters(campaignID) + "/" + escapeSegment(characterID)
}

// AppCampaignCharacterEdit returns the campaign character-edit route.
func AppCampaignCharacterEdit(campaignID string, characterID string) string {
	return AppCampaignCharacter(campaignID, characterID) + "/edit"
}

// AppCampaignCharacterOwner returns the character owner-set route.
func AppCampaignCharacterOwner(campaignID string, characterID string) string {
	return AppCampaignCharacter(campaignID, characterID) + "/owner"
}

// AppCampaignCharacterDelete returns the character-delete route.
func AppCampaignCharacterDelete(campaignID string, characterID string) string {
	return AppCampaignCharacter(campaignID, characterID) + "/delete"
}

// AppCampaignCharacterCreation returns the character creation page route.
func AppCampaignCharacterCreation(campaignID string, characterID string) string {
	return AppCampaignCharacter(campaignID, characterID) + "/creation"
}

// AppCampaignCharacterCreationStep returns the character creation step route.
func AppCampaignCharacterCreationStep(campaignID string, characterID string) string {
	return AppCampaignCharacter(campaignID, characterID) + "/creation/step"
}

// AppCampaignCharacterCreationReset returns the character creation reset route.
func AppCampaignCharacterCreationReset(campaignID string, characterID string) string {
	return AppCampaignCharacter(campaignID, characterID) + "/creation/reset"
}

// AppCampaignCharacterCreate returns the campaign character-create route.
func AppCampaignCharacterCreate(campaignID string) string {
	return AppCampaignCharacters(campaignID) + "/create"
}
