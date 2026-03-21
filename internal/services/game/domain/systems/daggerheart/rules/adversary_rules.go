package rules

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

// AdversaryDefaultSpotlightCap is the standard one-spotlight-per-GM-turn cap
// for adversaries without a numeric Relentless rule.
const AdversaryDefaultSpotlightCap = 1

// AdversarySpotlightCap returns the allowed spotlight count for one adversary
// during the active GM turn.
func AdversarySpotlightCap(entry contentstore.DaggerheartAdversaryEntry) int {
	if entry.RelentlessRule != nil && entry.RelentlessRule.MaxSpotlightsPerGMTurn > 0 {
		return entry.RelentlessRule.MaxSpotlightsPerGMTurn
	}
	return AdversaryDefaultSpotlightCap
}

// AdversaryIsBloodied returns whether an adversary has marked at least half of
// its HP.
func AdversaryIsBloodied(hp, hpMax int) bool {
	if hpMax <= 0 {
		return false
	}
	return hp >= 0 && hp*2 <= hpMax
}

// AdversaryStandardAttack resolves the effective standard attack for one
// adversary instance, applying the Horde bloodied override when available.
func AdversaryStandardAttack(entry contentstore.DaggerheartAdversaryEntry, hp, hpMax int) contentstore.DaggerheartAdversaryAttack {
	if entry.HordeRule != nil && AdversaryIsBloodied(hp, hpMax) {
		return entry.HordeRule.BloodiedAttack
	}
	return entry.StandardAttack
}

// AdversaryIsMinion returns whether an adversary entry has an automated minion
// spillover rule.
func AdversaryIsMinion(entry contentstore.DaggerheartAdversaryEntry) bool {
	return entry.MinionRule != nil && entry.MinionRule.SpilloverDamageStep > 0
}

// AdversaryMinionSpilloverDefeats returns how many additional same-scene
// minions are defeated by spillover damage after the primary minion is hit.
func AdversaryMinionSpilloverDefeats(entry contentstore.DaggerheartAdversaryEntry, damageAmount int) int {
	if !AdversaryIsMinion(entry) || damageAmount <= 0 {
		return 0
	}
	return damageAmount / entry.MinionRule.SpilloverDamageStep
}
