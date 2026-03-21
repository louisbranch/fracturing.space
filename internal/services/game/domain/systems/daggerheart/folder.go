package daggerheart

import daggerheartfolder "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/folder"

// NewFolder creates the Daggerheart replay folder with the module-owned
// level-up applier wired in at the composition root.
func NewFolder() *daggerheartfolder.Folder {
	return daggerheartfolder.NewFolder(applyLevelUpToCharacterProfile)
}
