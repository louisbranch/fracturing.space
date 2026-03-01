package app

// CharacterCreationWorkflow abstracts system-specific character creation.
type CharacterCreationWorkflow interface {
	// AssembleCatalog builds the system-specific catalog view from generic
	// gateway data (progress, catalog, profile).
	AssembleCatalog(
		progress CampaignCharacterCreationProgress,
		catalog CampaignCharacterCreationCatalog,
		profile CampaignCharacterCreationProfile,
	) CampaignCharacterCreation
}
