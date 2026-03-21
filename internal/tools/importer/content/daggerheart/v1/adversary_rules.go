package catalogimporter

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

var (
	minionRulePattern     = regexp.MustCompile(`^Minion \((\d+)\)$`)
	hordeRulePattern      = regexp.MustCompile(`^Horde \((\d+d\d+(?:\+\d+)?)\)$`)
	relentlessRulePattern = regexp.MustCompile(`^Relentless \((\d+)\)$`)
	damagePattern         = regexp.MustCompile(`^(\d+)d(\d+)(?:\+(\d+))?$`)
)

func deriveAdversaryRecurringRules(item adversaryRecord) adversaryRecord {
	for _, feature := range item.Features {
		name := strings.TrimSpace(feature.Name)
		if matches := minionRulePattern.FindStringSubmatch(name); len(matches) == 2 {
			if step, err := strconv.Atoi(matches[1]); err == nil && step > 0 {
				item.MinionRule = &adversaryMinionRuleRecord{SpilloverDamageStep: step}
			}
			continue
		}
		if matches := hordeRulePattern.FindStringSubmatch(name); len(matches) == 2 {
			if attack, ok := parseHordeBloodiedAttack(item.StandardAttack, matches[1]); ok {
				item.HordeRule = &adversaryHordeRuleRecord{BloodiedAttack: attack}
			}
			continue
		}
		if matches := relentlessRulePattern.FindStringSubmatch(name); len(matches) == 2 {
			if count, err := strconv.Atoi(matches[1]); err == nil && count > 0 {
				item.RelentlessRule = &adversaryRelentlessRuleRecord{MaxSpotlightsPerGMTurn: count}
			}
		}
	}
	return item
}

func parseHordeBloodiedAttack(base adversaryAttackRecord, expression string) (adversaryAttackRecord, bool) {
	matches := damagePattern.FindStringSubmatch(strings.TrimSpace(expression))
	if len(matches) != 4 {
		return adversaryAttackRecord{}, false
	}
	count, err := strconv.Atoi(matches[1])
	if err != nil || count <= 0 {
		return adversaryAttackRecord{}, false
	}
	sides, err := strconv.Atoi(matches[2])
	if err != nil || sides <= 0 {
		return adversaryAttackRecord{}, false
	}
	bonus := 0
	if matches[3] != "" {
		bonus, err = strconv.Atoi(matches[3])
		if err != nil {
			return adversaryAttackRecord{}, false
		}
	}
	attack := base
	attack.DamageDice = []damageDieRecord{{Count: count, Sides: sides}}
	attack.DamageBonus = bonus
	return attack, true
}

func toStorageAdversaryMinionRule(rule *adversaryMinionRuleRecord) *contentstore.DaggerheartAdversaryMinionRule {
	if rule == nil || rule.SpilloverDamageStep <= 0 {
		return nil
	}
	return &contentstore.DaggerheartAdversaryMinionRule{SpilloverDamageStep: rule.SpilloverDamageStep}
}

func toStorageAdversaryHordeRule(rule *adversaryHordeRuleRecord) *contentstore.DaggerheartAdversaryHordeRule {
	if rule == nil {
		return nil
	}
	return &contentstore.DaggerheartAdversaryHordeRule{BloodiedAttack: toStorageAdversaryAttack(rule.BloodiedAttack)}
}

func toStorageAdversaryRelentlessRule(rule *adversaryRelentlessRuleRecord) *contentstore.DaggerheartAdversaryRelentlessRule {
	if rule == nil || rule.MaxSpotlightsPerGMTurn <= 0 {
		return nil
	}
	return &contentstore.DaggerheartAdversaryRelentlessRule{MaxSpotlightsPerGMTurn: rule.MaxSpotlightsPerGMTurn}
}
