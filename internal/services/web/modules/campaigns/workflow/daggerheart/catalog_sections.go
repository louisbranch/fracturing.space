package daggerheart

import (
	"sort"
	"strings"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
)

// buildDomains normalizes domain entries and returns lookup data for card labeling.
func buildDomains(domains []campaignapp.CatalogDomain) ([]campaignapp.CatalogDomain, map[string]string) {
	result := make([]campaignapp.CatalogDomain, 0, len(domains))
	domainNameByID := make(map[string]string, len(domains))
	for _, domain := range domains {
		domainID := strings.TrimSpace(domain.ID)
		if domainID == "" {
			continue
		}
		domainName := strings.TrimSpace(domain.Name)
		if domainName == "" {
			domainName = domainID
		}
		domainNameByID[domainID] = domainName
		result = append(result, campaignapp.CatalogDomain{
			ID:           domainID,
			Name:         domainName,
			Illustration: domain.Illustration,
			Icon:         domain.Icon,
		})
	}
	sortByName(result, func(d campaignapp.CatalogDomain) string { return d.Name }, func(d campaignapp.CatalogDomain) string { return d.ID })
	return result, domainNameByID
}

// buildClasses normalizes classes and returns allowed-domain sets keyed by class ID.
func buildClasses(classes []campaignapp.CatalogClass) ([]campaignapp.CatalogClass, map[string]map[string]struct{}) {
	result := make([]campaignapp.CatalogClass, 0, len(classes))
	classDomainsByID := make(map[string]map[string]struct{}, len(classes))
	for _, class := range classes {
		classID := strings.TrimSpace(class.ID)
		if classID == "" {
			continue
		}
		className := strings.TrimSpace(class.Name)
		if className == "" {
			className = classID
		}
		domainIDs := make([]string, 0, len(class.DomainIDs))
		domains := make(map[string]struct{}, len(class.DomainIDs))
		for _, domainID := range class.DomainIDs {
			trimmedDomainID := strings.TrimSpace(domainID)
			if trimmedDomainID == "" {
				continue
			}
			domainIDs = append(domainIDs, trimmedDomainID)
			domains[trimmedDomainID] = struct{}{}
		}
		classDomainsByID[classID] = domains
		result = append(result, campaignapp.CatalogClass{
			ID:              classID,
			Name:            className,
			DomainIDs:       domainIDs,
			StartingHP:      class.StartingHP,
			StartingEvasion: class.StartingEvasion,
			HopeFeature: campaignapp.CatalogFeature{
				Name:        strings.TrimSpace(class.HopeFeature.Name),
				Description: strings.TrimSpace(class.HopeFeature.Description),
			},
			Features:     copyCatalogFeatures(class.Features),
			Illustration: class.Illustration,
			Icon:         class.Icon,
		})
	}
	sortByName(result, func(c campaignapp.CatalogClass) string { return c.Name }, func(c campaignapp.CatalogClass) string { return c.ID })
	return result, classDomainsByID
}

// buildSubclasses normalizes subclass entries for workflow rendering.
func buildSubclasses(subclasses []campaignapp.CatalogSubclass) []campaignapp.CatalogSubclass {
	result := make([]campaignapp.CatalogSubclass, 0, len(subclasses))
	for _, subclass := range subclasses {
		subclassID := strings.TrimSpace(subclass.ID)
		if subclassID == "" {
			continue
		}
		subclassName := strings.TrimSpace(subclass.Name)
		if subclassName == "" {
			subclassName = subclassID
		}
		result = append(result, campaignapp.CatalogSubclass{
			ID:             subclassID,
			Name:           subclassName,
			ClassID:        strings.TrimSpace(subclass.ClassID),
			SpellcastTrait: strings.TrimSpace(subclass.SpellcastTrait),
			Foundation:     copyCatalogFeatures(subclass.Foundation),
			Illustration:   subclass.Illustration,
		})
	}
	sortByName(result, func(s campaignapp.CatalogSubclass) string { return s.Name }, func(s campaignapp.CatalogSubclass) string { return s.ID })
	return result
}

// buildHeritages splits heritage entries into ancestry and community buckets.
func buildHeritages(heritages []campaignapp.CatalogHeritage) ([]campaignapp.CatalogHeritage, []campaignapp.CatalogHeritage) {
	ancestries := make([]campaignapp.CatalogHeritage, 0, len(heritages))
	communities := make([]campaignapp.CatalogHeritage, 0, len(heritages))
	for _, heritage := range heritages {
		heritageID := strings.TrimSpace(heritage.ID)
		if heritageID == "" {
			continue
		}
		heritageName := strings.TrimSpace(heritage.Name)
		if heritageName == "" {
			heritageName = heritageID
		}
		entry := campaignapp.CatalogHeritage{
			ID:           heritageID,
			Name:         heritageName,
			Kind:         strings.TrimSpace(heritage.Kind),
			Features:     copyCatalogFeatures(heritage.Features),
			Illustration: heritage.Illustration,
		}
		switch strings.ToLower(strings.TrimSpace(heritage.Kind)) {
		case "ancestry":
			ancestries = append(ancestries, entry)
		case "community":
			communities = append(communities, entry)
		}
	}
	sortByName(ancestries, func(h campaignapp.CatalogHeritage) string { return h.Name }, func(h campaignapp.CatalogHeritage) string { return h.ID })
	sortByName(communities, func(h campaignapp.CatalogHeritage) string { return h.Name }, func(h campaignapp.CatalogHeritage) string { return h.ID })
	return ancestries, communities
}

