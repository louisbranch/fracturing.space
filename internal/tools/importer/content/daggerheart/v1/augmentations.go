package catalogimporter

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"

var subclassCreationRequirementsByID = map[string][]contentstore.DaggerheartSubclassCreationRequirement{
	"subclass.beastbound": {
		contentstore.DaggerheartSubclassCreationRequirementCompanionSheet,
	},
}
