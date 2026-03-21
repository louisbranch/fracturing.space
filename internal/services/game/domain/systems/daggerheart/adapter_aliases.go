package daggerheart

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/adapter"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

// --- Adapter type and constructor aliases ---

type Adapter = adapter.Adapter

// NewAdapter creates an Adapter with the root-level applyLevelUpToCharacterProfile
// injected. manifest.go and root tests call NewAdapter(store) unchanged.
var NewAdapter = func(store projectionstore.Store) *adapter.Adapter {
	return adapter.NewAdapter(store, applyLevelUpToCharacterProfile)
}

// --- Unexported aliases for root-package test files ---

var (
	companionProjectionStateFromProfile    = adapter.CompanionProjectionStateFromProfile
	classStateToProjection                 = adapter.ClassStateToProjection
	subclassStateToProjection              = adapter.SubclassStateToProjection
	companionStateToProjection             = adapter.CompanionStateToProjection
	activeBeastformToProjection            = adapter.ActiveBeastformToProjection
	classStateFromProjection               = adapter.ClassStateFromProjection
	subclassStateFromProjection            = adapter.SubclassStateFromProjection
	companionStateFromProjection           = adapter.CompanionStateFromProjection
	conditionStatesToProjection            = adapter.ConditionStatesToProjection
	statModifiersFromProjection            = adapter.StatModifiersFromProjection
	toProjectionAdversaryFeatureStates     = adapter.ToProjectionAdversaryFeatureStates
	toProjectionAdversaryPendingExperience = adapter.ToProjectionAdversaryPendingExperience
)
