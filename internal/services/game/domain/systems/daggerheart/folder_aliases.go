package daggerheart

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/folder"

// --- Folder type and constructor aliases ---

type Folder = folder.Folder

// NewFolder creates a Folder with the root-level applyLevelUpToCharacterProfile
// injected. Root tests and module.go call NewFolder() unchanged.
var NewFolder = func() *folder.Folder {
	return folder.NewFolder(applyLevelUpToCharacterProfile)
}

// --- Unexported aliases for root-package test files ---

var (
	foldGMFearChanged    = folder.FoldGMFearChanged
	foldCountdownUpdated = folder.FoldCountdownUpdated
	foldEquipmentSwapped = folder.FoldEquipmentSwapped
)