// buildWeapons keeps only tier-1 weapons and separates primary from secondary.
func buildWeapons(weapons []campaignapp.CatalogWeapon) ([]campaignapp.CatalogWeapon, []campaignapp.CatalogWeapon) {
	primary := make([]campaignapp.CatalogWeapon, 0, len(weapons))
	secondary := make([]campaignapp.CatalogWeapon, 0, len(weapons))
	for _, weapon := range weapons {
		weaponID := strings.TrimSpace(weapon.ID)
		if weaponID == "" || weapon.Tier != 1 {
			continue
		}
		weaponName := strings.TrimSpace(weapon.Name)
		if weaponName == "" {
			weaponName = weaponID
		}
		entry := campaignapp.CatalogWeapon{
			ID:           weaponID,
			Name:         weaponName,
			Category:     strings.TrimSpace(weapon.Category),
			Tier:         weapon.Tier,
			Trait:        strings.TrimSpace(weapon.Trait),
			Range:        strings.TrimSpace(weapon.Range),
			Damage:       strings.TrimSpace(weapon.Damage),
			Feature:      strings.TrimSpace(weapon.Feature),
			Illustration: weapon.Illustration,
		}
		switch strings.ToLower(strings.TrimSpace(weapon.Category)) {
		case "primary":
			primary = append(primary, entry)
		case "secondary":
			secondary = append(secondary, entry)
		}
	}
	sortByName(primary, func(w campaignapp.CatalogWeapon) string { return w.Name }, func(w campaignapp.CatalogWeapon) string { return w.ID })
	sortByName(secondary, func(w campaignapp.CatalogWeapon) string { return w.Name }, func(w campaignapp.CatalogWeapon) string { return w.ID })
	return primary, secondary
}

// buildArmor keeps only tier-1 armor entries for creation-time selection.
func buildArmor(armor []campaignapp.CatalogArmor) []campaignapp.CatalogArmor {
	result := make([]campaignapp.CatalogArmor, 0, len(armor))
	for _, item := range armor {
		armorID := strings.TrimSpace(item.ID)
		if armorID == "" || item.Tier != 1 {
			continue
		}
		armorName := strings.TrimSpace(item.Name)
		if armorName == "" {
			armorName = armorID
		}
		result = append(result, campaignapp.CatalogArmor{
			ID:             armorID,
			Name:           armorName,
			Tier:           item.Tier,
			ArmorScore:     item.ArmorScore,
			BaseThresholds: strings.TrimSpace(item.BaseThresholds),
			Feature:        strings.TrimSpace(item.Feature),
			Illustration:   item.Illustration,
		})
	}
	sortByName(result, func(a campaignapp.CatalogArmor) string { return a.Name }, func(a campaignapp.CatalogArmor) string { return a.ID })
	return result
}

// buildPotionItems filters items by potion allowlist and normalizes display fields.
func buildPotionItems(items []campaignapp.CatalogItem) []campaignapp.CatalogItem {
	result := make([]campaignapp.CatalogItem, 0, len(items))
	for _, item := range items {
		itemID := strings.TrimSpace(item.ID)
		if itemID == "" || !isAllowedPotionItemID(itemID) {
			continue
		}
		itemName := strings.TrimSpace(item.Name)
		if itemName == "" {
			itemName = itemID
		}
		result = append(result, campaignapp.CatalogItem{
			ID:           itemID,
			Name:         itemName,
			Description:  strings.TrimSpace(item.Description),
			Illustration: item.Illustration,
		})
	}
	sortByName(result, func(i campaignapp.CatalogItem) string { return i.Name }, func(i campaignapp.CatalogItem) string { return i.ID })
	return result
}

// buildDomainCards filters and normalizes level-1 cards for the selected class context.
func buildDomainCards(
	domainCards []campaignapp.CatalogDomainCard,
	selectedClassID string,
	classDomainsByID map[string]map[string]struct{},
	domainNameByID map[string]string,
) []campaignapp.CatalogDomainCard {
	result := make([]campaignapp.CatalogDomainCard, 0, len(domainCards))
	selectedClassID = strings.TrimSpace(selectedClassID)
	allowedDomains := classDomainsByID[selectedClassID]
	for _, domainCard := range domainCards {
		domainCardID := strings.TrimSpace(domainCard.ID)
		if domainCardID == "" || domainCard.Level != 1 {
			continue
		}
		domainID := strings.TrimSpace(domainCard.DomainID)
		if selectedClassID != "" && len(allowedDomains) > 0 {
			if _, ok := allowedDomains[domainID]; !ok {
				continue
			}
		}
		domainCardName := strings.TrimSpace(domainCard.Name)
		if domainCardName == "" {
			domainCardName = domainCardID
		}
		result = append(result, campaignapp.CatalogDomainCard{
			ID:           domainCardID,
			Name:         domainCardName,
			DomainID:     domainID,
			DomainName:   domainNameByID[domainID],
			Level:        domainCard.Level,
			Type:         strings.TrimSpace(domainCard.Type),
			RecallCost:   domainCard.RecallCost,
			FeatureText:  strings.TrimSpace(domainCard.FeatureText),
			Illustration: domainCard.Illustration,
		})
	}
	sort.SliceStable(result, func(i, j int) bool {
		leftLevel := result[i].Level
		rightLevel := result[j].Level
		if leftLevel == rightLevel {
			leftName := strings.ToLower(strings.TrimSpace(result[i].Name))
			rightName := strings.ToLower(strings.TrimSpace(result[j].Name))
			if leftName == rightName {
				return strings.TrimSpace(result[i].ID) < strings.TrimSpace(result[j].ID)
			}
			return leftName < rightName
		}
		return leftLevel < rightLevel
	})
	return result
}

// copyCatalogFeatures isolates slice copying for catalog model normalization.
func copyCatalogFeatures(features []campaignapp.CatalogFeature) []campaignapp.CatalogFeature {
	copied := make([]campaignapp.CatalogFeature, len(features))
	copy(copied, features)
	return copied
}
