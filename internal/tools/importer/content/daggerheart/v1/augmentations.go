package catalogimporter

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"

var subclassCreationRequirementsByID = map[string][]contentstore.DaggerheartSubclassCreationRequirement{
	"subclass.beastbound": {
		contentstore.DaggerheartSubclassCreationRequirementCompanionSheet,
	},
}
