package gateway

import campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"

// CharacterCreationReadDeps keeps creation workflow read dependencies explicit.
type CharacterCreationReadDeps struct {
	Character          CharacterReadClient
	DaggerheartContent DaggerheartContentClient
	DaggerheartAsset   DaggerheartAssetClient
}

// CharacterCreationMutationDeps keeps creation workflow mutation dependencies explicit.
type CharacterCreationMutationDeps struct {
	Character CharacterMutationClient
}

// characterCreationReadGateway maps character-creation workflow reads.
type characterCreationReadGateway struct {
	read         CharacterCreationReadDeps
	assetBaseURL string
}

// characterCreationMutationGateway maps character-creation workflow mutations.
type characterCreationMutationGateway struct {
	mutation CharacterCreationMutationDeps
}

// NewCharacterCreationReadGateway builds the character-creation read adapter
// from explicit dependencies.
func NewCharacterCreationReadGateway(deps CharacterCreationReadDeps, assetBaseURL string) campaignapp.CharacterCreationReadGateway {
	if deps.Character == nil || deps.DaggerheartContent == nil || deps.DaggerheartAsset == nil {
		return nil
	}
	return characterCreationReadGateway{read: deps, assetBaseURL: assetBaseURL}
}

// NewCharacterCreationMutationGateway builds the character-creation mutation
// adapter from explicit dependencies.
func NewCharacterCreationMutationGateway(deps CharacterCreationMutationDeps) campaignapp.CharacterCreationMutationGateway {
	if deps.Character == nil {
		return nil
	}
	return characterCreationMutationGateway{mutation: deps}
}
