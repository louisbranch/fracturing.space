package app

// GameSystem represents a campaign game system as a domain-native value.
// Gateway implementations map these to proto enums at the transport boundary.
type GameSystem string

const (
	GameSystemUnspecified GameSystem = ""
	GameSystemDaggerheart GameSystem = "daggerheart"
)

// GmMode represents a campaign GM mode as a domain-native value.
type GmMode string

const (
	GmModeUnspecified GmMode = ""
	GmModeHuman       GmMode = "human"
	GmModeAI          GmMode = "ai"
	GmModeHybrid      GmMode = "hybrid"
)

// CharacterKind represents the kind of character as a domain-native value.
type CharacterKind string

const (
	CharacterKindUnspecified CharacterKind = ""
	CharacterKindPC          CharacterKind = "pc"
	CharacterKindNPC         CharacterKind = "npc"
)
