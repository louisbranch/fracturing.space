package campaign

// State captures the replayed campaign aggregate state used by deciders.
//
// New developers should read this as "campaign snapshot in-memory":
// it is derived from events and drives campaign-level policy decisions.
type State struct {
	// Created indicates whether campaign.create has been successfully applied.
	Created bool
	// Name is the campaign display name chosen by its creator.
	Name string
	// GameSystem is the chosen ruleset, used to route system-owned behavior.
	GameSystem string
	// GmMode captures who is allowed to lead game decisions.
	GmMode string
	// Status is the current lifecycle state that gates what operations are legal.
	Status Status
	// ThemePrompt stores optional campaign setup context for narrative features.
	ThemePrompt string
	// CoverAssetID stores the selected built-in campaign cover identifier.
	CoverAssetID string
}
