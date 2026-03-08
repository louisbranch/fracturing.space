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
	// Locale stores the campaign language preference as a BCP-47 tag.
	Locale string
	// GameSystem is the chosen ruleset, used to route system-owned behavior.
	GameSystem GameSystem
	// GmMode captures who is allowed to lead game decisions.
	GmMode GmMode
	// Status is the current lifecycle state that gates what operations are legal.
	Status Status
	// ThemePrompt stores optional campaign setup context for narrative features.
	ThemePrompt string
	// CoverAssetID stores the selected built-in campaign cover identifier.
	CoverAssetID string
	// CoverSetID stores the selected built-in campaign cover set identifier.
	CoverSetID string
	// AIAgentID stores the bound AI service opaque agent identifier.
	AIAgentID string
	// AIAuthEpoch stores the current AI authorization epoch for session grants.
	AIAuthEpoch uint64
}
