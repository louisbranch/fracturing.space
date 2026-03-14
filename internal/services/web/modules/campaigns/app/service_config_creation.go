package app

// CharacterCreationServiceConfig keeps character-creation dependencies explicit.
type CharacterCreationServiceConfig struct {
	Read     CharacterCreationReadGateway
	Mutation CharacterCreationMutationGateway
}
